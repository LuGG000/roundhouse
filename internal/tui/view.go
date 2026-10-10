package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	accent   = lipgloss.AdaptiveColor{Light: "#B5502D", Dark: "#D97757"}
	selectBg = lipgloss.AdaptiveColor{Light: "#F3E3DB", Dark: "#3A2A24"}

	brandStyle  = lipgloss.NewStyle().Bold(true).Foreground(accent)
	titleStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	groupStyle  = lipgloss.NewStyle().Bold(true).Foreground(accent)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	keyStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	selectStyle = lipgloss.NewStyle().Background(selectBg)
)

// margin is the space on the left of every line, holding the selection bar.
const margin = 2

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	lines := []string{m.titleLine(), m.filterLine(), dimStyle.Render(strings.Repeat("─", m.width))}

	body := make([]string, 0, m.bodyHeight())
	for i := m.top; i < len(m.items) && len(body) < m.bodyHeight(); i++ {
		body = append(body, m.render(i)...)
	}
	if len(m.items) == 0 {
		body = append(body, "", dimStyle.Render(strings.Repeat(" ", margin)+m.emptyText()))
	}
	if len(body) > m.bodyHeight() {
		body = body[:m.bodyHeight()]
	}
	for len(body) < m.bodyHeight() {
		body = append(body, "")
	}
	lines = append(lines, body...)
	lines = append(lines, m.statusLine(), m.helpLine())
	return strings.Join(lines, "\n")
}

func (m Model) titleLine() string {
	left := brandStyle.Render(" roundhouse") + dimStyle.Render("  "+m.subtitle())
	right := dimStyle.Render(m.counter() + " ")
	gap := max(1, m.width-ansi.StringWidth(left)-ansi.StringWidth(right))
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) counter() string {
	n := 0
	for _, it := range m.items {
		if !it.header {
			n++
		}
	}
	if m.screen == foldersScreen {
		return plural(n, "folder")
	}
	return plural(n, "chat")
}

func (m Model) subtitle() string {
	switch {
	case m.screen == foldersScreen:
		return "new chat · pick a folder"
	case m.grouped:
		return "chats by folder"
	default:
		return "recent chats"
	}
}

func (m Model) filterLine() string {
	if m.filtering || m.filter.Value() != "" {
		return strings.Repeat(" ", margin) + m.filter.View()
	}
	return dimStyle.Render(strings.Repeat(" ", margin) + "/ to filter")
}

func (m Model) emptyText() string {
	switch {
	case m.loading:
		return "loading…"
	case m.filter.Value() != "":
		return "nothing matches the filter"
	default:
		return "no chats found"
	}
}

func (m Model) statusLine() string {
	switch {
	case m.status != "":
		return errStyle.Render(ansi.Truncate(strings.Repeat(" ", margin)+m.status, m.width, "…"))
	case m.loading:
		return dimStyle.Render(strings.Repeat(" ", margin) + "loading…")
	default:
		return ""
	}
}

func (m Model) helpLine() string {
	var keys [][2]string
	switch {
	case m.filtering:
		keys = [][2]string{{"enter", "keep filter"}, {"esc", "clear"}}
	case m.screen == foldersScreen:
		keys = [][2]string{{"enter", "start chat here"}, {"/", "filter"}, {"esc", "back"}}
	default:
		keys = [][2]string{{"enter", "open"}, {"n", "new chat"}, {"g", "group"}, {"/", "filter"}, {"r", "reload"}, {"q", "quit"}}
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = keyStyle.Render(k[0]) + " " + dimStyle.Render(k[1])
	}
	return ansi.Truncate(strings.Repeat(" ", margin)+strings.Join(parts, dimStyle.Render("  ·  ")), m.width, "…")
}

// render draws item i as one or two lines.
func (m Model) render(i int) []string {
	it := m.items[i]
	selected := i == m.cursor
	w := m.width - margin - 1

	switch {
	case it.header:
		path := ansi.Truncate(shortPath(it.folder, w), w, "…")
		return []string{"", strings.Repeat(" ", margin) + groupStyle.Render(path)}
	case it.session == nil:
		return m.row(selected, w, shortPath(it.folder, w), ago(it.modTime, m.now()), plural(it.count, "chat"))
	default:
		return m.row(selected, w, oneLine(it.session.Title), ago(it.modTime, m.now()), m.detail(it, w))
	}
}

// detail is the dim second line of a chat.
func (m Model) detail(it item, w int) string {
	if !m.grouped {
		return shortPath(it.session.Cwd, w)
	}
	if p := oneLine(it.session.LastPrompt); p != oneLine(it.session.Title) {
		return p
	}
	return ""
}

// row lays out a title with a right-aligned age and a dim detail line below.
func (m Model) row(selected bool, w int, title, age, detail string) []string {
	base := lipgloss.NewStyle()
	bar := strings.Repeat(" ", margin)
	if selected {
		base = selectStyle
		bar = brandStyle.Background(selectBg).Render("▌") + selectStyle.Render(" ")
	}

	titleW := max(4, w-ansi.StringWidth(age)-2)
	title = ansi.Truncate(title, titleW, "…")
	gap := max(1, w-ansi.StringWidth(title)-ansi.StringWidth(age))
	first := bar +
		base.Inherit(titleStyle).Render(title) +
		base.Render(strings.Repeat(" ", gap)) +
		base.Inherit(dimStyle).Render(age) +
		base.Render(" ")

	detail = ansi.Truncate(detail, w, "…")
	second := bar +
		base.Inherit(dimStyle).Render(detail) +
		base.Render(strings.Repeat(" ", max(0, w-ansi.StringWidth(detail))+1))
	return []string{first, second}
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
