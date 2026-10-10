// Package favorites remembers which chats the user starred.
package favorites

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Store is a set of session ids kept in a JSON file.
type Store struct {
	path string
	ids  map[string]bool
}

type file struct {
	Favorites []string `json:"favorites"`
}

// Load reads the store at path; a missing file is an empty store.
func Load(path string) (*Store, error) {
	s := &Store{path: path, ids: map[string]bool{}}
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
	return s, nil
}

// Has reports whether id is a favorite.
func (s *Store) Has(id string) bool { return s.ids[id] }

// Toggle flips the favorite state of id and saves the store. When saving
// fails the change is undone.
func (s *Store) Toggle(id string) error {
	s.flip(id)
	if err := s.save(); err != nil {
		s.flip(id)
		return err
	}
	return nil
}

func (s *Store) flip(id string) {
	if s.ids[id] {
		delete(s.ids, id)
	} else {
		s.ids[id] = true
	}
}

func (s *Store) save() error {
	f := file{Favorites: make([]string, 0, len(s.ids))}
	for id := range s.ids {
		f.Favorites = append(f.Favorites, id)
	}
	sort.Strings(f.Favorites)
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
