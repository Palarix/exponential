package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var listStatusFlag string
var listAllFlag bool
var listSinceFlag string
var listBeforeFlag string
var listMatchFlag string
var listMineFlag bool
var listEpicFlag string
var listArchivedFlag bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List issues",
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)

		opts := beats.FilterOptions{
			Since:    listSinceFlag,
			Before:   listBeforeFlag,
			Match:    listMatchFlag,
			Mine:     listMineFlag,
			EpicID:   listEpicFlag,
			All:      listAllFlag,
			Archived: listArchivedFlag,
		}

		if listStatusFlag != "" {
			parts := strings.Split(listStatusFlag, ",")
			for _, p := range parts {
				opts.Statuses = append(opts.Statuses, strings.TrimSpace(p))
			}
		}

		issues, err := client.ListIssues(opts)
		if err != nil {
			fmt.Printf("Error listing issues: %v\n", err)
			os.Exit(1)
		}

		// Detect Terminal Width
		termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || termWidth <= 0 {
			termWidth = 120 // Fallback
		}

		ui.RenderIssueList(issues, termWidth)
	},
}

func init() {
	listCmd.Flags().StringVar(&listStatusFlag, "status", "", "Filter by status")
	listCmd.Flags().BoolVarP(&listAllFlag, "all", "a", false, "Show all issues (including DONE)")
	listCmd.Flags().StringVar(&listSinceFlag, "since", "", "Show issues created since duration/date (e.g. 24h, 2024-01-01)")
	listCmd.Flags().StringVar(&listBeforeFlag, "before", "", "Show issues created before duration/date")
	listCmd.Flags().StringVarP(&listMatchFlag, "match", "m", "", "Search for string in ID, title, status, etc.")
	listCmd.Flags().BoolVar(&listMineFlag, "mine", false, "Show issues created by current user")
	listCmd.Flags().StringVar(&listEpicFlag, "epic", "", "Filter by child of epic ID")
	listCmd.Flags().BoolVar(&listArchivedFlag, "archived", false, "Include archived issues")
	rootCmd.AddCommand(listCmd)
}
