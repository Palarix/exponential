package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current identity and server connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if the current project has a remote configured
		var serverURL string
		if cfg != nil && cfg.Remote.URL != "" {
			serverURL = cfg.Remote.URL
		}

		if serverURL == "" {
			fmt.Printf("Mode:     local\n")
			if cfg != nil && cfg.User != "" {
				fmt.Printf("Identity: %s\n", cfg.User)
			}
			printPermissions(cfg, "")
			return nil
		}

		cred, ok := config.GetServerCredential(serverURL)
		if !ok || cred.Token == "" {
			fmt.Printf("Mode:     remote\n")
			fmt.Printf("Server:   %s\n", serverURL)
			fmt.Println("Token:    (not logged in)")
			fmt.Println("Run 'xpo login' to authenticate.")
			return nil
		}

		fmt.Printf("Mode:     remote\n")
		fmt.Printf("Server:   %s\n", serverURL)

		parts := strings.SplitN(cred.Token, ".", 3)
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
			fmt.Println("Run 'xpo login' to re-authenticate.")
		} else {
			fmt.Printf("Expires:  %s\n", expiry.Format(time.RFC3339))
		}

		// Extract email for permission lookup
		email := ""
		if idx := strings.LastIndex(claims.Sub, "<"); idx != -1 {
			if end := strings.LastIndex(claims.Sub, ">"); end > idx {
				email = strings.ToLower(strings.TrimSpace(claims.Sub[idx+1 : end]))
			}
		}
		printPermissions(cfg, email)

		return nil
	},
}

func printPermissions(cfg *config.Config, email string) {
	if cfg == nil || !cfg.Permissions.Enabled() {
		return
	}
	perms := cfg.Permissions
	role := perms.RoleFor(email)
	if role == "" {
		fmt.Printf("Role:     (none — no access)\n")
		return
	}
	fmt.Printf("Role:     %s\n", role)
	caps := perms.Capabilities(role)
	fmt.Printf("Can:      %s\n", strings.Join(caps, ", "))
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
