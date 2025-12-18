package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/spf13/cobra"
)

var listStatusFlag string

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List issues",
	Run: func(cmd *cobra.Command, args []string) {
		events, err := storage.ReadEvents()
		if err != nil {
			fmt.Printf("Error reading events: %v\n", err)
			os.Exit(1)
		}

		issuesMap := model.ProjectIssues(events)
		issues := model.SortIssues(issuesMap)

		columns := []table.Column{
			{Title: "ID", Width: 12},
			{Title: "Type", Width: 8},
			{Title: "Status", Width: 10},
			{Title: "Title", Width: 30},
			{Title: "Parent", Width: 10},
			{Title: "Created", Width: 15},
			{Title: "By", Width: 20},
		}

		rows := []table.Row{}
		for _, i := range issues {
			if listStatusFlag != "" && string(i.Status) != listStatusFlag {
				continue
			}

			// Parse time for humanize
			tParsed, _ := time.Parse("2006-01-02 15:04", i.CreatedAt)
			relTime := humanize.Time(tParsed)

			rows = append(rows, table.Row{
				i.ID,
				i.Kind,
				string(i.Status),
				i.Title,
				i.ParentID,
				relTime,
				i.CreatedBy,
			})
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(false),
			table.WithHeight(len(rows)+1),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(true)
		s.Selected = s.Cell
		t.SetStyles(s)

		fmt.Println(t.View())
	},
}

func init() {
	listCmd.Flags().StringVar(&listStatusFlag, "status", "", "Filter by status")
	rootCmd.AddCommand(listCmd)
}
