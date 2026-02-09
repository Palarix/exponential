package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kuyio/beats/internal/beats"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:               "comment [issue ID] [text]",
	Short:             "Add a comment to an issue",
	Long:              `Add a comment to an existing issue. content can be provided as an argument or via stdin.`,
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]
		var text string

		// 1. Get Comment Text
		if len(args) == 2 {
			text = args[1]
		} else {
			// Read from Stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				bytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read from stdin: %w", err)
				}
				text = string(bytes)
			} else {
				return fmt.Errorf("comment text is required (provide as argument or pipe to stdin)")
			}
		}

		text = strings.TrimSpace(text)
		if text == "" {
			return fmt.Errorf("comment text cannot be empty")
		}

		// 2. Use Client to Add Comment
		client := beats.NewClient(cfg)

		// Verify issue exists first? AddComment appends event, projection happens later.
		// However, Client.AddComment doesn't verify existence currenty.
		// Should we verify? The original implementation did.
		// Let's verify existence using client.GetIssue first.
		if _, err := client.GetIssue(issueID); err != nil {
			return fmt.Errorf("issue %s not found", issueID)
		}

		if err := client.AddComment(issueID, text); err != nil {
			return fmt.Errorf("failed to add comment: %w", err)
		}

		fmt.Printf("Comment added to %s\n", issueID)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commentCmd)
}
