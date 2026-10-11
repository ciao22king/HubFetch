package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config holds user preferences stored next to the history file. Every field
// is optional: a zero value means "fall back to the built-in default", so a
// partial or missing config file is always valid.
type Config struct {
	// DefaultDir is the parent directory suggested for new clones. The -dir
	// flag overrides it.
	DefaultDir string `json:"default_dir"`
	// Depth is the default shallow-clone depth (0 = full clone). The -depth
	// flag overrides it.
	Depth int `json:"depth"`
	// NoBuild disables the automatic build after cloning unless -no-build is
	// passed explicitly.
	NoBuild bool `json:"no_build"`
}

// LoadConfig reads the config file from dir/config.json. Any problem (missing
// file, invalid JSON) degrades to the zero Config instead of failing: the app
// must keep working with defaults.
func LoadConfig(dir string) Config {
	var cfg Config
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return cfg
	}
	if json.Unmarshal(data, &cfg) != nil {
		return Config{}
	}
	return cfg.sanitized()
}

// SaveConfig writes the config atomically, reusing the same temp-file + rename
// strategy as the history so a crash cannot corrupt preferences.
func SaveConfig(dir string, cfg Config) error {
	data, err := json.MarshalIndent(cfg.sanitized(), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeAtomic(dir, "config-*.tmp", filepath.Join(dir, "config.json"), data)
}

// sanitized normalizes values that would otherwise cause confusing behavior:
// a negative depth is treated as "full clone" and the directory is trimmed.
func (c Config) sanitized() Config {
	if c.Depth < 0 {
		c.Depth = 0
	}
	c.DefaultDir = strings.TrimSpace(c.DefaultDir)
	return c
}
