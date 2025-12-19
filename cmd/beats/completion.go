package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	// logic to find recent DONE items
	var doneIssues []*model.Issue
	for _, i := range issues {
		if i.Status == model.StatusDone {
			doneIssues = append(doneIssues, i)
		}
	}
	// Sort by UpdatedAt Desc
	sort.Slice(doneIssues, func(i, j int) bool {
		return doneIssues[i].UpdatedAt.After(doneIssues[j].UpdatedAt)
	})
	recentDoneIDs := make(map[string]bool)
	for k := 0; k < 3 && k < len(doneIssues); k++ {
		recentDoneIDs[doneIssues[k].ID] = true
	}

	// Styles
	statusStyles := map[model.IssueStatus]lipgloss.Style{
		model.StatusBacklog: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#626262"}),
		model.StatusPlanned: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#626262"}),
		model.StatusDoing:   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1524ffff", Dark: "#337effff"}),
		model.StatusBlocked: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#b30000ff", Dark: "#ff0000ff"}),
		model.StatusDone:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#009938ff", Dark: "#00cb55ff"}),
	}
	// Type Styles
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	blueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	purpleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	whiteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	// Icons
	statusIcons := map[model.IssueStatus]string{
		model.StatusBacklog: "•",
		model.StatusPlanned: "●",
		model.StatusDoing:   "●",
		model.StatusBlocked: "x",
		model.StatusDone:    "●",
	}

	for id, issue := range issues {
		// Filter out DONE issues unless they are recent
		if issue.Status == model.StatusDone && !recentDoneIDs[id] {
			continue
		}

		if strings.HasPrefix(id, toComplete) {
			// Abbreviate Type: [T], [E], [B]
			typeTag := ""
			var tyStyle lipgloss.Style
			switch issue.Kind {
			case "BUG":
				tyStyle = redStyle
			case "EPIC":
				tyStyle = purpleStyle
			case "FEATURE":
				tyStyle = blueStyle
			default:
				tyStyle = whiteStyle
			}

			if len(issue.Kind) > 0 {
				typeTag = tyStyle.Render(fmt.Sprintf("[%c]", issue.Kind[0]))
			}

			// Get Icon
			icon := statusIcons[issue.Status]
			if icon == "" {
				icon = " "
			}
			stStyle := statusStyles[issue.Status]
			if stStyle.GetForeground() == nil {
				stStyle = whiteStyle
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
