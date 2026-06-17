package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/palarix/exponential/internal/model"
)

// RenderIssueDetails renders a detailed view of an issue.
func RenderIssueDetails(issue *model.Issue, children []*model.Issue, isArchived bool, termWidth int) string {
	var sb strings.Builder

	// --- Header ---
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))

	sb.WriteString(headerStyle.Render(issue.Title))
	sb.WriteString("\n")

	// Meta line: ID | Status | Labels
	stStyle := StatusStyle(issue.Status)
	icon := StatusIcon(issue.Status)
	metaParts := []string{
		MutedStyle.Render(issue.ID),
		stStyle.Render(fmt.Sprintf("%s %s", icon, string(issue.Status))),
	}

	if len(issue.Labels) > 0 {
		metaParts = append(metaParts, FormatLabelsBadge(issue.Labels))
	}

	if isArchived {
		metaParts = append(metaParts, WarningStyle.Render("[ARCHIVED]"))
	}

	sb.WriteString(strings.Join(metaParts, "  "))
	sb.WriteString("\n")

	// --- Details Block ---
	sb.WriteString(MutedStyle.Render(strings.Repeat("─", min(termWidth, 80))) + "\n")

	// Parent
	if issue.ParentID != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			MutedStyle.Render("Parent:"),
			AccentStyle.Render(issue.ParentID)))
	}

	// Assignee
	if issue.Assignee != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			MutedStyle.Render("Assignee:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render(issue.Assignee)))
	}

	// Estimate
	if issue.Estimate > 0 {
		sb.WriteString(fmt.Sprintf("  %s  %d pts\n",
			MutedStyle.Render("Estimate:"),
			issue.Estimate))
	}

	// Branch stats
	if issue.BranchStats != nil && issue.BranchStats.Commits > 0 {
		bs := issue.BranchStats
		branchLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render(bs.Branch)
		if bs.HeadSHA != "" {
			branchLine += "  " + MutedStyle.Render(bs.HeadSHA)
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			MutedStyle.Render("Branch:"),
			branchLine))
		commitWord := "commits"
		if bs.Commits == 1 {
			commitWord = "commit"
		}
		fileWord := "files changed"
		if bs.FilesChanged == 1 {
			fileWord = "file changed"
		}
		sb.WriteString(fmt.Sprintf("  %s  %d %s, %d %s, %s%s\n",
			MutedStyle.Render("Changes:"),
			bs.Commits, commitWord,
			bs.FilesChanged, fileWord,
			lipgloss.NewStyle().Foreground(DoingColor).Render(fmt.Sprintf("+%d", bs.Insertions)),
			lipgloss.NewStyle().Foreground(BlockedColor).Render(fmt.Sprintf(" -%d", bs.Deletions))))
	}

	// Created / Updated
	sb.WriteString(fmt.Sprintf("  %s  %s by %s\n",
		MutedStyle.Render("Created:"),
		humanize.Time(issue.CreatedAt),
		issue.CreatedBy))
	sb.WriteString(fmt.Sprintf("  %s  %s\n",
		MutedStyle.Render("Updated:"),
		humanize.Time(issue.UpdatedAt)))

	// Dependencies
	if len(issue.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s\n", MutedStyle.Render("Dependencies:")))
		for _, dep := range issue.Dependencies {
			kindStr := string(dep.Kind)
			sb.WriteString(fmt.Sprintf("    %s %s\n",
				MutedStyle.Render(kindStr+":"),
				AccentStyle.Render(dep.TargetID)))
		}
	}

	// --- Description ---
	if issue.Description != "" {
		sb.WriteString("\n")
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(min(termWidth-4, 76)),
		)
		rendered, err := renderer.Render(issue.Description)
		if err == nil {
			sb.WriteString(rendered)
		} else {
			sb.WriteString("  " + issue.Description + "\n")
		}
	}

	// --- Sub-Issues ---
	if len(children) > 0 {
		sb.WriteString(RenderHeader("Sub-Issues"))
		for _, child := range children {
			childSt := StatusStyle(child.Status)
			childIcon := StatusIcon(child.Status)
			childTitle := child.Title

			labelStr := ""
			if len(child.Labels) > 0 {
				labelStr = " " + FormatLabels(child.Labels)
			}

			sb.WriteString(fmt.Sprintf("  %s %s  %s%s\n",
				childSt.Render(childIcon+" "+string(child.Status)),
				MutedStyle.Render(child.ID),
				childTitle,
				labelStr))
		}
	}

	// --- Event History ---
	sb.WriteString(RenderHeader("History"))
	for _, evt := range issue.Events {
		timeStr := humanize.Time(evt.CreatedAt)
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			MutedStyle.Render(timeStr),
			MutedStyle.Render(string(evt.Type)),
			MutedStyle.Render(evt.CreatedBy)))
	}

	// --- Comments ---
	if len(issue.Comments) > 0 {
		sb.WriteString(RenderHeader("Comments"))
		for _, c := range issue.Comments {
			sb.WriteString(fmt.Sprintf("  %s  %s\n",
				AccentStyle.Render(c.CreatedBy),
				MutedStyle.Render(humanize.Time(c.CreatedAt))))
			sb.WriteString(fmt.Sprintf("  %s\n\n", c.Text))
		}
	}

	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
