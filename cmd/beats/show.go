package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/beats/cmd/beats/ui"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var showCmd = &cobra.Command{
	Use:               "show [id]",
	Short:             "Show issue details and history",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		showIssue(args[0])
	},
}

func showIssue(id string) {
	events, err := storage.ReadEvents()
	if err != nil {
		fmt.Printf("Error reading events: %v\n", err)
		os.Exit(1)
	}

	issues := model.ProjectIssues(events)
	issue, exists := issues[id]
	if !exists {
		fmt.Printf("Issue %s not found\n", id)
		os.Exit(1)
	}

	// Use shared status icons and colors from ui package
	statusIcons := ui.StatusIconsStr
	statusColorValues := ui.StatusStylesStr

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 0)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Detect Terminal Width
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termWidth <= 0 {
		termWidth = 100 // Fallback
	}

	// Apply safety margin and cap for readability
	termWidth -= 4
	if termWidth > 116 {
		termWidth = 116 // Total effective width including padding
	}

	// Print Header
	fmt.Println()
	fmt.Println(titleStyle.Render(fmt.Sprintf("[%s] %s", issue.ID, issue.Title)))
	fmt.Println()

	// Task 1: Show just status text, without icon
	stVal := string(issue.Status)
	stStyle := statusColorValues[string(issue.Status)]
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

	// Check for child issues
	childIssues := []*model.Issue{}
	for _, otherIssue := range issues {
		if otherIssue.ParentID == issue.ID {
			childIssues = append(childIssues, otherIssue)
		}
	}

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
		cols := []struct {
			Title string
			Width int
		}{
			{"ID", 16},
			{"Type", 6},
			{"Status", 16},
			{"Title", childTitleWidth},
		}

		// Header Style
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("240"))

		// Render Header
		var headerCells []string
		for _, col := range cols {
			cell := lipgloss.NewStyle().Width(col.Width).Padding(0, 1).Render(col.Title)
			headerCells = append(headerCells, cell)
		}
		fmt.Println(headerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, headerCells...)))

		for _, child := range childIssues {
			// Truncate title if too long
			title := child.Title
			if len(title) > childTitleWidth-2 { // -2 for padding
				title = title[:childTitleWidth-2-1] + "…"
			}

			// Type
			kindTag := ui.FormatKindTag(child.Kind)
			// Status
			stIcon := statusIcons[string(child.Status)]
			stText := fmt.Sprintf("%s %s", stIcon, child.Status)

			// Styles
			tyS := ui.TypeStyle(child.Kind)
			stS := statusColorValues[string(child.Status)]
			idS := ui.WhiteStyle
			tiS := ui.WhiteStyle
			if child.Status == model.StatusDone {
				tiS = ui.StrikeStyle
			}

			renderCell := ui.RenderCell

			c1 := renderCell(child.ID, idS, cols[0].Width)
			c2 := renderCell(kindTag, tyS, cols[1].Width)
			c3 := renderCell(stText, stS, cols[2].Width)
			c4 := renderCell(title, tiS, cols[3].Width)

			row := lipgloss.JoinHorizontal(lipgloss.Left, c1, c2, c3, c4)
			fmt.Println(row)
		}
		fmt.Println()
	}

	renderHistory(issue.Events, termWidth)
}

func renderHistory(events []model.Event, termWidth int) {
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

func init() {
	rootCmd.AddCommand(showCmd)
}

func uintPtr(u uint) *uint {
	return &u
}
