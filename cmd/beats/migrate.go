package main

import (
	"fmt"
	"os"

	"github.com/kuyio/beats/internal/beats"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data to new schema version",
	Run: func(cmd *cobra.Command, args []string) {
		currentVersion := 0
		if cfg != nil {
			currentVersion = cfg.Version
		}

		newVersion, err := beats.RunMigrations(currentVersion)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if newVersion == currentVersion {
			fmt.Println("Already up to date.")
		} else {
			fmt.Printf("Migrated to v%d.\n", newVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
