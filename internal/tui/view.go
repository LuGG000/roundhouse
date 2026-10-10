package tui

import (
	"fmt"
	"strings"
	"time"

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
	accentStyle = lipgloss.NewStyle().Foreground(accent)
	freshStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	keyStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent)
)

// margin is the space on the left of unboxed lines.
const margin = 2

// freshWithin is how recent a chat is to get its age highlighted.
const freshWithin = time.Hour

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
	right := dimStyle.Render(m.counter() + " ")
	left := ansi.Truncate(brandStyle.Render(" roundhouse")+dimStyle.Render("  "+m.subtitle()), max(1, m.width-ansi.StringWidth(right)-1), "…")
	gap := max(1, m.width-ansi.StringWidth(left)-ansi.StringWidth(right))
	return ansi.Truncate(left+strings.Repeat(" ", gap)+right, m.width, "")
}

func (m Model) counter() string {
	n := 0
	seen := map[string]bool{}
	for _, it := range m.items {
		switch {
		case it.session != nil && !seen[it.session.ID]:
			seen[it.session.ID] = true
			n++
		case it.session == nil && it.selectable():
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
		keys = [][2]string{{"enter", "open"}, {"n", "new chat"}, {"f", "favorite"}, {"g", "group"}, {"/", "filter"}, {"r", "reload"}, {"q", "quit"}}
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = keyStyle.Render(k[0]) + " " + dimStyle.Render(k[1])
	}
	return ansi.Truncate(strings.Repeat(" ", margin)+strings.Join(parts, dimStyle.Render("  ·  ")), m.width, "…")
}

// boxed reports whether chats are drawn inside a bordered section per folder.
func (m Model) boxed() bool { return m.screen == sessionsScreen && m.grouped }

// contentWidth is the width available to a chat inside its section or margin.
func (m Model) contentWidth() int {
	if m.boxed() {
		return m.width - 4
	}
	return m.width - margin - 1
}

// render draws item i.
func (m Model) render(i int) []string {
	it := m.items[i]
	switch {
	case it.header:
		return []string{m.sectionTop(it)}
	case it.footer:
		return []string{dimStyle.Render("╰" + strings.Repeat("─", m.width-2) + "╯"), ""}
	}

	cw := m.contentWidth()
	var lines []string
	if it.session == nil {
		lines = m.card(i == m.cursor, cw, shortPath(it.folder, cw), ago(it.modTime, m.now()), it.modTime, plural(it.count, "chat"), false)
	} else {
		s := it.session
		lines = m.card(i == m.cursor, cw, oneLine(s.Title), ago(it.modTime, m.now()), it.modTime, m.detail(it, cw), m.favs.Has(s.ID))
	}

	edge := strings.Repeat(" ", margin)
	if m.boxed() {
		edge = dimStyle.Render("│") + " "
	}
	for j, l := range lines {
		if m.boxed() {
			l += " " + dimStyle.Render("│")
		}
		lines[j] = edge + l
	}
	return lines
}

// sectionTop is the top border of a folder or favorites section.
func (m Model) sectionTop(it item) string {
	label := "★ Favorites"
	if !it.favorites {
		label = shortPath(it.folder, m.width/2)
	}
	count := plural(it.count, "chat")
	label = ansi.Truncate(label, max(1, m.width-ansi.StringWidth(count)-9), "…")
	fill := max(1, m.width-ansi.StringWidth(label)-ansi.StringWidth(count)-8)
	return dimStyle.Render("╭─ ") + groupStyle.Render(label) + dimStyle.Render(" "+strings.Repeat("─", fill)+" "+count+" ─╮")
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

// card lays out one entry as a title with a right-aligned age and a dim
// detail line. The selected entry gets a bordered, tinted card. Every line is
// exactly w cells wide.
func (m Model) card(selected bool, w int, title, age string, mod time.Time, detail string, fav bool) []string {
	tint := func(s lipgloss.Style) lipgloss.Style {
		if selected {
			return s.Background(selectBg)
		}
		return s
	}
	ageStyle := dimStyle
	if m.now().Sub(mod) < freshWithin {
		ageStyle = freshStyle
	}
	mark := "○"
	switch {
	case fav:
		mark = "★"
	case selected:
		mark = "●"
	}
	markStyle := dimStyle
	if fav || selected {
		markStyle = accentStyle
	}

	inner := w
	if selected {
		inner = w - 2
	}
	titleW := max(4, inner-5-ansi.StringWidth(age))
	first := tint(lipgloss.NewStyle()).Render(" ") + tint(markStyle).Render(mark) + tint(lipgloss.NewStyle()).Render(" ") +
		cell(tint(titleStyle), title, titleW) +
		tint(lipgloss.NewStyle()).Render(" ") + tint(ageStyle).Render(age) + tint(lipgloss.NewStyle()).Render(" ")
	second := cell(tint(dimStyle), "   "+detail, inner-1) + tint(lipgloss.NewStyle()).Render(" ")

	if !selected {
		return []string{first, second}
	}
	edge := accentStyle.Render("│")
	return []string{
		accentStyle.Render("╭" + strings.Repeat("─", inner) + "╮"),
		edge + first + edge,
		edge + second + edge,
		accentStyle.Render("╰" + strings.Repeat("─", inner) + "╯"),
	}
}

// cell renders text in style st, cut or padded to exactly w cells.
func cell(st lipgloss.Style, text string, w int) string {
	text = ansi.Truncate(text, w, "…")
	return st.Render(text + strings.Repeat(" ", max(0, w-ansi.StringWidth(text))))
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
