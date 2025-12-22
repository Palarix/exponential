package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var historyCmd = &cobra.Command{
	Use:               "history [id]",
	Short:             "Show audit trail for an issue",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		client := beats.NewClient(cfg)

		issue, _, archived, err := client.FindIssue(id)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if archived {
			fmt.Println("Note: This issue is archived.")
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

		ui.RenderHistory(issue.Events, termWidth)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
