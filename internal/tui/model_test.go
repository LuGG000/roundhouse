package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LuGG000/roundhouse/internal/index"
	"github.com/LuGG000/roundhouse/internal/starter"
	"github.com/LuGG000/roundhouse/internal/transcript"
)

var base = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

func session(id, cwd, title string, age time.Duration) index.Session {
	return index.Session{
		Summary: transcript.Summary{ID: id, Path: "/p/" + id + ".jsonl", Cwd: cwd, Title: title},
		ModTime: base.Add(-age),
	}
}

func testSessions() []index.Session {
	return []index.Session{
		session("1", "/work/a", "fix build", time.Minute),
		session("2", "/work/b", "write docs", time.Hour),
		session("3", "/work/a", "add tests", 2*time.Hour),
	}
}

func loaded(t *testing.T, sessions []index.Session) Model {
	t.Helper()
	m := New(func() ([]index.Session, error) { return sessions, nil }, starter.Builtins()[0])
	m.now = func() time.Time { return base }
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	next, _ = next.(Model).Update(loadedMsg{sessions: sessions})
	return next.(Model)
}

func press(m Model, key string) Model {
	var msg tea.KeyMsg
	switch key {
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestGroupedByFolderKeepsNewestFolderFirst(t *testing.T) {
	m := loaded(t, testSessions())
	var got []string
	for _, it := range m.items {
		if it.header {
			got = append(got, "H:"+it.folder)
		} else {
			got = append(got, it.session.ID)
		}
	}
	want := "H:/work/a 1 3 H:/work/b 2"
	if strings.Join(got, " ") != want {
		t.Errorf("items = %v, want %s", got, want)
	}
}

func TestCursorSkipsHeaders(t *testing.T) {
	m := loaded(t, testSessions())
	if m.items[m.cursor].session.ID != "1" {
		t.Fatalf("cursor starts on %+v", m.items[m.cursor])
	}
	m = press(m, "down")
	m = press(m, "down")
	if m.items[m.cursor].session.ID != "2" {
		t.Errorf("cursor on %+v, want session 2 after skipping the header", m.items[m.cursor])
	}
}

func TestFilter(t *testing.T) {
	m := loaded(t, testSessions())
	m = press(m, "/")
	for _, r := range "docs" {
		m = press(m, string(r))
	}
	if len(m.items) != 2 || m.items[1].session.ID != "2" {
		t.Errorf("items = %+v, want header and session 2", m.items)
	}
	m = press(m, "esc")
	if len(m.items) != 5 {
		t.Errorf("esc should clear the filter, got %d items", len(m.items))
	}
}

func TestToggleGrouping(t *testing.T) {
	m := press(loaded(t, testSessions()), "g")
	if len(m.items) != 3 || m.items[0].header {
		t.Errorf("flat view should list only sessions, got %+v", m.items)
	}
}

func TestFolderScreen(t *testing.T) {
	m := press(loaded(t, testSessions()), "n")
	if m.screen != foldersScreen || len(m.items) != 2 || m.items[0].count != 2 {
		t.Fatalf("folder screen wrong: %+v", m.items)
	}
	m = press(m, "esc")
	if m.screen != sessionsScreen {
		t.Error("esc should go back to the chats")
	}
}

func TestOpenReportsMissingFolder(t *testing.T) {
	m := loaded(t, testSessions()) // /work/a does not exist in the test environment
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("no process should start for a missing folder")
	}
	if !strings.Contains(next.(Model).status, "does not exist") {
		t.Errorf("status = %q", next.(Model).status)
	}
}

func TestClickSelectsThenOpens(t *testing.T) {
	m := loaded(t, testSessions())
	click := func(y int) (Model, tea.Cmd) {
		next, cmd := m.Update(tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, Y: y})
		return next.(Model), cmd
	}
	// Every item is two lines: header a, chat 1, chat 3, header b, chat 2.
	m, cmd := click(headerLines + 4)
	if cmd != nil || m.items[m.cursor].session.ID != "3" {
		t.Fatalf("first click should only select chat 3, got cursor %+v", m.items[m.cursor])
	}
	m, _ = click(headerLines + 0) // a folder header: ignored
	if m.items[m.cursor].session.ID != "3" {
		t.Error("clicking a header must not move the cursor")
	}
	m, _ = click(headerLines + 5) // second line of the same chat

	if m.status == "" {
		t.Error("second click on the selected row should try to open it")
	}
}

func TestViewFitsWindow(t *testing.T) {
	m := loaded(t, testSessions())
	view := m.View()
	if got := strings.Count(view, "\n") + 1; got != 20 {
		t.Errorf("view has %d lines, want 20", got)
	}
	if !strings.Contains(view, "fix build") || !strings.Contains(view, "1m ago") {
		t.Errorf("view misses content:\n%s", view)
	}
}

func TestAgo(t *testing.T) {
	for d, want := range map[time.Duration]string{
		10 * time.Second: "just now",
		5 * time.Minute:  "5m ago",
		3 * time.Hour:    "3h ago",
		48 * time.Hour:   "2d ago",
	} {
		if got := ago(base.Add(-d), base); got != want {
			t.Errorf("ago(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestWheelMovesOneChatPerStep(t *testing.T) {
	m := loaded(t, testSessions())
	wheel := func(b tea.MouseButton) {
		next, _ := m.Update(tea.MouseMsg{Button: b, Action: tea.MouseActionPress})
		m = next.(Model)
	}
	wheel(tea.MouseButtonWheelDown)
	if m.items[m.cursor].session.ID != "3" {
		t.Errorf("one step down should reach chat 3, got %+v", m.items[m.cursor])
	}
	wheel(tea.MouseButtonWheelDown)
	if m.items[m.cursor].session.ID != "2" {
		t.Errorf("next step should skip the header to chat 2, got %+v", m.items[m.cursor])
	}
	wheel(tea.MouseButtonWheelUp)
	if m.items[m.cursor].session.ID != "3" {
		t.Errorf("one step up should return to chat 3, got %+v", m.items[m.cursor])
	}
}

func TestViewAfterStartingFromFolderScreen(t *testing.T) {
	m := press(loaded(t, testSessions()), "n")
	next, _ := m.Update(startedMsg{})
	got := next.(Model)
	if got.screen != sessionsScreen {
		t.Fatal("starting a chat should return to the chat list")
	}
	_ = got.View() // must not panic on stale folder lines
}

func TestScrollKeepsCursorVisible(t *testing.T) {
	var many []index.Session
	for i := range 30 {
		many = append(many, session(string(rune('a'+i%26))+string(rune('a'+i/26)), "/w", "chat", time.Duration(i)*time.Minute))
	}
	m := loaded(t, many)
	m.grouped = false
	m.rebuild()
	for range 29 {
		m = press(m, "down")
	}
	if m.cursor != 29 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	if lines := m.linesThrough(m.top, m.cursor); lines > m.bodyHeight() {
		t.Errorf("cursor item is cut off: %d lines for a body of %d", lines, m.bodyHeight())
	}
}
