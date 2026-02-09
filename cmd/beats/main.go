package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/version"
	"github.com/spf13/cobra"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "beats",
	Short: "A JSONL-based issue tracker",
	Long:  `Beats is a JSONL-based issue tracker that is committed to Git together with your project.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			// If config fails to load, we only allow certain commands
			allowed := []string{"init", "help", "version", "doctor"}
			for _, a := range allowed {
				if cmd.Name() == a {
					return nil
				}
			}
			return err
		}

		// Check Data Model Version
		// Exceptions: commands that don't need strict version match or are used to fix it
		exceptions := []string{"init", "help", "version", "doctor", "migrate"}
		for _, ex := range exceptions {
			if cmd.Name() == ex {
				return nil
			}
		}

		if cfg.Version < version.DataModelVersion {
			return fmt.Errorf("beats data model version mismatch (config: v%d, cli expects: v%d).\nRun 'beats migrate' to update your project.", cfg.Version, version.DataModelVersion)
		}
		if cfg.Version > version.DataModelVersion {
			return fmt.Errorf("beats data model version mismatch (config: v%d, cli expects: v%d).\nYour CLI version is too old. Please upgrade beats.", cfg.Version, version.DataModelVersion)
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
