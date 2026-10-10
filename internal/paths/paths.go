// Package paths locates the directories roundhouse reads and writes.
package paths

import (
	"os"
	"path/filepath"
)

const (
	configDirEnv = "ROUNDHOUSE_CONFIG_DIR"
	dataDirEnv   = "ROUNDHOUSE_DATA_DIR"
	claudeDirEnv = "CLAUDE_CONFIG_DIR"
	appName      = "roundhouse"
)

// ClaudeDir is the claude config dir: CLAUDE_CONFIG_DIR, else ~/.claude.
func ClaudeDir() (string, error) {
	if d := os.Getenv(claudeDirEnv); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// ProjectsDir holds the session transcripts, one folder per project.
func ProjectsDir() (string, error) {
	d, err := ClaudeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "projects"), nil
}

// ConfigDir is ROUNDHOUSE_CONFIG_DIR, else <UserConfigDir>/roundhouse.
func ConfigDir() (string, error) {
	if d := os.Getenv(configDirEnv); d != "" {
		return d, nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, appName), nil
}

// DataDir is ROUNDHOUSE_DATA_DIR, else $XDG_DATA_HOME/roundhouse, else
// %LocalAppData%\roundhouse, else ~/.local/share/roundhouse.
func DataDir() (string, error) {
	if d := os.Getenv(dataDirEnv); d != "" {
		return d, nil
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, appName), nil
	}
	if d := os.Getenv("LocalAppData"); d != "" {
		return filepath.Join(d, appName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", appName), nil
}
