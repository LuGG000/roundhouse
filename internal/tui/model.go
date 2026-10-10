// Package tui is the interactive session list.
package tui

import (
	"errors"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/LuGG000/roundhouse/internal/index"
	"github.com/LuGG000/roundhouse/internal/starter"
)

type screen int

const (
	sessionsScreen screen = iota
	foldersScreen
)

// Chrome lines around the list: title, filter and rule above; status and help below.
const (
	headerLines = 3
	footerLines = 2
)

// Favorites remembers starred chats.
type Favorites interface {
	Has(id string) bool
	Toggle(id string) error
	HasFolder(dir string) bool
	ToggleFolder(dir string) error
}

// Scanner loads the sessions to show.
type Scanner func() ([]index.Session, error)

type loadedMsg struct {
	sessions []index.Session
	err      error
}

type startedMsg struct{ err error }

// Model is the bubbletea model of the hub.
type Model struct {
	scan    Scanner
	starter starter.Starter
	favs    Favorites

	sessions []index.Session
	items    []item
	screen   screen
	grouped  bool

	filter    textinput.Model
	filtering bool

	cursor, top   int
	width, height int
	status        string
	loading       bool
	now           func() time.Time
}

// New returns a model that loads its sessions on start.
func New(scan Scanner, s starter.Starter, favs Favorites) Model {
	f := textinput.New()
	f.Prompt = "/ "
	f.Placeholder = "type to filter"
	return Model{scan: scan, starter: s, favs: favs, filter: f, grouped: true, loading: true, now: time.Now}
}

func (m Model) Init() tea.Cmd { return m.load() }

func (m Model) load() tea.Cmd {
	return func() tea.Msg {
		s, err := m.scan()
		return loadedMsg{sessions: s, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clamp()
	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = "scan failed: " + msg.err.Error()
			return m, nil
		}
		m.sessions = msg.sessions
		m.rebuild()
	case startedMsg:
		// A non-zero exit of claude is not an error; only a failure to
		// start the command is.
		var exit *exec.ExitError
		if msg.err != nil && !errors.As(msg.err, &exit) {
			m.status = "could not start: " + msg.err.Error()
		}
		m.screen = sessionsScreen
		m.rebuild()
		m.loading = true
		return m, m.load()
	case tea.KeyMsg:
		return m.key(msg)
	case tea.MouseMsg:
		return m.mouse(msg)
	}
	return m, nil
}

func (m Model) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		switch msg.String() {
		case "enter":
			m.filtering = false
			m.filter.Blur()
		case "esc":
			m.filtering = false
			m.filter.Blur()
			m.filter.SetValue("")
			m.rebuild()
		default:
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.rebuild()
			return m, cmd
		}
		return m, nil
	}

	m.status = ""
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		switch {
		case m.screen == foldersScreen:
			m.screen = sessionsScreen
			m.rebuild()
		case m.filter.Value() != "":
			m.filter.SetValue("")
			m.rebuild()
		default:
			return m, tea.Quit
		}
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-m.bodyHeight() / 2)
	case "pgdown":
		m.move(m.bodyHeight() / 2)
	case "home":
		m.cursor = 0
		m.move(0)
	case "end":
		m.cursor = len(m.items) - 1
		m.move(0)
	case "/":
		m.filtering = true
		return m, m.filter.Focus()
	case "g":
		if m.screen == sessionsScreen {
			m.grouped = !m.grouped
			m.rebuild()
		}
	case "n":
		if m.screen == sessionsScreen {
			m.screen = foldersScreen
			m.rebuild()
		}
	case "f":
		m.toggleFavorite(false)
	case "F":
		m.toggleFavorite(true)
	case "r":
		m.loading = true
		return m, m.load()
	case "enter":
		return m.open()
	}
	return m, nil
}

func (m Model) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		m.move(-1)
	case msg.Button == tea.MouseButtonWheelDown:
		m.move(1)
	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		i := m.itemAt(msg.Y)
		if i < 0 || !m.items[i].selectable() {
			return m, nil
		}
		if i == m.cursor {
			return m.open()
		}
		m.cursor = i
	}
	return m, nil
}

// open starts the selected session, or a new session in the selected folder.
func (m Model) open() (tea.Model, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}
	it := m.items[m.cursor]
	var (
		cmd *exec.Cmd
		err error
	)
	if m.screen == foldersScreen {
		cmd, err = m.starter.NewCommand(starter.Target{Cwd: it.folder})
	} else {
		s := it.session
		cmd, err = m.starter.ResumeCommand(starter.Target{ID: s.ID, Path: s.Path, Cwd: s.Cwd})
	}
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return startedMsg{err} })
}

// toggleFavorite stars or unstars the selected chat, or its folder when
// folder is set. On the folder screen the selected folder is meant either
// way. The selection stays on the same entry.
func (m *Model) toggleFavorite(folder bool) {
	if len(m.items) == 0 {
		return
	}
	cur := m.items[m.cursor]
	folder = folder || m.screen == foldersScreen

	var err error
	if folder {
		dir := cur.folder
		if cur.session != nil {
			dir = cur.session.Cwd
		}
		err = m.favs.ToggleFolder(dir)
	} else {
		err = m.favs.Toggle(cur.session.ID)
	}
	if err != nil {
		m.status = "could not save favorites: " + err.Error()
		return
	}

	m.rebuild()
	for i, it := range m.items {
		if it.selectable() && it.sameEntry(cur) {
			m.cursor = i
			m.move(0)
			break
		}
	}
}

// rebuild recomputes the lines after the data, filter or view changed.
func (m *Model) rebuild() {
	q := m.filter.Value()
	if m.screen == foldersScreen {
		m.items = folderItems(m.sessions, q, m.favs)
	} else {
		m.items = sessionItems(m.sessions, q, m.grouped, m.favs)
	}
	m.cursor, m.top = 0, 0
	m.move(0)
}

// move shifts the cursor by delta lines and lands on a selectable line.
func (m *Model) move(delta int) {
	if len(m.items) == 0 {
		m.cursor = 0
		return
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	m.cursor = max(0, min(m.cursor+delta, len(m.items)-1))
	for i := m.cursor; i >= 0 && i < len(m.items); i += step {
		if m.items[i].selectable() {
			m.cursor = i
			m.clamp()
			return
		}
	}
	// Nothing selectable in that direction: search the other way.
	for i := m.cursor; i >= 0 && i < len(m.items); i -= step {
		if m.items[i].selectable() {
			m.cursor = i
			break
		}
	}
	m.clamp()
}

// clamp scrolls so that the cursor item is fully visible.
func (m *Model) clamp() {
	if len(m.items) == 0 {
		m.top = 0
		return
	}
	if m.cursor < m.top {
		m.top = m.cursor
		// Keep the folder header of the first visible chat in view.
		if m.top > 0 && m.items[m.top-1].header {
			m.top--
		}
	}
	for m.top < m.cursor && m.linesThrough(m.top, m.cursor) > m.bodyHeight() {
		m.top++
	}
}

// heightOf is the number of terminal lines item i takes. A section header is
// one line, its footer a closing line and a gap. A chat is a title and a
// detail line, and the selected one grows into a bordered card.
func (m Model) heightOf(i int) int {
	switch it := m.items[i]; {
	case it.header:
		return 1
	case it.footer:
		return 2
	case i == m.cursor:
		return 4
	default:
		return 2
	}
}

// linesThrough is the height of items from..to, both included.
func (m Model) linesThrough(from, to int) int {
	n := 0
	for i := from; i <= to; i++ {
		n += m.heightOf(i)
	}
	return n
}

// itemAt returns the item drawn on screen line y, or -1.
func (m Model) itemAt(y int) int {
	line := y - headerLines
	if line < 0 || line >= m.bodyHeight() {
		return -1
	}
	for i := m.top; i < len(m.items); i++ {
		if line < m.heightOf(i) {
			return i
		}
		line -= m.heightOf(i)
	}
	return -1
}

func (m Model) bodyHeight() int {
	return max(1, m.height-headerLines-footerLines)
}
