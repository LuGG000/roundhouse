// Package transcript reads the metadata of a Claude Code session transcript.
package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxLine bounds a single transcript line; tool results can be large.
const maxLine = 256 << 20

var (
	cwdKey        = []byte(`"cwd":`)
	aiTitlePrefix = []byte(`{"type":"ai-title"`)
	lastPromptPre = []byte(`{"type":"last-prompt"`)
)

// Summary is what the hub shows about one session.
type Summary struct {
	// ID is the session id, the transcript's file name without extension.
	ID string
	// Path is the absolute path of the transcript file.
	Path string
	// Cwd is the folder the session was started in: the first cwd in the file.
	// Later lines can differ when the session was resumed from another folder.
	Cwd string
	// Title is the generated title, or the last prompt when there is none.
	Title string
	// LastPrompt is the most recent prompt of the user.
	LastPrompt string
}

// Read scans the transcript at path. Unknown line types are ignored and an
// oversized line ends the scan with the metadata found so far.
func Read(path string) (Summary, error) {
	s := Summary{
		ID:   strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Path: path,
	}
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer func() { _ = f.Close() }() // read-only

	var aiTitle string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), maxLine)
	for sc.Scan() {
		line := sc.Bytes()
		switch {
		case bytes.HasPrefix(line, aiTitlePrefix):
			aiTitle = stringField(line, "aiTitle")
		case bytes.HasPrefix(line, lastPromptPre):
			s.LastPrompt = stringField(line, "lastPrompt")
		case s.Cwd == "":
			s.Cwd = cwdOf(line)
		}
	}
	if err := sc.Err(); err != nil && !errors.Is(err, bufio.ErrTooLong) {
		return s, err
	}

	s.Title = aiTitle
	if s.Title == "" {
		s.Title = s.LastPrompt
	}
	return s, nil
}

// Exists reports whether a transcript file is readable.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

// cwdOf returns the top-level cwd of a line, without decoding the whole line.
func cwdOf(line []byte) string {
	i := bytes.LastIndex(line, cwdKey)
	if i < 0 {
		return ""
	}
	var cwd string
	if err := json.NewDecoder(bytes.NewReader(line[i+len(cwdKey):])).Decode(&cwd); err != nil {
		return ""
	}
	return cwd
}

func stringField(line []byte, name string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(line, &m); err != nil {
		return ""
	}
	var v string
	if err := json.Unmarshal(m[name], &v); err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}
