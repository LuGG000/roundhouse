// Package index lists the sessions found under the claude projects folder.
package index

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/LuGG000/roundhouse/internal/transcript"
)

const workers = 8

// Session is a transcript summary plus its file state.
type Session struct {
	transcript.Summary
	ModTime time.Time
	Size    int64
}

// Scan reads every <projectsDir>/<project>/<id>.jsonl, newest first. Sessions
// without a prompt or a folder are skipped; unreadable files are skipped too.
// Nested folders such as subagents are not sessions and are not visited.
func Scan(projectsDir string) ([]Session, error) {
	files, err := filepath.Glob(filepath.Join(projectsDir, "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}

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
				if s, ok := read(path); ok {
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

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})
	return sessions, nil
}

func read(path string) (Session, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return Session{}, false
	}
	sum, err := transcript.Read(path)
	if err != nil || sum.Cwd == "" || sum.Title == "" {
		return Session{}, false
	}
	return Session{Summary: sum, ModTime: st.ModTime(), Size: st.Size()}, true
}
