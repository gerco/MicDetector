//go:build windows

package config

import (
	"os"
	"path/filepath"
)

// DefaultConfigPath returns %APPDATA%\MicDetector\config.json.
func DefaultConfigPath() string {
	// Use APPDATA environment variable for roaming app data
	appData := os.Getenv("APPDATA")
	if appData == "" {
		// Fallback to UserHomeDir\AppData\Roaming
		home, err := os.UserHomeDir()
		if err != nil {
			return "config.json"
		}
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	return filepath.Join(appData, "MicDetector", "config.json")
}
