package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/ui"
	"github.com/spf13/cobra"
)

var (
	listStatusFlag   []string
	listSinceFlag    string
	listBeforeFlag   string
	listMatchFlag    string
	listMineFlag     bool
	listParentFlag   string
	listLabelFlag    string
	listAssigneeFlag string
	listCycleFlag    string
	listAllFlag      bool
	listArchivedFlag bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all issues",
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)

		opts := beats.FilterOptions{
			Statuses: listStatusFlag,
			Since:    listSinceFlag,
			Before:   listBeforeFlag,
			Match:    listMatchFlag,
			Mine:     listMineFlag,
			ParentID: listParentFlag,
			Label:    listLabelFlag,
			Assignee: listAssigneeFlag,
			CycleID:  resolveCycleID(listCycleFlag),
			All:      listAllFlag,
			Archived: listArchivedFlag,
		}

		issues, err := client.ListIssues(opts)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if len(issues) == 0 {
			fmt.Println("No issues found.")
			return
		}

		// Detect terminal width
		width := ui.TerminalWidth()
		fmt.Print(ui.RenderIssueList(issues, width))
	},
}

func init() {
	listCmd.Flags().StringSliceVarP(&listStatusFlag, "status", "s", nil, "Filter by status (BACKLOG, PLANNED, DOING, BLOCKED, DONE)")
	listCmd.Flags().StringVar(&listSinceFlag, "since", "", "Show issues updated since (e.g. 1d, 1w, 2024-01-01)")
	listCmd.Flags().StringVar(&listBeforeFlag, "before", "", "Show issues updated before")
	listCmd.Flags().StringVarP(&listMatchFlag, "match", "m", "", "Search term to match against fields")
	listCmd.Flags().BoolVar(&listMineFlag, "mine", false, "Show only my issues")
	listCmd.Flags().StringVarP(&listParentFlag, "parent", "p", "", "Show children of a parent issue")
	listCmd.Flags().StringVarP(&listLabelFlag, "label", "l", "", "Filter by label")
	listCmd.Flags().StringVar(&listAssigneeFlag, "assignee", "", "Filter by assignee")
	listCmd.Flags().StringVar(&listCycleFlag, "cycle", "", "Filter by cycle (current, next, or YYYY-MM-DD)")
	listCmd.Flags().BoolVarP(&listAllFlag, "all", "a", false, "Show all issues (including old DONE)")
	listCmd.Flags().BoolVar(&listArchivedFlag, "archived", false, "Include archived issues")
	rootCmd.AddCommand(listCmd)
}
