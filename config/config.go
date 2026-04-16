//go:build darwin || windows

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MQTTConfig holds MQTT connection settings.
type MQTTConfig struct {
	Broker      string `json:"broker"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	ClientID    string `json:"client_id"`
	TopicPrefix string `json:"topic_prefix"`
}

// Config holds the full application configuration.
type Config struct {
	MQTT                   MQTTConfig `json:"mqtt"`
	Hostname               string     `json:"hostname"`
	PollInterval           string     `json:"poll_interval"`
	HomeAssistantDiscovery bool       `json:"homeassistant_discovery"`
	LogLevel               string     `json:"log_level"`

	// Parsed poll interval (not from JSON).
	PollDuration time.Duration `json:"-"`
	// Serial number of the device, used as a stable device identifier.
	SerialNumber string `json:"-"`
}

// ErrNotConfigured is returned when the config file was just created and
// the user needs to edit it before the application can start.
var ErrNotConfigured = errors.New("not configured")

const defaultConfig = `{
  "mqtt": {
    "broker": "",
    "username": "",
    "password": ""
  },
  "homeassistant_discovery": false
}
`

// Load reads the config file at path and returns a Config with defaults applied.
// If the file does not exist, it creates a default config and returns ErrNotConfigured.
// If the file exists but mqtt.broker is empty, it also returns ErrNotConfigured.
func Load(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("creating config directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return nil, fmt.Errorf("writing default config: %w", err)
		}
		return nil, ErrNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.MQTT.Broker == "" {
		return nil, ErrNotConfigured
	}

	// Apply defaults.
	if cfg.Hostname == "" {
		cfg.Hostname, err = os.Hostname()
		if err != nil {
			cfg.Hostname = "unknown"
		}
	}

	if cfg.MQTT.ClientID == "" {
		cfg.MQTT.ClientID = "micdetector-" + cfg.Hostname
	}

	if cfg.MQTT.TopicPrefix == "" {
		cfg.MQTT.TopicPrefix = "micdetector"
	}

	if cfg.PollInterval == "" {
		cfg.PollInterval = "2s"
	}

	cfg.PollDuration, err = time.ParseDuration(cfg.PollInterval)
	if err != nil {
		return nil, fmt.Errorf("parsing poll_interval %q: %w", cfg.PollInterval, err)
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	cfg.SerialNumber = macSerialNumber()

	return cfg, nil
}
