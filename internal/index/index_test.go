package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, root, project, id, body string, mod time.Time) {
	t.Helper()
	dir := filepath.Join(root, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func chat(cwd, prompt string) string {
	return `{"type":"user","cwd":"` + cwd + `"}` + "\n" +
		`{"type":"last-prompt","lastPrompt":"` + prompt + `"}` + "\n"
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	write(t, root, "-a", "old", chat("/a", "old chat"), now.Add(-time.Hour))
	write(t, root, "-b", "new", chat("/b", "new chat"), now)
	write(t, root, "-a", "empty", `{"type":"mode"}`+"\n", now)
	write(t, root, filepath.Join("-a", "old", "subagents"), "agent", chat("/a", "sub"), now)

	got, err := Scan(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Fatalf("got %+v, want [new old]", got)
	}
}

func TestScanMissingDir(t *testing.T) {
	got, err := Scan(filepath.Join(t.TempDir(), "none"), "")
	if err != nil || len(got) != 0 {
		t.Errorf("got %v, %v; want an empty list", got, err)
	}
}

func TestScanUsesCacheForUnchangedFiles(t *testing.T) {
	root := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	when := time.Now().Add(-time.Hour).Truncate(time.Second)
	write(t, root, "-a", "one", chat("/a", "original"), when)

	if _, err := Scan(root, cachePath); err != nil {
		t.Fatal(err)
	}

	// Same size and mtime, different content: only the cache can still say "original".
	write(t, root, "-a", "one", chat("/a", "changed!"), when)
	got, err := Scan(root, cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "original" {
		t.Fatalf("got %+v, want the cached title", got)
	}

	// A new mtime invalidates the entry.
	write(t, root, "-a", "one", chat("/a", "changed!"), when.Add(time.Minute))
	got, _ = Scan(root, cachePath)
	if got[0].Title != "changed!" {
		t.Errorf("title = %q, want the fresh one", got[0].Title)
	}
}

func TestScanIgnoresBrokenCache(t *testing.T) {
	root := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	if err := os.WriteFile(cachePath, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	write(t, root, "-a", "one", chat("/a", "hi"), time.Now())
	got, err := Scan(root, cachePath)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}
