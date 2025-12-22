package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/beats/internal/beats"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	deleteForceFlag  bool
	deleteReasonFlag string
)

var deleteCmd = &cobra.Command{
	Use:               "delete [id]",
	Aliases:           []string{"rm"},
	Short:             "Delete an issue",
	Long:              `Delete an issue from the board. The issue will no longer appear in lists or reports.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		client := beats.NewClient(cfg)

		// 1. Get Issue Details for Confirmation
		issue, err := client.GetIssue(id)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// 2. Confirmation prompt (unless --force)
		if !deleteForceFlag {
			fmt.Printf("About to delete issue:\n")
			fmt.Printf("  ID:     %s\n", issue.ID)
			fmt.Printf("  Title:  %s\n", issue.Title)
			fmt.Printf("  Status: %s\n", issue.Status)
			fmt.Println()

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Println("Error: confirmation required in non-interactive mode. Use --force to override.")
				os.Exit(1)
			}

			fmt.Print("Are you sure you want to delete this issue? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				os.Exit(0)
			}
		}

		// 3. Execute Delete
		if err := client.DeleteIssue(id, deleteReasonFlag); err != nil {
			fmt.Printf("Error deleting issue: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Deleted %s\n", id)
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForceFlag, "force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().StringVarP(&deleteReasonFlag, "reason", "r", "", "Reason for deletion")
	rootCmd.AddCommand(deleteCmd)
}
