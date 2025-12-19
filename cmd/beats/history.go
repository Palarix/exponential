package main

import (
	"fmt"
	"os"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

		// Detect Terminal Width
		termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || termWidth <= 0 {
			termWidth = 100 // Fallback
		}

		// Apply safety margin and cap for readability
		termWidth -= 4
		if termWidth > 116 {
			termWidth = 116 // Total effective width including padding
		}

		renderHistory(issue.Events, termWidth)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
