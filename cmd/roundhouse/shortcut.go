package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/LuGG000/roundhouse/internal/paths"
	"github.com/LuGG000/roundhouse/internal/shortcut"
)

func newShortcut() *cobra.Command {
	sc := &cobra.Command{
		Use:   "shortcut",
		Short: "Add roundhouse to the application launcher of your desktop",
	}

	install := &cobra.Command{
		Use:   "install",
		Short: "Make roundhouse findable and startable like an app",
		Long: "Create an entry for roundhouse in the application launcher of your user account: a Start menu\n" +
			"shortcut on Windows, a desktop entry on Linux and an app in ~/Applications on macOS. It opens a\n" +
			"terminal window with roundhouse in it. It writes only that one entry in your own folders, needs no\n" +
			"admin rights and points at this binary, so updating roundhouse in place keeps it working. Run it\n" +
			"again after moving the binary.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := shortcutEnv()
			if err != nil {
				return err
			}
			path, err := shortcut.Install(env)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\nroundhouse now shows up where your system lists apps.\n", path)
			return err
		},
	}

	remove := &cobra.Command{
		Use:   "remove",
		Short: "Remove the application launcher entry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := shortcutEnv()
			if err != nil {
				return err
			}
			path, removed, err := shortcut.Remove(env)
			if err != nil {
				return err
			}
			if !removed {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Nothing to remove: %s does not exist.\n", path)
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", path)
			return err
		},
	}

	sc.AddCommand(install, remove)
	return sc
}

// shortcutEnv describes this machine and this binary for the shortcut package.
func shortcutEnv() (shortcut.Env, error) {
	exe, err := executablePath()
	if err != nil {
		return shortcut.Env{}, err
	}
	data, err := paths.DataDir()
	if err != nil {
		return shortcut.Env{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return shortcut.Env{}, err
	}
	return shortcut.Env{
		GOOS:    runtime.GOOS,
		Exe:     exe,
		Home:    home,
		DataDir: data,
		Getenv:  os.Getenv,
		Look:    exec.LookPath,
		Run:     func(c *exec.Cmd) error { return c.Run() },
	}, nil
}
