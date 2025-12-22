package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kuyio/beats/internal/model"
)

// ListMagnitudes for humanize
var ListMagnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "now", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1 sec %s", DivBy: 1},
	{D: time.Minute, Format: "%d secs %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1 min %s", DivBy: 1},
	{D: time.Hour, Format: "%d mins %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1 hr %s", DivBy: 1},
	{D: time.Hour * 24, Format: "%d hrs %s", DivBy: time.Hour},
	{D: 2 * time.Hour * 24, Format: "1 day %s", DivBy: 1},
	{D: humanize.LongTime, Format: "%d days %s", DivBy: time.Hour * 24},
}

// RenderIssueList renders a table of issues.
func RenderIssueList(issues []*model.Issue, termWidth int) {
	// Empty line at start
	fmt.Println()

	// Apply safety margin
	if termWidth > 120 {
		termWidth = 120 // Cap max width for readability
	}
	effectiveWidth := termWidth - 4 // Margin

	// Column Config
	columns := []Column{
		{Title: "ID", Width: 14},
		{Title: " ", Width: 5},
		{Title: " ", Width: 6},
		{Title: "Title", Width: 20, Flex: true},
		{Title: "Updated", Width: 15},
		{Title: "Added By", Width: 20},
	}

	colWidths := CalculateColumnWidths(effectiveWidth, columns)

	// Header Style
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("240"))

	// Render Header
	var headerCells []string
	for i, col := range columns {
		cell := lipgloss.NewStyle().Width(colWidths[i]).Padding(0, 1).Render(col.Title)
		headerCells = append(headerCells, cell)
	}
	fmt.Println(headerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, headerCells...)))

	// Row Styles
	strikeStyle := StrikeStyle

	for _, i := range issues {
		updTime := humanize.CustomRelTime(i.UpdatedAt, time.Now(), "ago", "from now", ListMagnitudes)

		// Format 'By' column
		byStr := FormatByString(i.CreatedBy)

		// Base logic: Determine base style for the row
		var idS, stS, tiS, upS, byS lipgloss.Style

		switch i.Status {
		case model.StatusDone:
			idS, tiS, upS, byS = WhiteStyle, strikeStyle, WhiteStyle, WhiteStyle
		default:
			idS, tiS, upS, byS = WhiteStyle, WhiteStyle, WhiteStyle, WhiteStyle
		}

		stS = StatusStyle(i.Status)

		// Prepare Status String with Icon
		stIcon := StatusIcon(i.Status)
		stStr := fmt.Sprintf(" %s ", stIcon)

		// Determine Type Style
		tyS := TypeStyle(i.Kind)

		// Abbreviate Kind
		kindStr := FormatKindTag(i.Kind)

		// Check for ParentID and add visual prefix
		title := i.Title
		if i.ParentID != "" {
			title = " ↳ " + title
		}

		// Render
		c1 := RenderCell(i.ID, idS, colWidths[0])
		c2 := RenderCell(kindStr, tyS, colWidths[1])
		c3 := RenderCell(stStr, stS, colWidths[2])
		c4 := RenderCell(title, tiS, colWidths[3])
		c7 := RenderCell(updTime, upS, colWidths[4])
		c8 := RenderCell(byStr, byS, colWidths[5])

		row := lipgloss.JoinHorizontal(lipgloss.Left, c1, c2, c3, c4, c7, c8)
		fmt.Println(row)
	}
}
