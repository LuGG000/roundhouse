// Package starter builds the command that opens a session in Claude Code.
package starter

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Built-in starter names.
const (
	Claude     = "claude"
	Switchyard = "switchyard"
)

// Starter is a way to launch Claude Code. Each argument may use the
// placeholders {id}, {path} and {cwd}.
type Starter struct {
	Name string `toml:"name"`
	// Resume opens an existing session.
	Resume []string `toml:"resume"`
	// New opens a fresh session.
	New []string `toml:"new"`
}

// Target is the session a command is built for.
type Target struct {
	ID   string
	Path string
	// Cwd is the working directory of the started process.
	Cwd string
}

// Builtins are the starters that need no configuration.
func Builtins() []Starter {
	return []Starter{
		{Name: Claude, Resume: []string{"claude", "--resume", "{id}"}, New: []string{"claude"}},
		{Name: Switchyard, Resume: []string{"switchyard", "run", "--", "--resume", "{id}"}, New: []string{"switchyard", "run"}},
	}
}

// ResumeCommand returns the command that resumes t.
func (s Starter) ResumeCommand(t Target) (*exec.Cmd, error) {
	return build(s.Name, s.Resume, t)
}

// NewCommand returns the command that starts a fresh session in t.Cwd.
func (s Starter) NewCommand(t Target) (*exec.Cmd, error) {
	return build(s.Name, s.New, t)
}

func build(name string, args []string, t Target) (*exec.Cmd, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("starter %q has no command for this action", name)
	}
	if t.Cwd == "" {
		return nil, fmt.Errorf("no working directory known for this session")
	}
	if st, err := os.Stat(t.Cwd); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("folder %s does not exist", t.Cwd)
	}
	r := strings.NewReplacer("{id}", t.ID, "{path}", t.Path, "{cwd}", t.Cwd)
	expanded := make([]string, len(args))
	for i, a := range args {
		expanded[i] = r.Replace(a)
	}
	cmd := exec.Command(expanded[0], expanded[1:]...)
	cmd.Dir = t.Cwd
	return cmd, nil
}
