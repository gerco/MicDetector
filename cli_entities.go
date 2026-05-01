package main

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"micdetector/config"
)

func newEntitiesCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entities",
		Short: "Manage which entities are published",
		Long:  "Inspect or change which entities MicDetector publishes.\n\n" + entityCatalogText(),
	}
	cmd.AddCommand(
		newEntitiesListCmd(configPath),
		newEntitiesEnableCmd(configPath),
		newEntitiesDisableCmd(configPath),
	)
	return cmd
}

func entityCatalogText() string {
	var b strings.Builder
	b.WriteString("Available entities:\n")
	for _, e := range config.Available {
		fmt.Fprintf(&b, "  %-13s  %s\n", e.Name, e.Description)
	}
	return b.String()
}

func newEntitiesListCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List entities and whether each is enabled",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			enabled, _, err := config.ReadEnabled(*configPath)
			if err != nil {
				return err
			}
			set := map[string]bool{}
			for _, n := range enabled {
				set[n] = true
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, e := range config.Available {
				state := "disabled"
				if set[e.Name] {
					state = "enabled"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n", e.Name, state, e.Description)
			}
			return tw.Flush()
		},
	}
}

func newEntitiesEnableCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "enable ENTITY [ENTITY...]",
		Short: "Enable one or more entities",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateEntities(cmd, *configPath, args, true)
		},
	}
}

func newEntitiesDisableCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "disable ENTITY [ENTITY...]",
		Short: "Disable one or more entities",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateEntities(cmd, *configPath, args, false)
		},
	}
}

func updateEntities(cmd *cobra.Command, configPath string, names []string, enable bool) error {
	for _, name := range names {
		if !config.IsKnownEntity(name) {
			return fmt.Errorf("unknown entity %q\n\n%s", name, entityCatalogText())
		}
	}

	current, _, err := config.ReadEnabled(configPath)
	if err != nil {
		return err
	}

	set := map[string]bool{}
	for _, n := range current {
		set[n] = true
	}
	for _, n := range names {
		set[n] = enable
	}

	// Preserve catalog order so the file stays predictable.
	next := make([]string, 0, len(config.Available))
	for _, e := range config.Available {
		if set[e.Name] {
			next = append(next, e.Name)
		}
	}
	sort.SliceStable(next, func(i, j int) bool {
		return catalogIndex(next[i]) < catalogIndex(next[j])
	})

	if err := config.WriteEnabled(configPath, next); err != nil {
		return err
	}

	verb := "enabled"
	if !enable {
		verb = "disabled"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", verb, strings.Join(names, ", "))
	fmt.Fprintln(cmd.OutOrStdout(), "Restart the service to apply: brew services restart micdetector")
	return nil
}

func catalogIndex(name string) int {
	for i, e := range config.Available {
		if e.Name == name {
			return i
		}
	}
	return len(config.Available)
}
