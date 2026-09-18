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
	Short: ui.Tagline,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
			printNotAProject()
			return
		}
		cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
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

		exceptions := []string{"init", "help", "version", "doctor", "migrate", "login", "logout", "whoami", "serve", "demo"}
		if commandOrAncestorAllowed(cmd, exceptions) {
			return nil
		}

		if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
			if argsContainJSONFlag() {
				return fmt.Errorf("not an exponential project (no .xpo directory)")
			}
			printNotAProject()
			os.Exit(0)
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
	jsonMode := argsContainJSONFlag()
	if jsonMode {
		rootCmd.SilenceErrors = true
		rootCmd.SilenceUsage = true
	}
	if err := rootCmd.Execute(); err != nil {
		if jsonMode {
			exitJSONError(err)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func argsContainJSONFlag() bool {
	for _, a := range os.Args[1:] {
		if a == "--json" || a == "--json=true" {
			return true
		}
		if a == "--" {
			return false
		}
	}
	return false
}

func printNotAProject() {
	fmt.Printf("\n%s\n", ui.Banner())
	fmt.Printf("\nThis directory isn't an Exponential project yet.\n\n")
	fmt.Printf("  xpo init    Set up Exponential here\n")
	fmt.Printf("  xpo help    Show all commands\n\n")
}

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
