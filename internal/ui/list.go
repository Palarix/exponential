package ui

import (
	"fmt"
	"strings"

	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kuyio/beats/internal/model"
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
func RenderIssueList(issues []*model.Issue, termWidth int) string {
	if len(issues) == 0 {
		return ""
	}

	var sb strings.Builder

	// Column Widths
	idWidth := 14
	statusWidth := 10
	labelWidth := 12
	assigneeWidth := 12
	estWidth := 5
	timeWidth := 10
	padding := 6 // spaces between columns

	// Calculate title width from remaining space
	titleWidth := termWidth - idWidth - statusWidth - labelWidth - assigneeWidth - estWidth - timeWidth - padding
	if titleWidth < 20 {
		titleWidth = 20
	}

	// Header
	header := fmt.Sprintf("%s %s %s %s %s %s %s",
		RenderCell("ID", idWidth, MutedStyle),
		RenderCell("STATUS", statusWidth, MutedStyle),
		RenderCell("LABELS", labelWidth, MutedStyle),
		RenderCell("TITLE", titleWidth, MutedStyle),
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

		// Labels
		labelStr := ""
		if len(i.Labels) > 0 {
			var parts []string
			for _, l := range i.Labels {
				color := LabelColor(l)
				style := lipgloss.NewStyle().Foreground(color)
				parts = append(parts, style.Render(l))
			}
			joined := strings.Join(parts, ",")
			// Truncate based on visible width
			if lipgloss.Width(joined) > labelWidth {
				// Simple truncation for now, might cut off ANSI codes but lipgloss should handle it
				// Better approach: Truncate the source strings or just show as many full tags as fit
				// For now, let's just join and truncate, hoping lipgloss.Width logic in Truncate mimics visual width
				labelStr = Truncate(joined, labelWidth)
			} else {
				labelStr = joined
			}
		}

		// Title
		title := i.Title
		if i.ParentID != "" {
			title = "  └─ " + title // indent children
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
		if i.Status == model.StatusDone {
			titleStyle = titleStyle.Foreground(DoneColor).Strikethrough(true)
		}

		row := fmt.Sprintf("%s %s %s %s %s %s %s",
			RenderCell(idStr, idWidth, MutedStyle),
			RenderCell(statusStr, statusWidth, stStyle),
			RenderCell(labelStr, labelWidth, lipgloss.NewStyle()),
			RenderCell(title, titleWidth, titleStyle),
			RenderCell(estStr, estWidth, lipgloss.NewStyle()),
			RenderCell(assigneeStr, assigneeWidth, MutedStyle),
			RenderCell(timeStr, timeWidth, MutedStyle),
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}
