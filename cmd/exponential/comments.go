package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/spf13/cobra"
)

var commentsJSONFlag bool

var commentsCmd = &cobra.Command{
	Use:               "comments [issue ID]",
	Short:             "List comments for an issue",
	Long:              `Display the conversation history for a specific issue.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := exponential.NewClient(cfg)
		issue, err := client.GetIssue(args[0])
		if err != nil {
			return fmt.Errorf("issue %s not found: %w", args[0], err)
		}

		if commentsJSONFlag {
			out := jsonio.CommentsOutput{
				Comments: jsonio.ToCommentSummaries(issue.Comments),
			}
			if out.Comments == nil {
				out.Comments = []jsonio.CommentSummary{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)
			return nil
		}

		if len(issue.Comments) == 0 {
			fmt.Printf("No comments for issue %s\n", issue.ID)
			return nil
		}

		useColors := os.Getenv("NO_COLOR") == ""

		authorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		for i, comment := range issue.Comments {
			author := comment.CreatedBy
			if idx := strings.Index(author, "<"); idx > 0 {
				author = strings.TrimSpace(author[:idx])
			}

			timeAgo := time.Since(comment.CreatedAt).Round(time.Minute)
			timeStr := fmt.Sprintf("%s ago", timeAgo)

			header := fmt.Sprintf("%s (%s)", author, timeStr)
			if useColors {
				header = fmt.Sprintf("%s %s", authorStyle.Render(author), timeStyle.Render(fmt.Sprintf("(%s)", timeStr)))
			}

			fmt.Println(header)
			fmt.Println(comment.Text)

			if i < len(issue.Comments)-1 {
				sep := strings.Repeat("-", 40)
				if useColors {
					sep = borderStyle.Render(sep)
				}
				fmt.Println(sep)
			}
		}

		return nil
	},
}

func init() {
	commentsCmd.Flags().BoolVar(&commentsJSONFlag, "json", false, "Output as JSON matching MCP comment schema")
	rootCmd.AddCommand(commentsCmd)
}
