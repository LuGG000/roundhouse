// Package terminal builds the command that opens a program in a new terminal window.
package terminal

import (
	"errors"
	"os/exec"
	"strings"
)

// LookPath finds an executable on the PATH, like exec.LookPath.
type LookPath func(file string) (string, error)

// Command returns the command that runs argv in folder cwd inside a new
// terminal window of the platform goos. Starting it opens the window and does
// not wait for argv to finish. The window systems differ, and only Windows
// Terminal has been tried on a real machine; the others are best effort.
func Command(goos string, look LookPath, cwd string, argv []string) (*exec.Cmd, error) {
	if len(argv) == 0 {
		return nil, errors.New("nothing to run")
	}
	switch goos {
	case "windows":
		return windows(look, cwd, argv), nil
	case "darwin":
		return darwin(cwd, argv), nil
	default:
		return linux(look, cwd, argv)
	}
}

func windows(look LookPath, cwd string, argv []string) *exec.Cmd {
	if _, err := look("wt.exe"); err == nil {
		return exec.Command("wt.exe", append([]string{"-w", "new", "-d", cwd}, argv...)...)
	}
	return exec.Command("cmd.exe", append([]string{"/c", "start", "roundhouse", "/D", cwd}, argv...)...)
}

func darwin(cwd string, argv []string) *exec.Cmd {
	line := "cd " + shellQuote(cwd) + " && exec " + joinQuoted(argv)
	script := `tell application "Terminal" to activate` + "\n" +
		`tell application "Terminal" to do script "` + appleScriptEscape(line) + `"`
	return exec.Command("osascript", "-e", script)
}

// linuxTerminals are tried in order; each runs the arguments after its flag.
var linuxTerminals = []struct{ name, flag string }{
	{"x-terminal-emulator", "-e"},
	{"gnome-terminal", "--"},
	{"konsole", "-e"},
	{"xfce4-terminal", "-x"},
	{"xterm", "-e"},
}

func linux(look LookPath, cwd string, argv []string) (*exec.Cmd, error) {
	// sh enters the folder, so no terminal needs a working-directory flag of its own.
	inFolder := append([]string{"sh", "-c", `cd "$0" && exec "$@"`, cwd}, argv...)
	for _, t := range linuxTerminals {
		if _, err := look(t.name); err == nil {
			return exec.Command(t.name, append([]string{t.flag}, inFolder...)...), nil
		}
	}
	return nil, errors.New("no terminal emulator found (tried x-terminal-emulator, gnome-terminal, konsole, xfce4-terminal, xterm)")
}

func joinQuoted(argv []string) string {
	q := make([]string, len(argv))
	for i, a := range argv {
		q[i] = shellQuote(a)
	}
	return strings.Join(q, " ")
}

// shellQuote quotes s for a POSIX shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// appleScriptEscape escapes s for use inside an AppleScript string literal.
func appleScriptEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}
