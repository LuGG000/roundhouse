package terminal

import (
	"errors"
	"strings"
	"testing"
)

func has(names ...string) LookPath {
	return func(file string) (string, error) {
		for _, n := range names {
			if n == file {
				return "/bin/" + n, nil
			}
		}
		return "", errors.New("not found")
	}
}

func args(t *testing.T, goos string, look LookPath, cwd string, argv ...string) string {
	t.Helper()
	cmd, err := Command(goos, look, cwd, argv)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(cmd.Args, " ")
}

func TestWindowsTerminal(t *testing.T) {
	got := args(t, "windows", has("wt.exe"), `C:\work\repo`, "roundhouse", "open", "abc")
	want := `wt.exe -w new -d C:\work\repo roundhouse open abc`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWindowsFallsBackToCmdStart(t *testing.T) {
	got := args(t, "windows", has(), `C:\work\repo`, "roundhouse", "open", "abc")
	want := `cmd.exe /c start roundhouse /D C:\work\repo roundhouse open abc`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDarwinQuotesTheFolderAndArguments(t *testing.T) {
	cmd, err := Command("darwin", has(), "/Users/me/it's here", []string{"/opt/round house/roundhouse", "open", "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Args[0] != "osascript" {
		t.Fatalf("command = %v", cmd.Args)
	}
	script := cmd.Args[len(cmd.Args)-1]
	for _, want := range []string{
		`do script "cd '/Users/me/it'\\''s here' && exec '/opt/round house/roundhouse' 'open' 'abc'"`,
		`tell application "Terminal" to activate`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script misses %q:\n%s", want, script)
		}
	}
}

func TestLinuxPicksTheFirstTerminalThatExists(t *testing.T) {
	got := args(t, "linux", has("konsole", "xterm"), "/work/repo", "roundhouse", "open", "abc")
	want := `konsole -e sh -c cd "$0" && exec "$@" /work/repo roundhouse open abc`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLinuxTerminalFlags(t *testing.T) {
	for name, flag := range map[string]string{
		"x-terminal-emulator": "-e",
		"gnome-terminal":      "--",
		"xfce4-terminal":      "-x",
		"xterm":               "-e",
	} {
		got := args(t, "linux", has(name), "/w", "roundhouse")
		if !strings.HasPrefix(got, name+" "+flag+" sh -c ") {
			t.Errorf("%s: got %q", name, got)
		}
	}
}

func TestLinuxWithoutTerminal(t *testing.T) {
	if _, err := Command("linux", has(), "/w", []string{"roundhouse"}); err == nil {
		t.Error("want an error when no terminal emulator exists")
	}
}

func TestNothingToRun(t *testing.T) {
	if _, err := Command("windows", has("wt.exe"), "/w", nil); err == nil {
		t.Error("want an error for an empty argument list")
	}
}
