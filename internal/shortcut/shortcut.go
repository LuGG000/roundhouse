// Package shortcut puts roundhouse into the application launcher of the desktop, so it can be found
// and started like any other app: a Start menu entry on Windows, a .desktop entry on Linux and an
// application bundle on macOS. It writes one entry in the user's own folders and never needs admin
// rights.
package shortcut

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	appName     = "roundhouse"
	description = "Find, resume and start Claude Code chats from one list"
)

// The icon of the entry, drawn by icon/gen.go: the Windows shortcut points at the .ico, the desktop
// entry at the .png and the macOS bundle carries the .icns.
var (
	//go:embed icon/roundhouse.ico
	iconICO []byte
	//go:embed icon/roundhouse.png
	iconPNG []byte
	//go:embed icon/roundhouse.icns
	iconICNS []byte
)

// Env is everything an install or removal depends on, so tests can run without touching the system.
type Env struct {
	GOOS    string
	Exe     string                       // absolute path of the roundhouse binary the entry starts
	Home    string                       // the user's home folder
	DataDir string                       // roundhouse's data folder; the Windows shortcut keeps its icon file there
	Getenv  func(string) string          // environment lookup
	Look    func(string) (string, error) // executable lookup, like exec.LookPath
	Run     func(*exec.Cmd) error        // runs a helper program; only Windows needs one
}

// Install creates the launcher entry and returns where it is.
func Install(e Env) (string, error) {
	switch e.GOOS {
	case "windows":
		return installWindows(e)
	case "darwin":
		return installDarwin(e)
	case "linux", "freebsd", "openbsd", "netbsd":
		return installLinux(e)
	}
	return "", fmt.Errorf("no application launcher entry for %s", e.GOOS)
}

// Remove deletes the entry that Install created. It reports where the entry was, and whether there was
// one; removing nothing is not an error.
func Remove(e Env) (string, bool, error) {
	path, err := entryPath(e)
	if err != nil {
		return "", false, err
	}
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return path, false, nil
	}
	if e.GOOS == "darwin" {
		// A folder is only removed when it is the bundle this package writes.
		if _, err := os.Stat(filepath.Join(path, "Contents", "MacOS", darwinLauncher)); err != nil {
			return path, false, fmt.Errorf("%s is not a roundhouse bundle; leaving it alone", path)
		}
		return path, true, os.RemoveAll(path)
	}
	if err := os.Remove(path); err != nil {
		return path, false, err
	}
	if icon := iconPath(e); icon != "" {
		if err := os.Remove(icon); err != nil && !errors.Is(err, os.ErrNotExist) {
			return path, true, err
		}
	}
	return path, true, nil
}

// iconPath is where the icon file of the entry lives, or "" when it is part of the entry itself.
func iconPath(e Env) string {
	switch e.GOOS {
	case "windows":
		return filepath.Join(e.DataDir, appName+".ico")
	case "linux", "freebsd", "openbsd", "netbsd":
		path, _ := entryPath(e)
		return filepath.Join(filepath.Dir(filepath.Dir(path)), "icons", "hicolor", "256x256", "apps", appName+".png")
	}
	return ""
}

// entryPath is where the entry of the platform lives.
func entryPath(e Env) (string, error) {
	switch e.GOOS {
	case "windows":
		appData := e.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(e.Home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", appName+".lnk"), nil
	case "darwin":
		return filepath.Join(e.Home, "Applications", "Roundhouse.app"), nil
	case "linux", "freebsd", "openbsd", "netbsd":
		data := e.Getenv("XDG_DATA_HOME")
		if data == "" {
			data = filepath.Join(e.Home, ".local", "share")
		}
		return filepath.Join(data, "applications", appName+".desktop"), nil
	}
	return "", fmt.Errorf("no application launcher entry for %s", e.GOOS)
}

// writeFile writes content with mode, creating the folders above it.
func writeFile(path, content string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), mode)
}

// installWindows writes a Start menu shortcut. Windows has no plain-text form of a .lnk, so
// PowerShell creates it through the Windows Script Host. With Windows Terminal installed the
// shortcut opens a window of it; otherwise Windows opens the program in the default console.
func installWindows(e Env) (string, error) {
	path, _ := entryPath(e)
	target, args := e.Exe, ""
	if wt, err := e.Look("wt.exe"); err == nil {
		target, args = wt, `-w new "`+e.Exe+`"`
	}
	ps, err := e.Look("powershell.exe")
	if err != nil {
		return "", errors.New("powershell.exe not found; it creates the Start menu shortcut")
	}
	icon := iconPath(e)
	if err := writeFile(icon, string(iconICO), 0o644); err != nil {
		return "", err
	}
	script := strings.Join([]string{
		"$ErrorActionPreference = 'Stop'",
		"$s = (New-Object -ComObject WScript.Shell).CreateShortcut(" + psQuote(path) + ")",
		"$s.TargetPath = " + psQuote(target),
		"$s.Arguments = " + psQuote(args),
		"$s.WorkingDirectory = " + psQuote(e.Home),
		"$s.Description = " + psQuote(description),
		"$s.IconLocation = " + psQuote(icon+",0"),
		"$s.Save()",
	}, "; ")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := e.Run(exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", script)); err != nil {
		return "", fmt.Errorf("create the shortcut: %w", err)
	}
	return path, nil
}

// psQuote quotes s as a PowerShell single-quoted string.
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// installLinux writes a desktop entry; the desktop environment picks the terminal for Terminal=true.
func installLinux(e Env) (string, error) {
	path, _ := entryPath(e)
	icon := iconPath(e)
	if err := writeFile(icon, string(iconPNG), 0o644); err != nil {
		return "", err
	}
	entry := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=roundhouse\n" +
		"Comment=" + description + "\n" +
		"Exec=" + desktopExecQuote(e.Exe) + "\n" +
		"Icon=" + icon + "\n" +
		"Terminal=true\n" +
		"Categories=Utility;Development;\n" +
		"Keywords=claude;chat;resume;\n"
	return path, writeFile(path, entry, 0o644)
}

// desktopExecQuote quotes a program path for the Exec key of a desktop entry (always in double
// quotes; the characters that stay special inside them are escaped, then % is doubled).
func desktopExecQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\\\`, `"`, `\"`, "`", "\\`", `$`, `\$`, `%`, `%%`)
	return `"` + r.Replace(s) + `"`
}

const darwinLauncher = "roundhouse-launcher"

// installDarwin writes an application bundle whose launcher opens Terminal and runs roundhouse in it.
func installDarwin(e Env) (string, error) {
	path, _ := entryPath(e)
	script := `tell application "Terminal" to activate` + "\n" +
		`tell application "Terminal" to do script "exec ` + appleScriptEscape(shellQuote(e.Exe)) + `"`
	launcher := "#!/bin/sh\nexec osascript -e " + shellQuote(script) + "\n"
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>roundhouse</string>
	<key>CFBundleDisplayName</key><string>roundhouse</string>
	<key>CFBundleIdentifier</key><string>io.github.lugg000.roundhouse</string>
	<key>CFBundleIconFile</key><string>roundhouse</string>
	<key>CFBundleExecutable</key><string>` + darwinLauncher + `</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>LSUIElement</key><true/>
</dict>
</plist>
`
	if err := writeFile(filepath.Join(path, "Contents", "Info.plist"), plist, 0o644); err != nil {
		return "", err
	}
	if err := writeFile(filepath.Join(path, "Contents", "Resources", appName+".icns"), string(iconICNS), 0o644); err != nil {
		return "", err
	}
	return path, writeFile(filepath.Join(path, "Contents", "MacOS", darwinLauncher), launcher, 0o755)
}

// shellQuote quotes s for a POSIX shell.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// appleScriptEscape escapes s for use inside an AppleScript string literal.
func appleScriptEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}
