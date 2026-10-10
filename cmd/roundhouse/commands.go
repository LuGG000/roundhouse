package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/LuGG000/roundhouse/internal/index"
	"github.com/LuGG000/roundhouse/internal/starter"
)

// listSchema is the version of the JSON printed by list --json.
const listSchema = 1

type listJSON struct {
	Schema   int           `json:"schema"`
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

func newList() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List chats, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessions, err := scan()
			if err != nil {
				return err
			}
			if asJSON {
				out := listJSON{Schema: listSchema, Sessions: make([]sessionJSON, len(sessions))}
				for i, s := range sessions {
					out.Sessions[i] = sessionJSON{s.ID, s.Path, s.Cwd, s.Title, s.LastPrompt, s.ModTime, s.Size}
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, s := range sessions {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s  %s\n",
					s.ID, s.ModTime.Format("2006-01-02 15:04"), s.Cwd, strings.Join(strings.Fields(s.Title), " "))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print machine-readable JSON")
	return cmd
}

func newOpen(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "open <id>",
		Short: "Resume a chat in its folder; the id may be a unique prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
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
			return run(cmd)
		},
	}
}

func newNew(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "new [folder]",
		Short: "Start a new chat in a folder (default: the current folder)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
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
			return run(cmd)
		},
	}
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
