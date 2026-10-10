// Package config loads roundhouse's config.toml.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/LuGG000/roundhouse/internal/paths"
	"github.com/LuGG000/roundhouse/internal/starter"
)

const fileName = "config.toml"

// Config holds the user-editable settings.
type Config struct {
	// Starter names the starter used to open sessions: a built-in or one
	// of Starters. Empty means claude.
	Starter string `toml:"starter"`
	// Starters adds custom starters or overrides built-in ones by name.
	Starters []starter.Starter `toml:"starters"`
}

// Default returns the configuration used when no file exists.
func Default() Config {
	return Config{Starter: starter.Claude}
}

// Load reads the config file; a missing file yields the defaults.
func Load() (Config, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return Config{}, err
	}
	cfg := Default()
	b, err := os.ReadFile(filepath.Join(dir, fileName))
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", fileName, err)
	}
	return cfg, nil
}

// ActiveStarter resolves the configured starter.
func (c Config) ActiveStarter() (starter.Starter, error) {
	name := c.Starter
	if name == "" {
		name = starter.Claude
	}
	for _, s := range c.Starters {
		if s.Name == name {
			return s, nil
		}
	}
	for _, s := range starter.Builtins() {
		if s.Name == name {
			return s, nil
		}
	}
	return starter.Starter{}, fmt.Errorf("unknown starter %q", name)
}
