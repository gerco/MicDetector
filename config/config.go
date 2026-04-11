package config

import (
	"encoding/json"
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
	// Serial number of the Mac, used as a stable device identifier.
	SerialNumber string `json:"-"`
}

// DefaultConfigPath returns ~/Library/Application Support/MicDetector/config.json.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, "Library", "Application Support", "MicDetector", "config.json")
}

// Load reads the config file at path and returns a Config with defaults applied.
func Load(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.MQTT.Broker == "" {
		return nil, fmt.Errorf("mqtt.broker is required")
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
