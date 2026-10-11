// Command roundhouse is a hub to start new Claude Code chats and return to
// old ones, in any folder.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/LuGG000/roundhouse/internal/config"
	"github.com/LuGG000/roundhouse/internal/favorites"
	"github.com/LuGG000/roundhouse/internal/index"
	"github.com/LuGG000/roundhouse/internal/paths"
	"github.com/LuGG000/roundhouse/internal/starter"
	"github.com/LuGG000/roundhouse/internal/tui"
	"github.com/LuGG000/roundhouse/internal/update"
)

// version is set at build time.
var version = "dev"

func main() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "roundhouse:", err)
		os.Exit(1)
	}
}

// env is what the commands share: the config and the starter it names. A broken
// starter is only an error for the commands that start Claude Code.
type env struct {
	cfg        config.Config
	starterErr error
	starter    starter.Starter
}

func newRoot() *cobra.Command {
	var starterName string
	var e env

	root := &cobra.Command{
		Use:           "roundhouse",
		Short:         "Start new Claude Code chats and return to old ones, in any folder",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if starterName != "" {
				cfg.Starter = starterName
			}
			e.cfg = cfg
			e.starter, e.starterErr = cfg.ActiveStarter()
			if exe, err := executablePath(); err == nil {
				update.CleanOld(exe)
			}
			return nil
		},
		RunE: func(*cobra.Command, []string) error { return e.runTUI() },
	}
	root.PersistentFlags().StringVar(&starterName, "starter", "", "starter to launch Claude Code with (default from config, else claude)")
	root.AddCommand(newList(&e), newOpen(&e), newNew(&e), newMod(), newShortcut(), newUpdate(&e))
	return root
}

func (e *env) runTUI() error {
	if e.starterErr != nil {
		return e.starterErr
	}
	favs, err := openFavorites()
	if err != nil {
		return err
	}
	m := tui.New(scan, e.starter, favs, tui.WithUpdateNotice(e.updateNotice))
	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

func scan() ([]index.Session, error) {
	dir, err := paths.ProjectsDir()
	if err != nil {
		return nil, err
	}
	data, err := paths.DataDir()
	if err != nil {
		return nil, err
	}
	return index.Scan(dir, filepath.Join(data, "index-cache.json"))
}

func openFavorites() (*favorites.Store, error) {
	data, err := paths.DataDir()
	if err != nil {
		return nil, err
	}
	return favorites.Load(filepath.Join(data, "favorites.json"))
}

// run starts cmd attached to the terminal. A non-zero exit of the started
// program is passed on as the exit code of roundhouse, not reported as an error.
func run(cmd *exec.Cmd) error {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
	return err
}
