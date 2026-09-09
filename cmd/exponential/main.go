package main

import (
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/ui"
	"github.com/palarix/exponential/internal/version"
	"github.com/spf13/cobra"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "xpo",
	Short: "The git-native engineering system for human-AI teams",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			// If config fails to load, we only allow certain commands (and their subcommands)
			allowed := []string{"init", "help", "version", "doctor", "login", "logout", "whoami", "demo"}
			if commandOrAncestorAllowed(cmd, allowed) {
				return nil
			}
			return err
		}
		config.Set(cfg)

		if cfg.Labels != nil {
			ui.SetLabelColors(cfg.Labels)
		}

		// Check Data Model Version
		// Exceptions: commands that don't need strict version match or are used to fix it
		exceptions := []string{"init", "help", "version", "doctor", "migrate", "login", "logout", "whoami", "serve", "demo"}
		if commandOrAncestorAllowed(cmd, exceptions) {
			return nil
		}

		if cfg.Version < version.DataModelVersion {
			return fmt.Errorf("xpo data model version mismatch (config: v%d, cli expects: v%d).\nRun 'xpo migrate' to update your project.", cfg.Version, version.DataModelVersion)
		}
		if cfg.Version > version.DataModelVersion {
			return fmt.Errorf("xpo data model version mismatch (config: v%d, cli expects: v%d).\nYour CLI version is too old. Please upgrade exponential.", cfg.Version, version.DataModelVersion)
		}

		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// commandOrAncestorAllowed checks whether the command or any of its ancestors
// is in the allowed list. This ensures subcommands of allowed commands (e.g.
// "xpo init mcp", "xpo init skill") are also allowed.
func commandOrAncestorAllowed(cmd *cobra.Command, allowed []string) bool {
	for c := cmd; c != nil; c = c.Parent() {
		for _, a := range allowed {
			if c.Name() == a {
				return true
			}
		}
	}
	return false
}
