package favorites

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToggleIsPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "favorites.json")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Has("a") {
		t.Fatal("new store should be empty")
	}
	if err := s.Toggle("a"); err != nil {
		t.Fatal(err)
	}

	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Has("a") {
		t.Error("favorite was not saved")
	}
	if err := again.Toggle("a"); err != nil {
		t.Fatal(err)
	}
	if again.Has("a") {
		t.Error("second toggle should remove the favorite")
	}
}

func TestToggleRollsBackWhenSavingFails(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "data")
	s, err := Load(filepath.Join(parent, "favorites.json"))
	if err != nil {
		t.Fatal(err)
	}
	// A file where the data folder should be makes saving fail on every platform.
	if err := os.WriteFile(parent, nil, 0o600); err != nil {
		t.Fatal(err)
	}
		t.Fatal("want an error")
	}
	if s.Has("a") {
		t.Error("failed save must not leave the favorite set")
	}
}

func TestLoadInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("want an error for invalid JSON")
	}
}

func TestFoldersArePersistedSeparately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ToggleFolder("/work/a"); err != nil {
		t.Fatal(err)
	}
	if s.Has("/work/a") {
		t.Error("a folder must not count as a chat favorite")
	}

	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !again.HasFolder("/work/a") {
		t.Error("folder favorite was not saved")
	}
}
