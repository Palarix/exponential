package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/palarix/beats/internal/config"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current identity and server connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := config.LoadRemoteConfig()
		if rc.URL == "" {
			fmt.Printf("Mode:     local\n")
			if cfg != nil && cfg.User != "" {
				fmt.Printf("Identity: %s\n", cfg.User)
			}
			return nil
		}

		fmt.Printf("Mode:     remote\n")
		fmt.Printf("Server:   %s\n", rc.URL)

		if rc.Token == "" {
			fmt.Println("Token:    (none)")
			return nil
		}

		parts := strings.SplitN(rc.Token, ".", 3)
		if len(parts) != 3 {
			fmt.Println("Token:    (invalid)")
			return nil
		}

		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			fmt.Println("Token:    (unreadable)")
			return nil
		}

		var claims struct {
			Sub string `json:"sub"`
			Exp int64  `json:"exp"`
		}
		if err := json.Unmarshal(payload, &claims); err != nil {
			fmt.Println("Token:    (unreadable)")
			return nil
		}

		fmt.Printf("Identity: %s\n", claims.Sub)

		expiry := time.Unix(claims.Exp, 0)
		if time.Now().After(expiry) {
			fmt.Printf("Token:    expired (%s)\n", expiry.Format(time.RFC3339))
			fmt.Println("Run 'beats login' to re-authenticate.")
		} else {
			fmt.Printf("Expires:  %s\n", expiry.Format(time.RFC3339))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
