package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/exponential/internal/model"
)

type ReviewCommit struct {
	SHA     string
	Message string
}

type ReviewFile struct {
	Status     string
	Path       string
	Insertions int
	Deletions  int
}

type ReviewData struct {
	Issue   *model.Issue
	Base    string
	Commits []ReviewCommit
	Files   []ReviewFile
}

// RenderReview renders the review summary for an issue with a matching branch.
func RenderReview(data ReviewData, termWidth int) string {
	var sb strings.Builder
	issue := data.Issue
	bs := issue.BranchStats
	if bs == nil {
		sb.WriteString("No branch detected for this issue.\n")
		return sb.String()
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
	shaStyle := lipgloss.NewStyle().Foreground(MutedColor)
	addStyle := lipgloss.NewStyle().Foreground(DoingColor)
	delStyle := lipgloss.NewStyle().Foreground(BlockedColor)

	// Title
	sb.WriteString(headerStyle.Render(issue.Title))
	sb.WriteString("\n")

	// Meta line
	stStyle := StatusStyle(issue.Status)
	icon := StatusIconForIssue(issue)
	metaParts := []string{
		MutedStyle.Render(issue.ID),
		stStyle.Render(fmt.Sprintf("%s %s", icon, string(issue.Status))),
	}
	if len(issue.Labels) > 0 {
		metaParts = append(metaParts, FormatLabelsBadge(issue.Labels))
	}
	sb.WriteString(strings.Join(metaParts, "  "))
	sb.WriteString("\n")

	// Separator
	divider := MutedStyle.Render(strings.Repeat("═", min(termWidth, 80)))
	sb.WriteString(divider + "\n")

	// Branch → base
	sb.WriteString(fmt.Sprintf("%s → %s\n", headerStyle.Render(bs.Branch), MutedStyle.Render(data.Base)))

	// Stats line
	commitWord := "commits"
	if bs.Commits == 1 {
		commitWord = "commit"
	}
	fileWord := "files"
	if bs.FilesChanged == 1 {
		fileWord = "file"
	}
	statsLine := fmt.Sprintf("⊙ %s  %d %s, %d %s, %s %s",
		shaStyle.Render(bs.HeadSHA),
		bs.Commits, commitWord,
		bs.FilesChanged, fileWord,
		addStyle.Render(fmt.Sprintf("+%d", bs.Insertions)),
		delStyle.Render(fmt.Sprintf("-%d", bs.Deletions)))
	sb.WriteString(statsLine + "\n")

	// Commits
	if len(data.Commits) > 0 {
		sb.WriteString("\n")
		sb.WriteString(RenderHeader("Commits"))
		for _, c := range data.Commits {
			sb.WriteString(fmt.Sprintf("  %s  %s\n",
				shaStyle.Render(c.SHA),
				c.Message))
		}
	}

	// Description
	if issue.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(RenderHeader("Description"))
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

	// Files changed
	if len(data.Files) > 0 {
		sb.WriteString(RenderHeader(fmt.Sprintf("Files Changed (%d)", len(data.Files))))
		for _, f := range data.Files {
			statusChar := f.Status[:1]
			var statusStyle lipgloss.Style
			switch statusChar {
			case "A":
				statusStyle = lipgloss.NewStyle().Foreground(DoingColor)
			case "D":
				statusStyle = lipgloss.NewStyle().Foreground(BlockedColor)
			case "R":
				statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9800"))
			default:
				statusStyle = lipgloss.NewStyle().Foreground(MutedColor)
			}
			stats := fmt.Sprintf("%s %s",
				addStyle.Render(fmt.Sprintf("+%d", f.Insertions)),
				delStyle.Render(fmt.Sprintf("-%d", f.Deletions)))
			sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
				statusStyle.Render(statusChar),
				f.Path,
				stats))
		}
	}

	return sb.String()
}
