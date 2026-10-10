// Package favorites remembers which chats and folders the user starred.
package favorites

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Store is a set of starred session ids and folders kept in a JSON file.
type Store struct {
	path    string
	ids     map[string]bool
	folders map[string]bool
}

type file struct {
	Favorites []string `json:"favorites"`
	Folders   []string `json:"folders"`
}

// Load reads the store at path; a missing file is an empty store.
func Load(path string) (*Store, error) {
	s := &Store{path: path, ids: map[string]bool{}, folders: map[string]bool{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	for _, id := range f.Favorites {
		s.ids[id] = true
	}
	for _, dir := range f.Folders {
		s.folders[dir] = true
	}
	return s, nil
}

// Has reports whether the chat id is a favorite.
func (s *Store) Has(id string) bool { return s.ids[id] }

// HasFolder reports whether the folder is a favorite.
func (s *Store) HasFolder(dir string) bool { return s.folders[dir] }

// Toggle flips the favorite state of the chat id and saves the store. When
// saving fails the change is undone.
func (s *Store) Toggle(id string) error { return s.toggle(s.ids, id) }

// ToggleFolder flips the favorite state of a folder like Toggle does.
func (s *Store) ToggleFolder(dir string) error { return s.toggle(s.folders, dir) }

func (s *Store) toggle(set map[string]bool, key string) error {
	flip(set, key)
	if err := s.save(); err != nil {
		flip(set, key)
		return err
	}
	return nil
}

func flip(set map[string]bool, key string) {
	if set[key] {
		delete(set, key)
	} else {
		set[key] = true
	}
}

func (s *Store) save() error {
	f := file{Favorites: keys(s.ids), Folders: keys(s.folders)}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
