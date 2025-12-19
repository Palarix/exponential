package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:               "show [id]",
	Short:             "Show issue details and history",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

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

		// Status Config (Option A icons & colors)
		statusIcons := map[string]string{
			"BACKLOG": "•",
			"PLANNED": "○",
			"DOING":   "●",
			"BLOCKED": "x",
			"DONE":    "✔",
		}
		statusColorValues := map[string]lipgloss.Style{
			"BACKLOG": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#626262"}),
			"PLANNED": lipgloss.NewStyle().Foreground(lipgloss.Color("67")),
			"DOING":   lipgloss.NewStyle().Foreground(lipgloss.Color("172")),
			"BLOCKED": lipgloss.NewStyle().Foreground(lipgloss.Color("160")),
			"DONE":    lipgloss.NewStyle().Foreground(lipgloss.Color("64")),
		}

		// Styles
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 0)

		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		// Print Header
		fmt.Println()
		fmt.Println(titleStyle.Render(fmt.Sprintf("[%s] %s", issue.ID, issue.Title)))
		fmt.Println()

		stVal := string(issue.Status)
		stIcon := statusIcons[stVal]
		if stIcon != "" {
			stVal = fmt.Sprintf("%s %s", stIcon, stVal)
		}

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
			fmt.Println(issue.Description)
		} else {
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

			childColumns := []table.Column{
				{Title: "ID", Width: 16},
				{Title: "Status", Width: 16},
				{Title: "Title", Width: 60},
			}

			childRows := []table.Row{}
			for _, child := range childIssues {
				// Truncate title if too long
				title := child.Title
				if len(title) > 47 {
					title = title[:47] + "..."
				}
				childRows = append(childRows, table.Row{child.ID, string(child.Status), title})
			}

			childTable := table.New(
				table.WithColumns(childColumns),
				table.WithRows(childRows),
				table.WithFocused(false),
				table.WithHeight(len(childRows)+1),
			)

			// Helper style for child table
			s := table.DefaultStyles()
			s.Header = s.Header.
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				BorderBottom(true).
				Bold(true)
			s.Selected = lipgloss.NewStyle()
			childTable.SetStyles(s)

			fmt.Println(childTable.View())
			fmt.Println()
		}

		renderHistory(issue.Events)
	},
}

func renderHistory(events []model.Event) {
	fmt.Println(lipgloss.NewStyle().Bold(true).Underline(true).Render("History"))
	fmt.Println()

	columns := []table.Column{
		{Title: "Time", Width: 20},
		{Title: "User", Width: 20},
		{Title: "Action", Width: 10},
		{Title: "Details", Width: 40},
	}

	rows := []table.Row{}
	for _, evt := range events {
		timeStr := evt.CreatedAt.Local().Format(time.RFC822)

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
		rows = append(rows, table.Row{timeStr, evt.CreatedBy, string(evt.Type), details})
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
