package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/kuyio/beats/internal/ui"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

func completeIssueIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	events, err := storage.ReadEvents()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	issues := beats.ProjectIssues(events)
	var completions []string

	// Get recent DONE items using shared utility
	recentDoneIDs := ui.GetRecentDoneIDsFromMap(issues, 3)

	// Detect shell to determine formatting approach
	shell := os.Getenv("SHELL")
	useColors := true
	if strings.Contains(shell, "bash") {
		useColors = false
	} else if strings.Contains(shell, "zsh") {
		lipgloss.SetColorProfile(termenv.ANSI256)
	}

	for id, issue := range issues {
		// Filter out DONE issues unless they are recent
		if issue.Status == model.StatusDone && !recentDoneIDs[id] {
			continue
		}

		if strings.HasPrefix(id, toComplete) {
			var labelTag, icon string

			// Format labels
			labelStr := ""
			if len(issue.Labels) > 0 {
				labelStr = "[" + strings.Join(issue.Labels, ",") + "]"
			}

			if useColors {
				// Use colored formatting for zsh and other supporting shells
				if labelStr != "" {
					labelTag = ui.AccentStyle.Render(labelStr)
				}

				// Get Icon using shared styles
				icon = ui.StatusIcon(issue.Status)
				if icon == "" {
					icon = " "
				}
				stStyle := ui.StatusStyle(issue.Status)
				if stStyle.GetForeground() == nil {
					stStyle = ui.WhiteStyle
				}
				icon = stStyle.Render(icon)

				// Render Title (Strikethrough if Done)
				title := issue.Title
				if issue.Status == model.StatusDone {
					title = fmt.Sprintf("\x1b[9m%s\x1b[29m", title)
				}

				desc := fmt.Sprintf("%s %s %s\x1b[0m", labelTag, icon, title)
				completions = append(completions, fmt.Sprintf("%s\t%s", id, desc))
			} else {
				// Use plain text for bash
				icon = ui.StatusIcon(issue.Status)
				if icon == "" {
					icon = " "
				}

				title := issue.Title
				if issue.Status == model.StatusDone {
					title = title + " (DONE)"
				}

				desc := fmt.Sprintf("%s %s %s", labelStr, icon, title)
				completions = append(completions, fmt.Sprintf("%s\t%s", id, desc))
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
