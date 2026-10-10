package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/LuGG000/roundhouse/internal/index"
	"github.com/LuGG000/roundhouse/internal/starter"
	"github.com/LuGG000/roundhouse/internal/transcript"
)

var base = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

type fakeFavs struct {
	ids     map[string]bool
	folders map[string]bool
	fail    error
}

func (f *fakeFavs) Has(id string) bool { return f.ids[id] }

func (f *fakeFavs) HasFolder(dir string) bool { return f.folders[dir] }

func (f *fakeFavs) Toggle(id string) error {
	if f.fail != nil {
		return f.fail
	}
	f.ids[id] = !f.ids[id]
	return nil
}

func (f *fakeFavs) ToggleFolder(dir string) error {
	if f.fail != nil {
		return f.fail
	}
	if f.folders == nil {
		f.folders = map[string]bool{}
	}
	f.folders[dir] = !f.folders[dir]
	return nil
}

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

func loadedWith(t *testing.T, sessions []index.Session, favs *fakeFavs) Model {
	t.Helper()
	m := New(func() ([]index.Session, error) { return sessions, nil }, starter.Builtins()[0], favs)
	m.now = func() time.Time { return base }
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	next, _ = next.(Model).Update(loadedMsg{sessions: sessions})
	return next.(Model)
}

func loaded(t *testing.T, sessions []index.Session) Model {
	t.Helper()
	return loadedWith(t, sessions, &fakeFavs{ids: map[string]bool{}})
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

// outline describes items as H:folder, F (footer) or the session id.
func outline(m Model) string {
	var got []string
	for _, it := range m.items {
		switch {
		case it.header && it.favorites:
			got = append(got, "H:*")
		case it.header:
			got = append(got, "H:"+it.folder)
		case it.footer:
			got = append(got, "F")
		default:
			got = append(got, it.session.ID)
		}
	}
	return strings.Join(got, " ")
}

func TestGroupedByFolderKeepsNewestFolderFirst(t *testing.T) {
	m := loaded(t, testSessions())
	want := "H:/work/a 1 3 F H:/work/b 2 F"
	if got := outline(m); got != want {
		t.Errorf("items = %s, want %s", got, want)
	}
}

func TestFavoritesSectionComesFirst(t *testing.T) {
	m := loadedWith(t, testSessions(), &fakeFavs{ids: map[string]bool{"2": true}})
	want := "H:* 2 F H:/work/a 1 3 F H:/work/b 2 F"
	if got := outline(m); got != want {
		t.Errorf("items = %s, want %s", got, want)
	}
	if m.counter() != "3 chats" {
		t.Errorf("counter = %q; a favorite listed twice must count once", m.counter())
	}

	m = press(m, "g")
	if got := outline(m); got != "2 1 3" {
		t.Errorf("flat items = %s, want favorites first", got)
	}
}

func TestToggleFavoriteKeepsSelection(t *testing.T) {
	favs := &fakeFavs{ids: map[string]bool{}}
	m := loadedWith(t, testSessions(), favs)
	m = press(m, "down") // chat 3
	m = press(m, "f")
	if !favs.ids["3"] {
		t.Fatal("chat 3 should be a favorite")
	}
	if m.items[m.cursor].session.ID != "3" {
		t.Errorf("selection moved to %+v", m.items[m.cursor])
	}
	if !strings.HasPrefix(outline(m), "H:* 3 F") {
		t.Errorf("items = %s, want a favorites section first", outline(m))
	}
}

func TestToggleFavoriteReportsFailure(t *testing.T) {
	favs := &fakeFavs{ids: map[string]bool{}, fail: errors.New("disk full")}
	m := press(loadedWith(t, testSessions(), favs), "f")
	if !strings.Contains(m.status, "disk full") {
		t.Errorf("status = %q", m.status)
	}
}

func TestCursorSkipsHeadersAndFooters(t *testing.T) {
	m := loaded(t, testSessions())
	if m.items[m.cursor].session.ID != "1" {
		t.Fatalf("cursor starts on %+v", m.items[m.cursor])
	}
	m = press(m, "down")
	m = press(m, "down")
	if m.items[m.cursor].session.ID != "2" {
		t.Errorf("cursor on %+v, want chat 2 after skipping footer and header", m.items[m.cursor])
	}
}

func TestFilter(t *testing.T) {
	m := loaded(t, testSessions())
	m = press(m, "/")
	for _, r := range "docs" {
		m = press(m, string(r))
	}
	if got := outline(m); got != "H:/work/b 2 F" {
		t.Errorf("items = %s", got)
	}
	m = press(m, "esc")
	if len(m.items) != 7 {
		t.Errorf("esc should clear the filter, got %d items", len(m.items))
	}
}

func TestToggleGrouping(t *testing.T) {
	m := press(loaded(t, testSessions()), "g")
	if got := outline(m); got != "1 2 3" {
		t.Errorf("flat view should list only chats, got %s", got)
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
	// Selected chat 1 is a four-line card: header 0, chat 1 at 1-4, chat 3 at 5-6.
	m, cmd := click(headerLines + 5)
	if cmd != nil || m.items[m.cursor].session.ID != "3" {
		t.Fatalf("first click should only select chat 3, got cursor %+v", m.items[m.cursor])
	}
	m, _ = click(headerLines + 0) // a section header: ignored
	if m.items[m.cursor].session.ID != "3" {
		t.Error("clicking a header must not move the cursor")
	}
	// Chat 3 is now the card: header 0, chat 1 at 1-2, chat 3 at 3-6.
	m, _ = click(headerLines + 6)
	if m.status == "" {
		t.Error("second click on the selected chat should try to open it")
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
		t.Errorf("next step should skip footer and header to chat 2, got %+v", m.items[m.cursor])
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
	for _, grouped := range []bool{true, false} {
		m := loaded(t, many)
		m.grouped = grouped
		m.rebuild()
		for range 29 {
			m = press(m, "down")
		}
		if lines := m.linesThrough(m.top, m.cursor); lines > m.bodyHeight() {
			t.Errorf("grouped=%v: cursor card is cut off: %d lines for a body of %d", grouped, lines, m.bodyHeight())
		}
		if got := len(strings.Split(m.View(), "\n")); got != 24 {
			t.Errorf("grouped=%v: view has %d lines, want 24", grouped, got)
		}
	}
}

func TestViewFitsWindow(t *testing.T) {
	m := loaded(t, testSessions())
	view := m.View()
	if got := strings.Count(view, "\n") + 1; got != 24 {
		t.Errorf("view has %d lines, want 24", got)
	}
	for _, want := range []string{"fix build", "1m ago", "/work/a", "2 chats"} {
		if !strings.Contains(view, want) {
			t.Errorf("view misses %q:\n%s", want, view)
		}
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

func TestNoLineExceedsWindowWidth(t *testing.T) {
	favs := &fakeFavs{ids: map[string]bool{"2": true}}
	for _, width := range []int{40, 80, 100} {
		for _, key := range []string{"", "g", "n"} {
			m := loadedWith(t, testSessions(), favs)
			next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
			m = next.(Model)
			if key != "" {
				m = press(m, key)
			}
			for i, line := range strings.Split(m.View(), "\n") {
				if w := ansi.StringWidth(line); w > width {
					t.Errorf("width %d, key %q, line %d is %d cells wide: %q", width, key, i, w, line)
				}
			}
		}
	}
}

func TestFavoriteFolderComesFirst(t *testing.T) {
	favs := &fakeFavs{ids: map[string]bool{}, folders: map[string]bool{"/work/b": true}}
	m := loadedWith(t, testSessions(), favs)
	if got, want := outline(m), "H:/work/b 2 F H:/work/a 1 3 F"; got != want {
		t.Errorf("items = %s, want %s", got, want)
	}

	m = press(m, "n")
	if m.items[0].folder != "/work/b" || !m.items[0].starred {
		t.Errorf("folder screen should list the favorite folder first, got %+v", m.items)
	}
}

func TestStarFolderOfSelectedChat(t *testing.T) {
	favs := &fakeFavs{ids: map[string]bool{}}
	m := loadedWith(t, testSessions(), favs)
	m = press(m, "down") // chat 3 in /work/a
	m = press(m, "F")
	if !favs.folders["/work/a"] {
		t.Fatalf("folders = %v, want /work/a starred", favs.folders)
	}
	if m.items[m.cursor].session.ID != "3" {
		t.Errorf("selection moved to %+v", m.items[m.cursor])
	}
}

func TestStarFolderOnFolderScreen(t *testing.T) {
	favs := &fakeFavs{ids: map[string]bool{}}
	m := press(loadedWith(t, testSessions(), favs), "n")
	m = press(m, "down") // second folder: /work/b
	m = press(m, "f")
	if !favs.folders["/work/b"] {
		t.Fatalf("folders = %v, want /work/b starred", favs.folders)
	}
	if m.items[m.cursor].folder != "/work/b" || m.cursor != 0 {
		t.Errorf("starred folder should move to the top and stay selected, cursor %d on %+v", m.cursor, m.items[m.cursor])
	}
}
