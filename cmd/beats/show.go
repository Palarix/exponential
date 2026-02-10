package main

import (
	"fmt"
	"os"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var showCmd = &cobra.Command{
	Use:               "show [id]",
	Short:             "Show issue details and history",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		showIssue(args[0])
	},
}

func showIssue(id string) {
	client := beats.NewClient(cfg)
	issue, children, archived, err := client.FindIssue(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Detect Terminal Width
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termWidth <= 0 {
		termWidth = 100 // Fallback
	}

	fmt.Println(ui.RenderIssueDetails(issue, children, archived, termWidth))
}

func init() {
	rootCmd.AddCommand(showCmd)
}
