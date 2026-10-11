package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ciao22king/GoFetch/internal/core"
)

// LoadConfig reads the user's config file from ~/.config/gofetch/config.json.
// Any problem (missing file, invalid JSON, unknown location) degrades to the
// zero Config instead of failing: the app must always start with defaults.
func LoadConfig() core.Config {
	dir, err := core.DefaultStoreDir()
	if err != nil {
		return core.Config{}
	}
	return core.LoadConfig(dir)
}

// SaveConfig persists the user's preferences next to the history file.
func SaveConfig(cfg core.Config) error {
	dir, err := core.DefaultStoreDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("non riesco a creare %s: %w", dir, err)
	}
	return core.SaveConfig(dir, cfg)
}

// defaultHistoryPath is used by headless runs and tests; the TUI keeps its own
// *core.Store instance for cached access.
func defaultHistoryPath() (string, error) {
	dir, err := core.DefaultStoreDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}
