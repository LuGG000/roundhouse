package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const (
	// defaultModSource is the GitHub repository whose marketplace file lists the mod.
	defaultModSource = "LuGG000/roundhouse"
	modMarketplace   = "roundhouse"
	modPlugin        = "roundhouse-mod"
)

func newMod() *cobra.Command {
	mod := &cobra.Command{
		Use:   "mod",
		Short: "Manage the optional Claude Code mod",
	}
	var source, scope string

	install := &cobra.Command{
		Use:   "install",
		Short: "Install the roundhouse mod into Claude Code",
		Long: "Add the roundhouse marketplace to Claude Code and install the mod from it. An installed mod is\n" +
			"switched on in every Claude Code session you start from then on, in every profile (the plugins\n" +
			"folder is shared), and adds /roundhouse. --source takes a GitHub repository (owner/repo), a git URL or\n" +
			"a local checkout of this repository.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			run, err := claudeRunner(cmd)
			if err != nil {
				return err
			}
			if err := installMod(run, source, scope); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Installed %s. It is on in every new Claude Code session; restart running ones to load it.\n", modPlugin)
			return err
		},
	}
	install.Flags().StringVar(&source, "source", defaultModSource, "marketplace source: owner/repo, git URL or local path")
	install.Flags().StringVar(&scope, "scope", "user", "installation scope: user, project or local")

	uninstall := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the roundhouse mod from Claude Code",
		Long:  "Uninstall the mod, so Claude Code sessions no longer load it. The marketplace entry stays, so a later install is quick.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			run, err := claudeRunner(cmd)
			if err != nil {
				return err
			}
			if err := uninstallMod(run, scope); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Uninstalled %s. New Claude Code sessions no longer load it.\n", modPlugin)
			return err
		},
	}
	uninstall.Flags().StringVar(&scope, "scope", "user", "scope to remove it from: user, project or local")

	mod.AddCommand(install, uninstall)
	return mod
}

// claudeRunner returns a function that runs `claude` with the given arguments,
// attached to the terminal of cmd.
func claudeRunner(cmd *cobra.Command) (func(args ...string) error, error) {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return nil, errors.New("claude executable not found in PATH")
	}
	return func(args ...string) error {
		c := exec.CommandContext(cmd.Context(), claude, args...)
		c.Env = os.Environ()
		c.Stdin, c.Stdout, c.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()
		return c.Run()
	}, nil
}

// installMod adds the marketplace and installs the mod from it using run, which
// executes `claude` with the given arguments.
func installMod(run func(args ...string) error, source, scope string) error {
	if err := run("plugin", "marketplace", "add", source, "--scope", scope); err != nil {
		return fmt.Errorf("add marketplace %s: %w", source, err)
	}
	if err := run("plugin", "install", modPlugin+"@"+modMarketplace, "--scope", scope); err != nil {
		return fmt.Errorf("install %s: %w", modPlugin, err)
	}
	return nil
}

// uninstallMod removes the mod using run, which executes `claude`.
func uninstallMod(run func(args ...string) error, scope string) error {
	if err := run("plugin", "uninstall", modPlugin+"@"+modMarketplace, "--scope", scope); err != nil {
		return fmt.Errorf("uninstall %s: %w", modPlugin, err)
	}
	return nil
}
