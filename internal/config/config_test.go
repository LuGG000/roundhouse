package config

import (
	"os"
	"path/filepath"
	"testing"
)

func load(t *testing.T, toml string) (Config, error) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ROUNDHOUSE_CONFIG_DIR", dir)
	if toml != "" {
		if err := os.WriteFile(filepath.Join(dir, fileName), []byte(toml), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return Load()
}

func TestMissingFileGivesDefaults(t *testing.T) {
	cfg, err := load(t, "")
	if err != nil {
		t.Fatal(err)
	}
	s, err := cfg.ActiveStarter()
	if err != nil || s.Name != "claude" {
		t.Errorf("starter = %+v, %v; want claude", s, err)
	}
}

func TestCustomStarterOverridesBuiltin(t *testing.T) {
	cfg, err := load(t, `
starter = "claude"

[[starters]]
name = "claude"
resume = ["wrapper", "{id}"]
new = ["wrapper"]
`)
	if err != nil {
		t.Fatal(err)
	}
	s, err := cfg.ActiveStarter()
	if err != nil || s.Resume[0] != "wrapper" {
		t.Errorf("starter = %+v, %v; want the custom one", s, err)
	}
}

func TestUnknownStarter(t *testing.T) {
	cfg, err := load(t, `starter = "nope"`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.ActiveStarter(); err == nil {
		t.Error("want an error for an unknown starter")
	}
}

func TestInvalidTOML(t *testing.T) {
	if _, err := load(t, "starter = ["); err == nil {
		t.Error("want an error for invalid TOML")
	}
}

func TestUpdateCheckIsOnUnlessSwitchedOff(t *testing.T) {
	cfg, err := load(t, "")
	if err != nil || !cfg.UpdateCheck {
		t.Fatalf("default: %+v, %v; want the update check on", cfg, err)
	}
	cfg, err = load(t, "update_check = false")
	if err != nil || cfg.UpdateCheck {
		t.Fatalf("switched off: %+v, %v", cfg, err)
	}
	cfg, err = load(t, `starter = "claude"`)
	if err != nil || !cfg.UpdateCheck {
		t.Fatalf("a file without the key keeps the default: %+v, %v", cfg, err)
	}
}
