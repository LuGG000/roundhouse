// Package index lists the sessions found under the claude projects folder.
package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/LuGG000/roundhouse/internal/transcript"
)

const (
	workers      = 8
	cacheVersion = 1
)

// Session is a transcript summary plus its file state.
type Session struct {
	transcript.Summary
	ModTime time.Time
	Size    int64
}

// cache remembers summaries so unchanged transcripts are not read again.
type cache struct {
	Version int                   `json:"version"`
	Entries map[string]cacheEntry `json:"entries"`
}

type cacheEntry struct {
	Size    int64              `json:"size"`
	ModTime time.Time          `json:"mod_time"`
	Summary transcript.Summary `json:"summary"`
}

// Scan reads every <projectsDir>/<project>/<id>.jsonl, newest first. Sessions
// without a prompt or a folder are skipped; unreadable files are skipped too.
// Nested folders such as subagents are not sessions and are not visited.
//
// cachePath names a file that spares re-reading unchanged transcripts; empty
// disables the cache. A cache that cannot be read or written only costs speed.
func Scan(projectsDir, cachePath string) ([]Session, error) {
	files, err := filepath.Glob(filepath.Join(projectsDir, "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}
	old := loadCache(cachePath)

	var (
		mu       sync.Mutex
		sessions = make([]Session, 0, len(files))
		wg       sync.WaitGroup
		jobs     = make(chan string)
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if s, ok := read(path, old.Entries); ok {
					mu.Lock()
					sessions = append(sessions, s)
					mu.Unlock()
				}
			}
		}()
	}
	for _, f := range files {
		jobs <- f
	}
	close(jobs)
	wg.Wait()

	saveCache(cachePath, sessions)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})
	return sessions, nil
}

func read(path string, cached map[string]cacheEntry) (Session, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return Session{}, false
	}
	if e, ok := cached[path]; ok && e.Size == st.Size() && e.ModTime.Equal(st.ModTime()) {
		return Session{Summary: e.Summary, ModTime: st.ModTime(), Size: st.Size()}, true
	}
	sum, err := transcript.Read(path)
	if err != nil || sum.Cwd == "" || sum.Title == "" {
		return Session{}, false
	}
	return Session{Summary: sum, ModTime: st.ModTime(), Size: st.Size()}, true
}

func loadCache(path string) cache {
	empty := cache{Version: cacheVersion, Entries: map[string]cacheEntry{}}
	if path == "" {
		return empty
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return empty
	}
	var c cache
	if err := json.Unmarshal(b, &c); err != nil || c.Version != cacheVersion || c.Entries == nil {
		return empty
	}
	return c
}

// saveCache replaces the cache with the current sessions, which also drops
// entries of deleted transcripts.
func saveCache(path string, sessions []Session) {
	if path == "" {
		return
	}
	c := cache{Version: cacheVersion, Entries: make(map[string]cacheEntry, len(sessions))}
	for _, s := range sessions {
		c.Entries[s.Path] = cacheEntry{Size: s.Size, ModTime: s.ModTime, Summary: s.Summary}
	}
	b, err := json.Marshal(c)
	if err != nil || os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}
