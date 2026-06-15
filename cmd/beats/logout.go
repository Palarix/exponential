package main

import (
	"fmt"

	"github.com/palarix/beats/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout [server-url]",
	Short: "Remove cached remote server credentials",
	Long: `Remove cached authentication token for a remote beats server.

Without arguments, removes credentials for the server configured in
the current project. With a URL argument, removes credentials for
that specific server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var serverURL string
		if len(args) > 0 {
			serverURL = args[0]
		} else if cfg != nil && cfg.Remote.URL != "" {
			serverURL = cfg.Remote.URL
		} else {
			creds := config.LoadUserConfig()
			if len(creds.Servers) == 0 {
				fmt.Println("Not logged in to any server.")
				return nil
			}
			for _, cred := range creds.Servers {
				serverURL = cred.URL
				break
			}
		}

		if err := config.RemoveServerCredential(serverURL); err != nil {
			return fmt.Errorf("failed to clear credentials: %w", err)
		}
		fmt.Printf("Logged out from %s\n", serverURL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
