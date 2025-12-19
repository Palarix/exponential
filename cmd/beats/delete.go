package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	deleteForceFlag  bool
	deleteReasonFlag string
)

var deleteCmd = &cobra.Command{
	Use:               "delete [id]",
	Short:             "Delete an issue",
	Long:              `Delete an issue from the board. The issue will no longer appear in lists or reports.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		user := getUser()

		// Load events and project issues to verify issue exists
		events, err := storage.ReadEvents()
		if err != nil {
			fmt.Printf("Error reading events: %v\n", err)
			os.Exit(1)
		}

		issuesMap := model.ProjectIssues(events)
		issue, exists := issuesMap[id]
		if !exists {
			fmt.Printf("Issue %s not found\n", id)
			os.Exit(1)
		}

		// Confirmation prompt (unless --force)
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

		// Create delete event
		payload := model.DeletePayload{
			Reason: deleteReasonFlag,
		}

		event := model.Event{
			ID:        id,
			Type:      model.EventTypeDelete,
			Payload:   payload,
			CreatedAt: time.Now().UTC(),
			CreatedBy: user,
		}

		if err := storage.AppendEvent(event); err != nil {
			fmt.Printf("Error appending event: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Deleted %s\n", id)

		if cfg.AutoCommit {
			commitMsg := fmt.Sprintf("beats: delete %s", id)
			fmt.Println("Auto-committing...")
			if err := exec.Command("git", "add", ".beats/issues.db").Run(); err != nil {
				fmt.Printf("Error adding to git: %v\n", err)
			} else if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
				fmt.Printf("Error committing: %v\n", err)
			}
		}
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForceFlag, "force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().StringVarP(&deleteReasonFlag, "reason", "r", "", "Reason for deletion")
	rootCmd.AddCommand(deleteCmd)
}
