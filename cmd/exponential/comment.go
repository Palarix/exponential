package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
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

		switch {
		case commentJSONFlag:
			content, err := readStdinExplicit()
			if err != nil {
				exitJSONError(err)
			}
			var input jsonio.CommentInput
			if err := jsonio.DecodeStrict(content, &input); err != nil {
				exitJSONError(err)
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
			if commentJSONFlag {
				exitJSONError(fmt.Errorf("comment text cannot be empty"))
			}
			return fmt.Errorf("comment text cannot be empty")
		}

		client := exponential.NewClient(cfg)

		if _, err := client.GetIssue(issueID); err != nil {
			if commentJSONFlag {
				exitJSONError(fmt.Errorf("issue %s not found", issueID))
			}
			return fmt.Errorf("issue %s not found", issueID)
		}

		if err := client.AddComment(issueID, text); err != nil {
			if commentJSONFlag {
				exitJSONError(fmt.Errorf("failed to add comment: %w", err))
			}
			return fmt.Errorf("failed to add comment: %w", err)
		}

		if commentJSONFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(jsonio.CommentOutput{ID: issueID})
			return nil
		}

		fmt.Printf("Comment added to %s\n", issueID)
		return nil
	},
}

func init() {
	commentCmd.Flags().BoolVar(&commentJSONFlag, "json", false, "Read a structured comment payload as JSON from stdin")
	rootCmd.AddCommand(commentCmd)
}
