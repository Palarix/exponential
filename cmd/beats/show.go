package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show issue details and history",
	Args:  cobra.ExactArgs(1),
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

		// Styles
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		// Print Header
		fmt.Println(titleStyle.Render(fmt.Sprintf("[%s] %s", issue.ID, issue.Title)))
		fmt.Println()

		fmt.Printf("%s %s\n", labelStyle.Render("Status:"), statusStyle.Render(string(issue.Status)))
		fmt.Printf("%s   %s\n", labelStyle.Render("Kind:"), issue.Kind)

		if issue.ParentID != "" {
			fmt.Printf("%s %s\n", labelStyle.Render("Parent:"), issue.ParentID)
		}
		if issue.Estimate > 0 {
			fmt.Printf("%s    %d\n", labelStyle.Render("Est:"), issue.Estimate)
		}
		if issue.BlockedBy != "" {
			blockStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
			fmt.Printf("%s %s (%s)\n", blockStyle.Render("Blocked By:"), issue.BlockedBy, issue.BlockReason)
		}

		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Bold(true).Underline(true).Render("Description"))
		fmt.Println(issue.Description)
		fmt.Println()

		fmt.Println(lipgloss.NewStyle().Bold(true).Underline(true).Render("History"))

		columns := []table.Column{
			{Title: "Time", Width: 20},
			{Title: "User", Width: 20},
			{Title: "Action", Width: 10},
			{Title: "Details", Width: 40},
		}

		rows := []table.Row{}
		for _, evt := range issue.Events {
			timeStr := evt.CreatedAt.Format(time.RFC822)

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
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
