package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LuGG000/roundhouse/internal/index"
)

// item is one line of the list.
type item struct {
	// header lines group sessions by folder and cannot be selected.
	header  bool
	session *index.Session
	// folder is set for header lines and on the folder screen.
	folder string
	// count is the number of sessions in the folder (folder screen).
	count   int
	modTime time.Time
}

func (it item) selectable() bool { return !it.header }

// matches reports whether every word of query occurs in the haystack.
func matches(query string, fields ...string) bool {
	hay := strings.ToLower(strings.Join(fields, "\n"))
	for _, w := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}

// sessionItems lists the sessions matching query, optionally under a header
// line per folder. Folders keep the order of their newest session.
func sessionItems(sessions []index.Session, query string, grouped bool) []item {
	var hits []*index.Session
	for i := range sessions {
		s := &sessions[i]
		if matches(query, s.Title, s.Cwd, s.LastPrompt) {
			hits = append(hits, s)
		}
	}
	if !grouped {
		items := make([]item, len(hits))
		for i, s := range hits {
			items[i] = item{session: s, modTime: s.ModTime}
		}
		return items
	}

	var order []string
	byFolder := map[string][]*index.Session{}
	for _, s := range hits {
		if _, ok := byFolder[s.Cwd]; !ok {
			order = append(order, s.Cwd)
		}
		byFolder[s.Cwd] = append(byFolder[s.Cwd], s)
	}
	var items []item
	for _, folder := range order {
		items = append(items, item{header: true, folder: folder})
		for _, s := range byFolder[folder] {
			items = append(items, item{session: s, modTime: s.ModTime})
		}
	}
	return items
}

// folderItems lists the distinct folders matching query, newest first.
func folderItems(sessions []index.Session, query string) []item {
	var items []item
	pos := map[string]int{}
	for _, s := range sessions {
		if !matches(query, s.Cwd) {
			continue
		}
		i, ok := pos[s.Cwd]
		if !ok {
			i = len(items)
			pos[s.Cwd] = i
			items = append(items, item{folder: s.Cwd, modTime: s.ModTime})
		}
		items[i].count++
	}
	return items
}

// shortPath replaces the home folder with ~ and keeps the last two elements
// when the path is longer than max cells.
func shortPath(p string, max int) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			if rel == "." {
				p = "~"
			} else {
				p = filepath.Join("~", rel)
			}
		}
	}
	if len([]rune(p)) <= max {
		return p
	}
	return filepath.Join("…", filepath.Base(filepath.Dir(p)), filepath.Base(p))
}

// ago renders a coarse relative time.
func ago(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}
