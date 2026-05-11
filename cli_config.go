package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"micdetector/config"
)

func newConfigCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show, get, set, or unset config values",
		Long:  "Show, get, set, or unset MicDetector config values.\n\n" + configKeysText(),
	}
	cmd.AddCommand(
		newConfigShowCmd(configPath),
		newConfigGetCmd(configPath),
		newConfigSetCmd(configPath),
		newConfigUnsetCmd(configPath),
	)
	return cmd
}

func configKeysText() string {
	var b strings.Builder
	b.WriteString("Available keys:\n")
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, k := range config.Keys {
		fmt.Fprintf(tw, "  %s\t(%s)\t%s\n", k.Name, k.KindLabel(), k.Description)
	}
	tw.Flush()
	b.WriteString("\nThe entities array is managed via 'micdetector entities enable/disable'.\n")
	return b.String()
}

func newConfigShowCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the current config (passwords masked)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := config.ReadRaw(*configPath)
			if err != nil {
				return err
			}
			maskSensitive(raw)
			out, err := json.MarshalIndent(raw, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
}

func newConfigGetCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY",
		Short: "Print the value of a single config key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := requireKnownKey(name); err != nil {
				return err
			}
			raw, err := config.ReadRaw(*configPath)
			if err != nil {
				return err
			}
			v, ok := config.GetRawValue(raw, name)
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "(not set)")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), formatValue(v))
			return nil
		},
	}
}

func newConfigSetCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a config value",
		Long:  "Set a config value.\n\n" + configKeysText(),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, raw := args[0], args[1]
			key, err := lookupKey(name)
			if err != nil {
				return err
			}
			value, err := key.ParseValue(raw)
			if err != nil {
				return err
			}
			rawCfg, err := config.ReadOrCreateRaw(*configPath)
			if err != nil {
				return err
			}
			config.SetRawValue(rawCfg, name, value)
			if err := config.WriteRaw(*configPath, rawCfg); err != nil {
				return err
			}
			displayed := formatValue(value)
			if key.Sensitive && raw != "" {
				displayed = "***"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set %s = %s\n", name, displayed)
			fmt.Fprintln(cmd.OutOrStdout(), "Restart the service to apply: brew services restart micdetector")
			return nil
		},
	}
}

func newConfigUnsetCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "unset KEY",
		Short: "Remove a config key (revert to its default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if _, err := lookupKey(name); err != nil {
				return err
			}
			rawCfg, err := config.ReadOrCreateRaw(*configPath)
			if err != nil {
				return err
			}
			if !config.UnsetRawValue(rawCfg, name) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s was not set\n", name)
				return nil
			}
			if err := config.WriteRaw(*configPath, rawCfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "unset %s\n", name)
			fmt.Fprintln(cmd.OutOrStdout(), "Restart the service to apply: brew services restart micdetector")
			return nil
		},
	}
}

func lookupKey(name string) (config.Key, error) {
	if name == "entities" {
		return config.Key{}, fmt.Errorf("the entities array is managed via 'micdetector entities enable/disable'")
	}
	key, ok := config.FindKey(name)
	if !ok {
		return config.Key{}, fmt.Errorf("unknown config key %q\n\n%s", name, configKeysText())
	}
	return key, nil
}

func requireKnownKey(name string) error {
	_, err := lookupKey(name)
	return err
}

func formatValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		out, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(out)
	}
}

// maskSensitive walks the catalog and replaces any non-empty string value
// flagged as sensitive with "***" in the raw map. Other values are untouched.
func maskSensitive(raw map[string]any) {
	for _, k := range config.Keys {
		if !k.Sensitive {
			continue
		}
		v, ok := config.GetRawValue(raw, k.Name)
		if !ok {
			continue
		}
		if s, isString := v.(string); isString && s != "" {
			config.SetRawValue(raw, k.Name, "***")
		}
	}
}
