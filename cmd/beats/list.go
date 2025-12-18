package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
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
			{Title: "ID", Width: 10},
			{Title: "Type", Width: 10},
			{Title: "Status", Width: 10},
			{Title: "Title", Width: 40},
			{Title: "Parent", Width: 10},
		}

		rows := []table.Row{}
		for _, i := range issues {
			if listStatusFlag != "" && string(i.Status) != listStatusFlag {
				continue
			}
			rows = append(rows, table.Row{
				i.ID,
				i.Kind,
				string(i.Status),
				i.Title,
				i.ParentID,
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
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		t.SetStyles(s)

		fmt.Println(t.View())
	},
}

func init() {
	listCmd.Flags().StringVar(&listStatusFlag, "status", "", "Filter by status")
	rootCmd.AddCommand(listCmd)
}
