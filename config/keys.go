package config

import (
	"fmt"
	"strings"
	"time"
)

// Kind classifies how a config value is parsed and validated.
type Kind int

const (
	KindString Kind = iota
	KindBool
	KindDuration
	KindEnum
)

// Key describes a single settable config key.
type Key struct {
	Name        string // dot-separated JSON path, e.g. "mqtt.broker"
	Description string
	Kind        Kind
	EnumValues  []string // populated for KindEnum
	Sensitive   bool     // value is masked in `config show`
}

// Keys is the catalog of every key the CLI can set, get, or unset.
// "entities" is intentionally omitted — managed via the entities subcommand.
var Keys = []Key{
	{Name: "mqtt.broker", Description: `MQTT broker URL, e.g. "tcp://192.168.1.100:1883"`, Kind: KindString},
	{Name: "mqtt.username", Description: "MQTT username", Kind: KindString},
	{Name: "mqtt.password", Description: "MQTT password", Kind: KindString, Sensitive: true},
	{Name: "mqtt.client_id", Description: `MQTT client ID (default "micdetector-<hostname>")`, Kind: KindString},
	{Name: "mqtt.topic_prefix", Description: `MQTT topic prefix (default "micdetector")`, Kind: KindString},
	{Name: "hostname", Description: "Hostname used in topics and HA names (default: system hostname)", Kind: KindString},
	{Name: "poll_interval", Description: `Polling interval as Go duration, e.g. "2s"`, Kind: KindDuration},
	{Name: "homeassistant_discovery", Description: "Publish Home Assistant MQTT discovery configs", Kind: KindBool},
	{Name: "log_level", Description: "Logging verbosity", Kind: KindEnum, EnumValues: []string{"debug", "info", "warn", "error"}},
}

// FindKey looks up a key by name.
func FindKey(name string) (Key, bool) {
	for _, k := range Keys {
		if k.Name == name {
			return k, true
		}
	}
	return Key{}, false
}

// ParseValue parses a string into the JSON-friendly value appropriate for
// this key's Kind. Strings (including durations) are stored as strings;
// booleans become bool.
func (k Key) ParseValue(s string) (any, error) {
	switch k.Kind {
	case KindString:
		return s, nil
	case KindBool:
		switch strings.ToLower(s) {
		case "true", "yes", "on", "1":
			return true, nil
		case "false", "no", "off", "0":
			return false, nil
		}
		return nil, fmt.Errorf("invalid boolean %q (use true or false)", s)
	case KindDuration:
		if _, err := time.ParseDuration(s); err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		return s, nil
	case KindEnum:
		for _, v := range k.EnumValues {
			if v == s {
				return s, nil
			}
		}
		return nil, fmt.Errorf("invalid value %q (allowed: %s)", s, strings.Join(k.EnumValues, ", "))
	}
	return nil, fmt.Errorf("unknown kind for key %q", k.Name)
}

// KindLabel returns a human-readable type label for use in help text.
func (k Key) KindLabel() string {
	switch k.Kind {
	case KindString:
		return "string"
	case KindBool:
		return "bool"
	case KindDuration:
		return "duration"
	case KindEnum:
		return "one of: " + strings.Join(k.EnumValues, ", ")
	}
	return "?"
}
