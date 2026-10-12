package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/LuGG000/roundhouse/internal/index"
	"github.com/LuGG000/roundhouse/internal/starter"
	"github.com/LuGG000/roundhouse/internal/terminal"
)

// listSchema is the version of the JSON printed by list --json.
const listSchema = 1

type listJSON struct {
	Schema int `json:"schema"`
	// Version is this roundhouse; Update is a newer release found by the daily check, or null.
	Version  string        `json:"version"`
	Update   *updateJSON   `json:"update"`
	Sessions []sessionJSON `json:"sessions"`
}

type sessionJSON struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Cwd        string    `json:"cwd"`
	Title      string    `json:"title"`
	LastPrompt string    `json:"last_prompt"`
	Modified   time.Time `json:"modified"`
	Size       int64     `json:"size"`
}

func newList(e *env) *cobra.Command {
	var asJSON bool
	var folder string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List chats, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessions, err := scan()
			if err != nil {
				return err
			}
			if folder != "" {
				sessions = inFolder(sessions, folder)
			}
			if asJSON {
				out := listJSON{Schema: listSchema, Version: version, Sessions: make([]sessionJSON, len(sessions))}
				if rel, ok := e.availableUpdate(cmd.Context(), true); ok {
					out.Update = &updateJSON{Version: rel.Version, URL: rel.URL}
				}
				for i, s := range sessions {
					out.Sessions[i] = sessionJSON{s.ID, s.Path, s.Cwd, s.Title, s.LastPrompt, s.ModTime, s.Size}
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, s := range sessions {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s  %s\n",
					s.ID, s.ModTime.Format("2006-01-02 15:04"), s.Cwd, strings.Join(strings.Fields(s.Title), " ")); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print machine-readable JSON")
	cmd.Flags().StringVar(&folder, "folder", "", "list only the chats started in this folder")
	return cmd
}

func newOpen(e *env) *cobra.Command {
	var inTerminal bool
	cmd := &cobra.Command{
		Use:   "open <id>",
		Short: "Resume a chat in its folder; the id may be a unique prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if e.starterErr != nil {
				return e.starterErr
			}
			sessions, err := scan()
			if err != nil {
				return err
			}
			s, err := find(sessions, args[0])
			if err != nil {
				return err
			}
			cmd, err := e.starter.ResumeCommand(starter.Target{ID: s.ID, Path: s.Path, Cwd: s.Cwd})
			if err != nil {
				return err
			}
			if inTerminal {
				return inNewWindow(s.Cwd, "open", s.ID, "--starter", e.starter.Name)
			}
			return run(cmd)
		},
	}
	cmd.Flags().BoolVar(&inTerminal, "terminal", false, "open the chat in a new terminal window and return at once")
	return cmd
}

// inNewWindow runs roundhouse with args in a new terminal window in dir and
// does not wait for it.
func inNewWindow(dir string, args ...string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd, err := terminal.Command(runtime.GOOS, exec.LookPath, dir, append([]string{self}, args...))
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func newNew(e *env) *cobra.Command {
	var inTerminal bool
	cmd := &cobra.Command{
		Use:   "new [folder]",
		Short: "Start a new chat in a folder (default: the current folder)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if e.starterErr != nil {
				return e.starterErr
			}
			dir, err := os.Getwd()
			if len(args) == 1 {
				dir, err = args[0], nil
			}
			if err != nil {
				return err
			}
			cmd, err := e.starter.NewCommand(starter.Target{Cwd: dir})
			if err != nil {
				return err
			}
			if inTerminal {
				return inNewWindow(dir, "new", dir, "--starter", e.starter.Name)
			}
			return run(cmd)
		},
	}
	cmd.Flags().BoolVar(&inTerminal, "terminal", false, "start the chat in a new terminal window and return at once")
	return cmd
}

// find returns the session whose id equals or uniquely starts with prefix.
func find(sessions []index.Session, prefix string) (index.Session, error) {
	var hit []index.Session
	for _, s := range sessions {
		if s.ID == prefix {
			return s, nil
		}
		if strings.HasPrefix(s.ID, prefix) {
			hit = append(hit, s)
		}
	}
	switch len(hit) {
	case 0:
		return index.Session{}, fmt.Errorf("no chat with id %q", prefix)
	case 1:
		return hit[0], nil
	default:
		return index.Session{}, fmt.Errorf("id %q is ambiguous (%d chats)", prefix, len(hit))
	}
}

// inFolder keeps the sessions that were started in folder.
func inFolder(sessions []index.Session, folder string) []index.Session {
	var out []index.Session
	for _, s := range sessions {
		if sameFolder(s.Cwd, folder) {
			out = append(out, s)
		}
	}
	return out
}

// sameFolder reports whether a and b name the same folder; Windows ignores case.
func sameFolder(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
