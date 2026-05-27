package main

import (
	"fmt"
	"strings"

	"github.com/kuyio/beats/internal/beats"
	"github.com/spf13/cobra"
)

var commentJSONFlag bool

var commentCmd = &cobra.Command{
	Use:               "comment [issue ID] [text]",
	Short:             "Add a comment to an issue",
	Long:              `Add a comment to an existing issue. Text can be passed as a positional argument, piped via stdin, or read from stdin explicitly with '-' as the second argument. Pass --json to read a structured payload {"body": "..."} from stdin.`,
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]
		var text string

		// 1. Get Comment Text
		switch {
		case commentJSONFlag:
			content, err := readStdinExplicit()
			if err != nil {
				return err
			}
			var input commentJSONInput
			if err := decodeStrict(content, &input); err != nil {
				return err
			}
			text = input.Body
		case len(args) == 2 && args[1] == "-":
			content, err := readStdinExplicit()
			if err != nil {
				return err
			}
			text = content
		case len(args) == 2:
			text = strings.TrimSpace(args[1])
		case isStdinPiped():
			content, err := readAllStdin()
			if err != nil {
				return err
			}
			text = content
		default:
			return fmt.Errorf("comment text is required (provide as argument, pipe to stdin, or use '-' to read stdin explicitly)")
		}

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
	commentCmd.Flags().BoolVar(&commentJSONFlag, "json", false, "Read a structured comment payload as JSON from stdin")
	rootCmd.AddCommand(commentCmd)
}
