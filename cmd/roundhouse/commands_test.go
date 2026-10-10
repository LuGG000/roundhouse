package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/LuGG000/roundhouse/internal/index"
	"github.com/LuGG000/roundhouse/internal/transcript"
)

func session(id, cwd string) index.Session {
	return index.Session{Summary: transcript.Summary{ID: id, Cwd: cwd}}
}

func TestSameFolderIgnoresSeparatorsAndTrailingSlash(t *testing.T) {
	a := filepath.Join("work", "repo")
	if !sameFolder(a, a+string(filepath.Separator)) {
		t.Error("a trailing separator should not matter")
	}
	if sameFolder(a, filepath.Join("work", "other")) {
		t.Error("different folders must differ")
	}
}

func TestSameFolderCaseOnWindowsOnly(t *testing.T) {
	got := sameFolder(filepath.Join("Work", "Repo"), filepath.Join("work", "repo"))
	if want := runtime.GOOS == "windows"; got != want {
		t.Errorf("sameFolder with different case = %v, want %v on %s", got, want, runtime.GOOS)
	}
}

func TestInFolderKeepsOnlyExactMatches(t *testing.T) {
	sessions := []index.Session{
		session("1", filepath.Join("work", "a")),
		session("2", filepath.Join("work", "a", "sub")),
		session("3", filepath.Join("work", "b")),
		session("4", filepath.Join("work", "a")+string(filepath.Separator)),
	}
	got := inFolder(sessions, filepath.Join("work", "a"))
	if len(got) != 2 || got[0].ID != "1" || got[1].ID != "4" {
		t.Errorf("got %+v, want chats 1 and 4 (a sub folder is not the folder)", got)
	}
}

func TestFindByPrefix(t *testing.T) {
	sessions := []index.Session{session("abc1", "/a"), session("abc2", "/a"), session("xyz", "/a")}
	if s, err := find(sessions, "xy"); err != nil || s.ID != "xyz" {
		t.Errorf("unique prefix: got %v, %v", s.ID, err)
	}
	if _, err := find(sessions, "abc"); err == nil {
		t.Error("an ambiguous prefix should fail")
	}
	if _, err := find(sessions, "nope"); err == nil {
		t.Error("an unknown id should fail")
	}
	if s, err := find(sessions, "abc1"); err != nil || s.ID != "abc1" {
		t.Errorf("exact id wins over prefix matches: got %v, %v", s.ID, err)
	}
}
