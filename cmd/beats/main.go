package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/config"
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
			return err
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
