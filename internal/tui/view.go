package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	cursorStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

const (
	ageWidth    = 9
	folderWidth = 26
)

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	var b strings.Builder

	b.WriteString(titleStyle.Render("roundhouse") + dimStyle.Render("  "+m.subtitle()) + "\n")
	b.WriteString(m.filterLine() + "\n")

	body := m.bodyHeight()
	lines := make([]string, 0, body)
	for i := m.top; i < len(m.items) && len(lines) < body; i++ {
		lines = append(lines, m.line(i))
	}
	if len(m.items) == 0 {
		lines = append(lines, dimStyle.Render(m.emptyText()))
	}
	for len(lines) < body {
		lines = append(lines, "")
	}
	b.WriteString(strings.Join(lines, "\n") + "\n")

	b.WriteString(m.statusLine() + "\n")
	b.WriteString(dimStyle.Render(ansi.Truncate(m.help(), m.width, "…")))
	return b.String()
}

func (m Model) subtitle() string {
	if m.screen == foldersScreen {
		return "new chat: pick a folder"
	}
	if m.grouped {
		return "chats by folder"
	}
	return "recent chats"
}

func (m Model) filterLine() string {
	if m.filtering || m.filter.Value() != "" {
		return m.filter.View()
	}
	return dimStyle.Render("press / to filter")
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
		return errStyle.Render(ansi.Truncate(m.status, m.width, "…"))
	case m.loading:
		return dimStyle.Render("loading…")
	default:
		return ""
	}
}

func (m Model) help() string {
	switch {
	case m.filtering:
		return "enter keep filter · esc clear"
	case m.screen == foldersScreen:
		return "enter/click start chat here · / filter · esc back"
	default:
		return "enter/click open · n new chat · g group · / filter · r reload · q quit"
	}
}

// line renders item i, padded to the window width.
func (m Model) line(i int) string {
	it := m.items[i]
	selected := i == m.cursor

	var text string
	switch {
	case it.header:
		return headerStyle.Render(ansi.Truncate(shortPath(it.folder, m.width), m.width, "…"))
	case m.screen == foldersScreen:
		text = m.folderText(it)
	default:
		text = m.sessionText(it)
	}
	if selected {
		return cursorStyle.Render(pad(text, m.width))
	}
	return text
}

func (m Model) sessionText(it item) string {
	age := ago(it.modTime, m.now())
	indent := "  "
	if m.grouped {
		indent = "    "
	}
	cols := []string{pad(age, ageWidth)}
	if !m.grouped {
		cols = append(cols, pad(shortPath(it.session.Cwd, folderWidth), folderWidth))
	}
	rest := strings.Join(cols, "  ")
	titleWidth := max(10, m.width-len(indent)-ansi.StringWidth(rest)-2)
	title := ansi.Truncate(oneLine(it.session.Title), titleWidth, "…")
	return indent + pad(title, titleWidth) + "  " + rest
}

func (m Model) folderText(it item) string {
	count := pad(plural(it.count, "chat"), ageWidth)
	width := max(10, m.width-4-ageWidth-2)
	return "  " + pad(ansi.Truncate(shortPath(it.folder, width), width, "…"), width) + "  " + count
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

// pad fills s with spaces up to width cells.
func pad(s string, width int) string {
	if w := ansi.StringWidth(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}
