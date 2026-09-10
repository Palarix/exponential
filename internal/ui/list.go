package ui

import (
	"fmt"
	"strings"

	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/palarix/exponential/internal/model"
)

var listMagnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "now", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1s %s", DivBy: 1},
	{D: time.Minute, Format: "%ds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1m %s", DivBy: 1},
	{D: time.Hour, Format: "%dm %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1h %s", DivBy: 1},
	{D: 24 * time.Hour, Format: "%dh %s", DivBy: time.Hour},
	{D: 2 * 24 * time.Hour, Format: "1d %s", DivBy: 1},
	{D: 7 * 24 * time.Hour, Format: "%dd %s", DivBy: 24 * time.Hour},
	{D: 2 * 7 * 24 * time.Hour, Format: "1w %s", DivBy: 1},
	{D: 30 * 24 * time.Hour, Format: "%dw %s", DivBy: 7 * 24 * time.Hour},
	{D: 2 * 30 * 24 * time.Hour, Format: "1mo %s", DivBy: 1},
	{D: 365 * 24 * time.Hour, Format: "%dmo %s", DivBy: 30 * 24 * time.Hour},
	{D: 2 * 365 * 24 * time.Hour, Format: "1y %s", DivBy: 1},
	{D: 100 * 365 * 24 * time.Hour, Format: "%dy %s", DivBy: 365 * 24 * time.Hour},
}

// RenderIssueList renders a formatted issue list for the terminal.
// The prefix parameter is the configured issue ID prefix (e.g. "xpo");
// it determines the width of the ID column.
func RenderIssueList(issues []*model.Issue, termWidth int, prefix string) string {
	if len(issues) == 0 {
		return ""
	}

	var sb strings.Builder

	// Column Widths: prefix + "-" + 6 hex chars + padding
	idWidth := len(prefix) + 1 + 8
	statusWidth := 10
	labelWidth := 12
	tagsWidth := 5
	assigneeWidth := 12
	estWidth := 5
	timeWidth := 10
	padding := 7 // spaces between columns

	// Calculate title width from remaining space
	titleWidth := termWidth - idWidth - statusWidth - labelWidth - tagsWidth - assigneeWidth - estWidth - timeWidth - padding
	if titleWidth < 20 {
		titleWidth = 20
	}

	// Header
	header := fmt.Sprintf("%s %s %s %s %s %s %s %s",
		RenderCell("ID", idWidth, MutedStyle),
		RenderCell("STATUS", statusWidth, MutedStyle),
		RenderCell("LABEL", labelWidth, MutedStyle),
		RenderCell("TITLE", titleWidth, MutedStyle),
		RenderCell("TAGS", tagsWidth, MutedStyle),
		RenderCell("EST", estWidth, MutedStyle),
		RenderCell("ASSIGNEE", assigneeWidth, MutedStyle),
		RenderCell("UPDATED", timeWidth, MutedStyle),
	)
	sb.WriteString(header + "\n")
	sb.WriteString(MutedStyle.Render(strings.Repeat("─", termWidth)) + "\n")

	for _, i := range issues {
		stStyle := StatusStyle(i.Status)
		icon := StatusIconForIssue(i)

		// ID
		idStr := FormatID(i.ID, idWidth)

		// Status
		statusStr := fmt.Sprintf("%s %s", icon, string(i.Status))

		// Primary label (first label, colored)
		labelStr := ""
		if len(i.Labels) > 0 {
			color := LabelColor(i.Labels[0])
			style := lipgloss.NewStyle().Foreground(color)
			labelStr = style.Render(Truncate(i.Labels[0], labelWidth))
		}

		// Tags: +N for additional labels beyond the primary
		tagsStr := ""
		if len(i.Labels) > 1 {
			tagsStr = fmt.Sprintf("+%d", len(i.Labels)-1)
		}

		// Title (with optional branch stats suffix)
		title := i.Title
		if i.ParentID != "" {
			title = "  └─ " + title
		}
		branchTag := FormatBranchStats(i.BranchStats)
		if branchTag != "" {
			available := titleWidth - lipgloss.Width(title) - 1
			if available >= lipgloss.Width(branchTag) {
				title = title + " " + branchTag
			}
		}
		title = Truncate(title, titleWidth)

		// Estimate
		estStr := ""
		if i.Estimate > 0 {
			estStr = fmt.Sprintf("%d", i.Estimate)
		}

		// Assignee
		assigneeStr := ""
		if i.Assignee != "" {
			assigneeStr = Truncate(ExtractName(i.Assignee), assigneeWidth)
		}

		// Time
		timeStr := humanize.CustomRelTime(i.UpdatedAt, time.Now(), "ago", "from now", listMagnitudes)
		timeStr = Truncate(timeStr, timeWidth)

		// Apply status style to the row
		titleStyle := lipgloss.NewStyle()
		if model.IsTerminal(i.Status) {
			titleStyle = StatusStyle(i.Status)
		}

		row := fmt.Sprintf("%s %s %s %s %s %s %s %s",
			RenderCell(idStr, idWidth, MutedStyle),
			RenderCell(statusStr, statusWidth, stStyle),
			RenderCell(labelStr, labelWidth, lipgloss.NewStyle()),
			RenderCell(title, titleWidth, titleStyle),
			RenderCell(tagsStr, tagsWidth, MutedStyle),
			RenderCell(estStr, estWidth, lipgloss.NewStyle()),
			RenderCell(assigneeStr, assigneeWidth, MutedStyle),
			RenderCell(timeStr, timeWidth, MutedStyle),
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}
