package shortcut

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func has(names ...string) func(string) (string, error) {
	return func(file string) (string, error) {
		for _, n := range names {
			if n == file {
				return filepath.Join("bin", n), nil
			}
		}
		return "", errors.New("not found")
	}
}

func env(goos, home string, look func(string) (string, error)) (Env, *[]*exec.Cmd) {
	var ran []*exec.Cmd
	return Env{
		GOOS:    goos,
		Exe:     "/opt/round house/roundhouse",
		Home:    home,
		DataDir: filepath.Join(home, "data"),
		Getenv:  func(string) string { return "" },
		Look:    look,
		Run:     func(c *exec.Cmd) error { ran = append(ran, c); return nil },
	}, &ran
}

func TestLinuxWritesADesktopEntryThatRunsInATerminal(t *testing.T) {
	home := t.TempDir()
	e, _ := env("linux", home, has())
	path, err := Install(e)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "share", "applications", "roundhouse.desktop"); path != want {
		t.Fatalf("path = %s, want %s", path, want)
	}
	b, _ := os.ReadFile(path)
	for _, want := range []string{"[Desktop Entry]", "Type=Application", "Name=roundhouse", `Exec="/opt/round house/roundhouse"`, "Terminal=true"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("entry misses %q:\n%s", want, b)
		}
	}
}

func TestLinuxEntryPointsAtItsIcon(t *testing.T) {
	home := t.TempDir()
	e, _ := env("linux", home, has())
	path, err := Install(e)
	if err != nil {
		t.Fatal(err)
	}
	icon := filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps", "roundhouse.png")
	if b, err := os.ReadFile(icon); err != nil || !strings.HasPrefix(string(b), "\x89PNG") {
		t.Fatalf("icon %s: %v", icon, err)
	}
	if entry, _ := os.ReadFile(path); !strings.Contains(string(entry), "Icon="+icon+"\n") {
		t.Errorf("entry does not name the icon:\n%s", entry)
	}
}

func TestLinuxHonoursXDGDataHome(t *testing.T) {
	home, data := t.TempDir(), t.TempDir()
	e, _ := env("linux", home, has())
	e.Getenv = func(k string) string {
		if k == "XDG_DATA_HOME" {
			return data
		}
		return ""
	}
	path, err := Install(e)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(data, "applications", "roundhouse.desktop"); path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
}

func TestDesktopExecQuoteEscapesWhatStaysSpecial(t *testing.T) {
	got := desktopExecQuote(`/a b/"c"/$d/100%`)
	want := `"/a b/\"c\"/\$d/100%%"`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestDarwinWritesAnAppBundleWithAnExecutableLauncher(t *testing.T) {
	home := t.TempDir()
	e, _ := env("darwin", home, has())
	e.Exe = "/Users/me/it's here/roundhouse"
	path, err := Install(e)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "Applications", "Roundhouse.app"); path != want {
		t.Fatalf("path = %s, want %s", path, want)
	}
	launcher := filepath.Join(path, "Contents", "MacOS", darwinLauncher)
	fi, err := os.Stat(launcher)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode()&0o111 == 0 {
		t.Errorf("launcher is not executable: %v", fi.Mode())
	}
	b, _ := os.ReadFile(launcher)
	for _, want := range []string{"#!/bin/sh", "osascript", `tell application "Terminal" to activate`, `do script "exec '\''/Users/me/it'\''\\'\'''\''s here/roundhouse'\''"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("launcher misses %q:\n%s", want, b)
		}
	}
	if b, err := os.ReadFile(filepath.Join(path, "Contents", "Resources", "roundhouse.icns")); err != nil || !strings.HasPrefix(string(b), "icns") {
		t.Errorf("icon file is missing or not an .icns: %v", err)
	}
	plist, _ := os.ReadFile(filepath.Join(path, "Contents", "Info.plist"))
	if !strings.Contains(string(plist), "<string>"+darwinLauncher+"</string>") {
		t.Errorf("Info.plist does not name the launcher:\n%s", plist)
	}
}

func TestWindowsCreatesTheStartMenuShortcutThroughPowerShell(t *testing.T) {
	home := t.TempDir()
	e, ran := env("windows", home, has("wt.exe", "powershell.exe"))
	e.Exe = `C:\Users\Lu's PC\roundhouse.exe`
	path, err := Install(e)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "roundhouse.lnk"); path != want {
		t.Fatalf("path = %s, want %s", path, want)
	}
	if b, err := os.ReadFile(filepath.Join(home, "data", "roundhouse.ico")); err != nil || len(b) < 6 || b[2] != 1 {
		t.Errorf("icon file is missing or not an .ico: %v", err)
	}
	if len(*ran) != 1 {
		t.Fatalf("ran %d commands, want 1", len(*ran))
	}
	script := (*ran)[0].Args[len((*ran)[0].Args)-1]
	for _, want := range []string{
		"CreateShortcut('" + path + "')",
		"$s.TargetPath = '" + filepath.Join("bin", "wt.exe") + "'",
		`$s.Arguments = '-w new "C:\Users\Lu''s PC\roundhouse.exe"'`,
		"$s.WorkingDirectory = '" + home + "'",
		"$s.IconLocation = '" + filepath.Join(home, "data", "roundhouse.ico") + ",0'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script misses %q:\n%s", want, script)
		}
	}
}

func TestWindowsWithoutWindowsTerminalStartsTheProgramItself(t *testing.T) {
	e, ran := env("windows", t.TempDir(), has("powershell.exe"))
	e.Exe = `C:\bin\roundhouse.exe`
	if _, err := Install(e); err != nil {
		t.Fatal(err)
	}
	script := (*ran)[0].Args[len((*ran)[0].Args)-1]
	if !strings.Contains(script, `$s.TargetPath = 'C:\bin\roundhouse.exe'`) || !strings.Contains(script, "$s.Arguments = ''") {
		t.Errorf("script:\n%s", script)
	}
}

func TestWindowsNeedsPowerShell(t *testing.T) {
	e, _ := env("windows", t.TempDir(), has())
	if _, err := Install(e); err == nil {
		t.Fatal("want an error without powershell.exe")
	}
}

func TestWindowsReportsAFailedHelper(t *testing.T) {
	e, _ := env("windows", t.TempDir(), has("powershell.exe"))
	e.Run = func(*exec.Cmd) error { return errors.New("boom") }
	if _, err := Install(e); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownSystemIsRefused(t *testing.T) {
	e, _ := env("plan9", t.TempDir(), has())
	if _, err := Install(e); err == nil {
		t.Error("Install: want an error")
	}
	if _, _, err := Remove(e); err == nil {
		t.Error("Remove: want an error")
	}
}

func TestRemoveDeletesWhatInstallCreated(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		home := t.TempDir()
		e, _ := env(goos, home, has())
		path, err := Install(e)
		if err != nil {
			t.Fatal(err)
		}
		got, removed, err := Remove(e)
		if err != nil || !removed || got != path {
			t.Fatalf("%s: Remove = %s, %v, %v", goos, got, removed, err)
		}
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: %s is still there", goos, path)
		}
		if icon := iconPath(e); icon != "" {
			if _, err := os.Lstat(icon); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("%s: the icon %s is still there", goos, icon)
			}
		}
	}
}

func TestRemoveWithoutAnEntryIsNotAnError(t *testing.T) {
	e, _ := env("linux", t.TempDir(), has())
	_, removed, err := Remove(e)
	if err != nil || removed {
		t.Errorf("removed = %v, err = %v", removed, err)
	}
}

func TestRemoveLeavesAForeignAppFolderAlone(t *testing.T) {
	home := t.TempDir()
	app := filepath.Join(home, "Applications", "Roundhouse.app")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	e, _ := env("darwin", home, has())
	if _, _, err := Remove(e); err == nil {
		t.Fatal("want an error")
	}
	if _, err := os.Stat(app); err != nil {
		t.Errorf("the folder was removed: %v", err)
	}
}
