package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/beats/cmd/beats/ui"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

func completeIssueIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Force colors for completion output (since it's piped)
	lipgloss.SetColorProfile(termenv.ANSI256)

	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	events, err := storage.ReadEvents()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	issues := model.ProjectIssues(events)
	var completions []string

	// Get recent DONE items using shared utility
	recentDoneIDs := ui.GetRecentDoneIDsFromMap(issues, 3)

	for id, issue := range issues {
		// Filter out DONE issues unless they are recent
		if issue.Status == model.StatusDone && !recentDoneIDs[id] {
			continue
		}

		if strings.HasPrefix(id, toComplete) {
			// Abbreviate Type using shared utility
			tyStyle := ui.TypeStyle(issue.Kind)
			typeTag := tyStyle.Render(ui.FormatKindTag(issue.Kind))

			// Get Icon using shared styles
			icon := ui.StatusIcons[issue.Status]
			if icon == "" {
				icon = " "
			}
			stStyle := ui.StatusStyles[issue.Status]
			if stStyle.GetForeground() == nil {
				stStyle = ui.WhiteStyle
			}
			icon = stStyle.Render(icon)

			// Render Title (Strikethrough if Done)
			title := issue.Title
			if issue.Status == model.StatusDone {
				// Manual ANSI codes to ensure correct resetting in Zsh
				// \x1b[9m = Strikethrough, \x1b[29m = Strikethrough Off
				title = fmt.Sprintf("\x1b[9m%s\x1b[29m", title)
			}

			// Format: "id\t[Type] Icon Title"
			// Append explicit reset \x1b[0m to prevent style leaking
			desc := fmt.Sprintf("%s %s %s\x1b[0m", typeTag, icon, title)
			completions = append(completions, fmt.Sprintf("%s\t%s", id, desc))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
