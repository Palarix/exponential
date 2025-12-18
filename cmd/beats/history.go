package main

import (
	"fmt"
	"os"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history [id]",
	Short: "Show audit trail for an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		events, err := storage.ReadEvents()
		if err != nil {
			fmt.Printf("Error reading events: %v\n", err)
			os.Exit(1)
		}

		issues := model.ProjectIssues(events)
		issue, exists := issues[id]
		if !exists {
			fmt.Printf("Issue %s not found\n", id)
			os.Exit(1)
		}

		renderHistory(issue.Events)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
