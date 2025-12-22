package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/kuyio/beats/internal/model"
)

// RenderIssueDetails renders the full details of an issue, including children and history.
func RenderIssueDetails(issue *model.Issue, childIssues []*model.Issue, archived bool, termWidth int) {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(1, 1)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Apply safety margin and cap for readability
	// (Assumed termWidth passed in is raw terminal width)
	termWidth -= 4
	if termWidth > 116 {
		termWidth = 116 // Total effective width including padding
	}

	// Print Header
	fmt.Println()
	fmt.Println(titleStyle.Render(fmt.Sprintf("[%s] %s", issue.ID, issue.Title)))
	fmt.Println()

	if archived {
		fmt.Println("Note: This issue is archived.")
		fmt.Println()
	}

	// Task 1: Show just status text, without icon
	stVal := string(issue.Status)
	stStyle := StatusStyle(issue.Status)
	if stStyle.GetForeground() == nil {
		stStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // Fallback
	}

	fmt.Printf("%s %s\n", labelStyle.Render("Status:"), stStyle.Render(stVal))
	fmt.Printf("%s   %s\n", labelStyle.Render("Kind:"), issue.Kind)

	if issue.ParentID != "" {
		fmt.Printf("%s %s\n", labelStyle.Render("Parent:"), issue.ParentID)
	}
	if issue.Estimate > 0 {
		fmt.Printf("%s    %d\n", labelStyle.Render("Est:"), issue.Estimate)
	}
	if issue.Burned > 0 {
		fmt.Printf("%s %d\n", labelStyle.Render("Burned:"), issue.Burned)
	}
	if issue.BlockedBy != "" {
		blockStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
		fmt.Printf("%s %s (%s)\n", blockStyle.Render("Blocked By:"), issue.BlockedBy, issue.BlockReason)
	}

	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Underline(true).Render("Description"))
	fmt.Println()

	if issue.Description != "" {
		// Render markdown using glamour
		// Custom style to remove margins/indentation
		styleConfig := styles.DarkStyleConfig
		styleConfig.Document.Margin = uintPtr(0)

		r, _ := glamour.NewTermRenderer(
			glamour.WithStyles(styleConfig),
			glamour.WithWordWrap(termWidth),
		)
		out, err := r.Render(issue.Description)
		if err != nil {
			// Fallback to simple wrap if glamour fails
			fmt.Println() // Add the newline back if we fallback
			wrapped := lipgloss.NewStyle().Width(termWidth).Render(issue.Description)
			fmt.Println(wrapped)
		} else {
			fmt.Print(strings.TrimSpace(out))
			fmt.Println()
		}
	} else {
		fmt.Println()
		fmt.Println("No description provided.")
	}

	fmt.Println()

	if len(childIssues) > 0 {
		header := "Child Issues"
		if issue.Kind == "EPIC" {
			header = "Tasks"
		}
		fmt.Println(lipgloss.NewStyle().Bold(true).Underline(true).Render(header))
		fmt.Println()

		childTitleWidth := termWidth - 16 - 6 - 16 - 6 // ID(16) + Type(6) + Status(16) + borders/padding approx
		if childTitleWidth < 20 {
			childTitleWidth = 20
		}

		// Column Config
		columns := []Column{
			{Title: "ID", Width: 16},
			{Title: "Type", Width: 6},
			{Title: "Status", Width: 16},
			{Title: "Title", Width: childTitleWidth, Flex: true},
		}

		colWidths := CalculateColumnWidths(termWidth, columns)

		// Header Style
		headerStyle := HeaderStyle

		// Render Header
		var headerCells []string
		for i, col := range columns {
			cell := lipgloss.NewStyle().Width(colWidths[i]).Padding(0, 1).Render(col.Title)
			headerCells = append(headerCells, cell)
		}
		fmt.Println(headerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, headerCells...)))

		for _, child := range childIssues {
			// Truncate title if too long
			title := child.Title
			// We handle truncation in RenderCell now

			// Type
			kindTag := FormatKindTag(child.Kind)
			// Status
			stIcon := StatusIcon(child.Status)
			stText := fmt.Sprintf("%s %s", stIcon, child.Status)

			// Styles
			tyS := TypeStyle(child.Kind)
			stS := StatusStyle(child.Status)
			idS := WhiteStyle
			tiS := WhiteStyle
			if child.Status == model.StatusDone {
				tiS = StrikeStyle
			}

			c1 := RenderCell(child.ID, idS, colWidths[0])
			c2 := RenderCell(kindTag, tyS, colWidths[1])
			c3 := RenderCell(stText, stS, colWidths[2])
			c4 := RenderCell(title, tiS, colWidths[3])

			row := lipgloss.JoinHorizontal(lipgloss.Left, c1, c2, c3, c4)
			fmt.Println(row)
		}
		fmt.Println()
	}

	RenderHistory(issue.Events, termWidth)
}

func RenderHistory(events []model.Event, termWidth int) {
	fmt.Println(lipgloss.NewStyle().Bold(true).Underline(true).Render("History"))
	fmt.Println()

	detailsWidth := termWidth - 20 - 28 - 10 - 6 // Time(20) + User(20) + Action(10) + borders
	if detailsWidth < 20 {
		detailsWidth = 20
	}

	columns := []table.Column{
		{Title: "Time", Width: 20},
		{Title: "User", Width: 20},
		{Title: "Action", Width: 10},
		{Title: "Details", Width: detailsWidth},
	}

	rows := []table.Row{}
	for _, evt := range events {
		timeStr := evt.CreatedAt.Local().Format(time.RFC822)

		// Extract email from "Name <email>" format if present
		user := evt.CreatedBy
		if start := strings.LastIndex(user, "<"); start != -1 {
			if end := strings.LastIndex(user, ">"); end != -1 && end > start {
				if email := strings.TrimSpace(user[start+1 : end]); email != "" {
					user = email
				}
			}
		}

		details := ""
		switch evt.Type {
		case model.EventTypeCreate:
			details = "Created"
		case model.EventTypeUpdate:
			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.UpdatePayload
			json.Unmarshal(payloadBytes, &p)

			changes := []string{}
			if p.Status != nil {
				changes = append(changes, "Status->"+*p.Status)
			}
			if p.Title != nil {
				changes = append(changes, "Title Changed")
			}
			if p.ParentID != nil {
				changes = append(changes, "Parent->"+*p.ParentID)
			}
			details = fmt.Sprintf("%v", changes)
		case model.EventTypeWorkLog:
			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.WorkLogPayload
			json.Unmarshal(payloadBytes, &p)
			details = fmt.Sprintf("Logged %d SP", p.Amount)
		}
		rows = append(rows, table.Row{timeStr, user, string(evt.Type), details})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1),
	)

	// Re-use simple styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	t.SetStyles(s)

	fmt.Println(t.View())
}

func uintPtr(u uint) *uint {
	return &u
}
