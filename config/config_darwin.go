//go:build darwin

package config

import (
	"os"
	"path/filepath"
)

// DefaultConfigPath returns ~/Library/Application Support/MicDetector/config.json.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, "Library", "Application Support", "MicDetector", "config.json")
}
