package main

import (
	"fmt"

	"github.com/palarix/beats/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove cached remote server credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := config.LoadRemoteConfig()
		if rc.URL == "" {
			fmt.Println("Not logged in to any server.")
			return nil
		}
		if err := config.ClearRemoteConfig(); err != nil {
			return fmt.Errorf("failed to clear credentials: %w", err)
		}
		fmt.Printf("Logged out from %s\n", rc.URL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
