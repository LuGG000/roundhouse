package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadBasic(t *testing.T) {
	s, err := Read(filepath.Join("testdata", "basic.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "basic" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.Cwd != `C:\work\repo` {
		t.Errorf("Cwd = %q, want the first cwd", s.Cwd)
	}
	if s.Title != "Fix the build" {
		t.Errorf("Title = %q", s.Title)
	}
	if s.LastPrompt != "and now tests" {
		t.Errorf("LastPrompt = %q, want the latest", s.LastPrompt)
	}
}

func TestReadFallsBackToLastPrompt(t *testing.T) {
	s, err := Read(filepath.Join("testdata", "untitled.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "do the thing" || s.Cwd != "/home/u/proj" {
		t.Errorf("got title %q cwd %q", s.Title, s.Cwd)
	}
}

func TestReadHugeLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.jsonl")
	big := `{"type":"user","message":{"content":"` + strings.Repeat("a", 5<<20) + `"},"cwd":"/p","sessionId":"huge"}` + "\n"
	if err := os.WriteFile(path, []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Cwd != "/p" {
		t.Errorf("Cwd = %q", s.Cwd)
	}
}

func TestReadMissing(t *testing.T) {
	if _, err := Read(filepath.Join(t.TempDir(), "none.jsonl")); err == nil {
		t.Error("want an error for a missing file")
	}
}
