package starter

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResumeExpandsPlaceholders(t *testing.T) {
	dir := t.TempDir()
	s := Starter{Name: "x", Resume: []string{"tool", "--resume", "{id}", "--file={path}", "{cwd}"}}
	cmd, err := s.ResumeCommand(Target{ID: "abc", Path: "/p/abc.jsonl", Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Dir != dir {
		t.Errorf("Dir = %q, want %q", cmd.Dir, dir)
	}
	want := "tool --resume abc --file=/p/abc.jsonl " + dir
	if got := strings.Join(cmd.Args, " "); got != want {
		t.Errorf("args = %q, want %q", got, want)
	}
}

func TestBuildErrors(t *testing.T) {
	s := Starter{Name: "x", Resume: []string{"tool"}}
	if _, err := s.ResumeCommand(Target{Cwd: filepath.Join(t.TempDir(), "gone")}); err == nil {
		t.Error("want an error for a missing folder")
	}
	if _, err := s.ResumeCommand(Target{}); err == nil {
		t.Error("want an error without a folder")
	}
	if _, err := s.NewCommand(Target{Cwd: t.TempDir()}); err == nil {
		t.Error("want an error without a new command")
	}
}

func TestBuiltinsHaveBothCommands(t *testing.T) {
	for _, s := range Builtins() {
		if len(s.Resume) == 0 || len(s.New) == 0 {
			t.Errorf("builtin %q is incomplete", s.Name)
		}
	}
}
