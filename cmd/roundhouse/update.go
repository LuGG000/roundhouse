package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/LuGG000/roundhouse/internal/paths"
	"github.com/LuGG000/roundhouse/internal/terminal"
	"github.com/LuGG000/roundhouse/internal/update"
)

// updateSchema is the version of the JSON printed by update --check --json.
const updateSchema = 1

// Time limits of the lookups: the update command is asked for, the notices are not.
const (
	lookupTimeout = 15 * time.Second
	noticeTimeout = 3 * time.Second
)

// devVersion is what a build without a release version reports.
const devVersion = "dev"

func newChecker() (*update.Checker, error) {
	dir, err := paths.DataDir()
	if err != nil {
		return nil, err
	}
	return &update.Checker{Dir: dir}, nil
}

func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// updateJSON is the `update` field of list --json: the newer release, if any.
type updateJSON struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// availableUpdate returns a release that is newer than this build, from the
// cache of the daily check. With network set, a cache older than a day is
// refreshed first, within noticeTimeout. It reports nothing for a development
// build, when the check is switched off, and when the lookup fails.
func (e *env) availableUpdate(ctx context.Context, network bool) (update.Release, bool) {
	if version == devVersion || !e.cfg.UpdateCheck {
		return update.Release{}, false
	}
	checker, err := newChecker()
	if err != nil {
		return update.Release{}, false
	}
	rel, isStale := checker.Cached()
	if isStale && network {
		ctx, cancel := context.WithTimeout(ctx, noticeTimeout)
		defer cancel()
		rel, _ = checker.Latest(ctx, false)
	}
	if rel.Version == "" || !update.Newer(version, rel.Version) {
		return update.Release{}, false
	}
	return rel, true
}

// updateNotice is the version of a newer release for the list to mention, or "".
func (e *env) updateNotice() string {
	rel, ok := e.availableUpdate(context.Background(), true)
	if !ok {
		return ""
	}
	return rel.Version
}

func newUpdate(e *env) *cobra.Command {
	var check, asJSON, inTerminal, pause bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Look for a newer release and, after you confirm, install it",
		Long: "Asks GitHub for the latest release and tells you whether it is newer than this one. Nothing is\n" +
			"installed until you answer yes at the prompt: there is no flag that skips it, and without a terminal\n" +
			"to answer in, update refuses. The archive is checked against the checksums published with the\n" +
			"release (the signature is not verified here; see the README for cosign). Running roundhouse\n" +
			"processes keep the old version until they are restarted.\n\n" +
			"--check only looks and prints (with --json for scripts); --terminal opens a new terminal window\n" +
			"that runs the update there, where you can answer.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if asJSON && !check {
				return errors.New("--json goes with --check")
			}
			if inTerminal {
				return updateInTerminal()
			}
			checker, err := newChecker()
			if err != nil {
				return err
			}
			exe, err := executablePath()
			if err != nil {
				return err
			}
			if pause {
				defer waitForEnter(cmd.InOrStdin(), cmd.OutOrStdout())
			}
			return updateFlow(cmd.Context(), updateDeps{
				checker:     checker,
				installer:   &update.Installer{},
				current:     version,
				exe:         exe,
				in:          cmd.InOrStdin(),
				out:         cmd.OutOrStdout(),
				check:       check,
				asJSON:      asJSON,
				interactive: isInteractive(cmd.InOrStdin()),
			})
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only look for a newer release and print it; install nothing")
	cmd.Flags().BoolVar(&asJSON, "json", false, "with --check: print machine-readable JSON")
	cmd.Flags().BoolVar(&inTerminal, "terminal", false, "run the update in a new terminal window, where you confirm it")
	cmd.Flags().BoolVar(&pause, "pause", false, "wait for Enter at the end, so a window that closes by itself can be read")
	_ = cmd.Flags().MarkHidden("pause")
	return cmd
}

// latestLooker finds the newest release.
type latestLooker interface {
	Latest(ctx context.Context, force bool) (update.Release, error)
}

// releaseInstaller installs a given version over target.
type releaseInstaller interface {
	Install(ctx context.Context, version, target string) error
}

type updateDeps struct {
	checker     latestLooker
	installer   releaseInstaller
	current     string
	exe         string
	in          io.Reader
	out         io.Writer
	check       bool
	asJSON      bool
	interactive bool
}

// checkJSON is what update --check --json prints.
type checkJSON struct {
	Schema          int    `json:"schema"`
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	URL             string `json:"url"`
}

// updateFlow looks up the latest release and, only when the user answers yes
// at the prompt, installs exactly the version that was shown.
func updateFlow(ctx context.Context, d updateDeps) error {
	if d.current == devVersion {
		return say(d.out, "This is a development build; it has no release to compare with. Releases: "+update.ReleasesPage)
	}

	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	rel, err := d.checker.Latest(ctx, true)
	if err != nil {
		return fmt.Errorf("could not look up the latest release: %w", err)
	}
	isNewer := update.Newer(d.current, rel.Version)

	if d.check {
		if d.asJSON {
			enc := json.NewEncoder(d.out)
			enc.SetIndent("", "  ")
			return enc.Encode(checkJSON{updateSchema, d.current, rel.Version, isNewer, rel.URL})
		}
		if !isNewer {
			return say(d.out, fmt.Sprintf("roundhouse %s is up to date.", d.current))
		}
		return say(d.out, fmt.Sprintf("roundhouse %s is available (you have %s): %s\nRun roundhouse update to install it.", rel.Version, d.current, rel.URL))
	}

	if !isNewer {
		return say(d.out, fmt.Sprintf("roundhouse %s is up to date.", d.current))
	}
	if !d.interactive {
		return errors.New("update needs a terminal to ask you; run it in one, or use --check to only look")
	}

	if err := say(d.out, fmt.Sprintf("roundhouse %s is available (you have %s).\nRelease notes: %s\nIt would replace %s after checking the download against the published checksums.", rel.Version, d.current, rel.URL, d.exe)); err != nil {
		return err
	}
	if !confirm(d.in, d.out, "Install roundhouse "+rel.Version+"? [y/N] ") {
		return say(d.out, "Nothing was changed.")
	}
	if err := d.installer.Install(ctx, rel.Version, d.exe); err != nil {
		return fmt.Errorf("update failed, nothing was changed: %w", err)
	}
	return say(d.out, "Updated to roundhouse "+rel.Version+". Open lists and Claude Code sessions keep running the old version until you restart them.\nThe mod updates separately: claude plugin update roundhouse-mod@roundhouse")
}

func say(out io.Writer, text string) error {
	_, err := fmt.Fprintln(out, text)
	return err
}

// confirm asks a yes/no question; only "y" or "yes" is a yes, anything else, an
// empty line or the end of input is a no.
func confirm(in io.Reader, out io.Writer, question string) bool {
	_, _ = fmt.Fprint(out, question)
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

// isInteractive reports whether in is a terminal a person can answer in.
func isInteractive(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

// updateInTerminal runs "roundhouse update" in a new terminal window, where the
// user answers the question.
func updateInTerminal() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd, err := terminal.Command(runtime.GOOS, exec.LookPath, cwd, []string{self, "update", "--pause"})
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// waitForEnter keeps a window that closes with its program open until Enter is pressed.
func waitForEnter(in io.Reader, out io.Writer) {
	_, _ = fmt.Fprint(out, "\nPress Enter to close this window.")
	_, _ = bufio.NewReader(in).ReadString('\n')
}
