package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/storage"
	"github.com/spf13/cobra"
)

var commentsCmd = &cobra.Command{
	Use:               "comments [issue ID]",
	Short:             "List comments for an issue",
	Long:              `Display the conversation history for a specific issue.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		events, err := storage.ReadEvents()
		if err != nil {
			return fmt.Errorf("failed to read events: %w", err)
		}

		issues := beats.ProjectIssues(events)
		issue, ok := issues[issueID]
		if !ok {
			return fmt.Errorf("issue %s not found", issueID)
		}

		if len(issue.Comments) == 0 {
			fmt.Printf("No comments for issue %s\n", issueID)
			return nil
		}

		// Calculate max width for indentation/alignment
		// We want to format like:
		// Author Name (Time Ago)
		// Comment Text...
		// ----------------------

		// Detect if we should use colors (sanity check on top of lipgloss auto-detect)
		// This follows the pattern in other commands
		useColors := true
		if os.Getenv("NO_COLOR") != "" {
			useColors = false
		}

		// Styles
		authorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")) // Blue
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))            // Gray
		borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))          // Dark Gray

		for i, comment := range issue.Comments {
			// Header: Author (Time)
			author := comment.CreatedBy
			// Attempt to extract name from "First Last <email>"
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
	rootCmd.AddCommand(commentsCmd)
}
