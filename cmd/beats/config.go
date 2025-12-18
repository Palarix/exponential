package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Displays the current configuration loaded from files and environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		if cfg == nil {
			fmt.Println("Config not loaded")
			return
		}

		// Marshal config to YAML for display
		enc := yaml.NewEncoder(os.Stdout)
		enc.SetIndent(2)
		if err := enc.Encode(cfg); err != nil {
			fmt.Println("Error displaying config:", err)
		}
		enc.Close()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
