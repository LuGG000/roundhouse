package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/LuGG000/roundhouse/internal/index"
)

// item is one entry of the list.
type item struct {
	// header opens a boxed section of chats and footer closes it. Neither
	// can be selected.
	header, footer bool
	// favorites marks the header of the favorites section.
	favorites bool
	// starred marks a favorite folder (header or folder screen).
	starred bool
	session *index.Session
	// folder is set for headers and on the folder screen.
	folder string
	// count is the number of chats in the section or folder.
	count   int
	modTime time.Time
}

func (it item) selectable() bool { return !it.header && !it.footer }

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

// sessionItems lists the sessions matching query. Grouped, the favorites come
// first as their own section, followed by one section per folder in the order
// of its newest chat, favorite folders first. Flat, favorite chats are moved to
// the front of the list.
func sessionItems(sessions []index.Session, query string, grouped bool, favs Favorites) []item {
	var hits []*index.Session
	for i := range sessions {
		s := &sessions[i]
		if matches(query, s.Title, s.Cwd, s.LastPrompt) {
			hits = append(hits, s)
		}
	}

	if !grouped {
		sort.SliceStable(hits, func(i, j int) bool { return favs.Has(hits[i].ID) && !favs.Has(hits[j].ID) })
		items := make([]item, len(hits))
		for i, s := range hits {
			items[i] = item{session: s, modTime: s.ModTime}
		}
		return items
	}

	var favChats []*index.Session
	var order []string
	byFolder := map[string][]*index.Session{}
	for _, s := range hits {
		if favs.Has(s.ID) {
			favChats = append(favChats, s)
		}
		if _, ok := byFolder[s.Cwd]; !ok {
			order = append(order, s.Cwd)
		}
		byFolder[s.Cwd] = append(byFolder[s.Cwd], s)
	}

	var items []item
	section := func(head item, chats []*index.Session) {
		head.header, head.count = true, len(chats)
		items = append(items, head)
		for _, s := range chats {
			items = append(items, item{session: s, modTime: s.ModTime})
		}
		items = append(items, item{footer: true})
	}
	if len(favChats) > 0 {
		section(item{favorites: true}, favChats)
	}
	sort.SliceStable(order, func(i, j int) bool { return favs.HasFolder(order[i]) && !favs.HasFolder(order[j]) })
	for _, folder := range order {
		section(item{folder: folder, starred: favs.HasFolder(folder)}, byFolder[folder])
	}
	return items
}

// folderItems lists the distinct folders matching query, favorites first and
// then newest first.
func folderItems(sessions []index.Session, query string, favs Favorites) []item {
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
			items = append(items, item{folder: s.Cwd, starred: favs.HasFolder(s.Cwd), modTime: s.ModTime})
		}
		items[i].count++
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].starred && !items[j].starred })
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

// sameEntry reports whether both items show the same chat or folder.
func (it item) sameEntry(o item) bool {
	if it.session != nil || o.session != nil {
		return it.session != nil && o.session != nil && it.session.ID == o.session.ID
	}
	return it.folder == o.folder
}
