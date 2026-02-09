package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/beats"
	"github.com/spf13/cobra"
)

var migrateDryRun bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate old data to new schema (e.g. ParentID -> Dependencies)",
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)

		report, err := client.RunMigrations(beats.MigrateOptions{
			DryRun: migrateDryRun,
		})
		if err != nil {
			fmt.Printf("Error running migration: %v\n", err)
			os.Exit(1)
		}

		if len(report) == 0 {
			fmt.Println("No migration needed.")
		} else {
			for _, line := range report {
				fmt.Println(line)
			}
			if migrateDryRun {
				fmt.Println("\n(Dry Run - no changes persisted)")
			} else {
				fmt.Println("\nMigration completed successfully.")
			}
		}
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Preview migration without applying changes")
	rootCmd.AddCommand(migrateCmd)
}
