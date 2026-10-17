package main

import (
	"fmt"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the identity xpo records on events",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Identity: %s\n", whoamiIdentity(cfg))
		return nil
	},
}

// whoamiIdentity returns the author xpo records on new events: the
// configured user, else git config. cfg is nil outside a project.
func whoamiIdentity(c *config.Config) string {
	if c == nil {
		c = &config.Config{}
	}
	return exponential.NewClient(c).GetUser()
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
