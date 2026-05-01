package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadRaw reads the JSON config file as a generic map. If the file does not
// exist, it returns an empty map (not an error) so read-only operations can
// succeed against a not-yet-created config.
func ReadRaw(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return raw, nil
}

// ReadOrCreateRaw is like ReadRaw but writes the default scaffold to disk
// when the file is missing, so subsequent writes have a valid base.
func ReadOrCreateRaw(path string) (map[string]any, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("creating config directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return nil, fmt.Errorf("writing default config: %w", err)
		}
	}
	return ReadRaw(path)
}

// WriteRaw serializes raw as pretty-printed JSON and writes it to path.
func WriteRaw(path string, raw map[string]any) error {
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// SetRawValue sets the value at the given dot-separated path in raw,
// creating intermediate maps as needed.
func SetRawValue(raw map[string]any, name string, value any) {
	parts := strings.Split(name, ".")
	cur := raw
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			return
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
}

// GetRawValue looks up a dot-separated path in raw.
func GetRawValue(raw map[string]any, name string) (any, bool) {
	parts := strings.Split(name, ".")
	var cur any = raw
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, present := m[p]
		if !present {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// UnsetRawValue removes the value at name. Returns true if a value was removed.
func UnsetRawValue(raw map[string]any, name string) bool {
	parts := strings.Split(name, ".")
	cur := raw
	for i, p := range parts {
		if i == len(parts)-1 {
			if _, present := cur[p]; !present {
				return false
			}
			delete(cur, p)
			return true
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			return false
		}
		cur = next
	}
	return false
}

// ReadEnabled returns the currently enabled entity list as recorded in the
// config file, plus a boolean indicating whether the "entities" key was
// explicitly present. When absent (or the file is missing), returns
// (default-all-enabled, false).
func ReadEnabled(path string) ([]string, bool, error) {
	raw, err := ReadRaw(path)
	if err != nil {
		return nil, false, err
	}
	v, present := raw["entities"]
	if !present {
		return defaultEnabledNames(), false, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, true, fmt.Errorf("config entities key is not a JSON array")
	}
	out := make([]string, 0, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, true, fmt.Errorf("config entities[%d] is not a string", i)
		}
		out = append(out, s)
	}
	return out, true, nil
}

// WriteEnabled replaces the "entities" key in the config file with names,
// preserving all other keys. The file (with default scaffold) is created
// if it does not yet exist.
func WriteEnabled(path string, names []string) error {
	raw, err := ReadOrCreateRaw(path)
	if err != nil {
		return err
	}
	values := make([]any, len(names))
	for i, n := range names {
		values[i] = n
	}
	raw["entities"] = values
	return WriteRaw(path, raw)
}

func defaultEnabledNames() []string {
	out := make([]string, len(Available))
	for i, e := range Available {
		out[i] = e.Name
	}
	return out
}
