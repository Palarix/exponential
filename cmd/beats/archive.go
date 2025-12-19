package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/spf13/cobra"
)

var archiveDays int
var archiveKeep int
var archiveYes bool

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive completed and deleted issues",
	Long:  `Moves old DONE issues and all DELETED issues to .beats/archive.db.`,
	Run: func(cmd *cobra.Command, args []string) {
		events, err := storage.ReadEvents()
		if err != nil {
			fmt.Printf("Error reading events: %v\n", err)
			os.Exit(1)
		}

		issues := model.ProjectIssues(events)

		// Identify candidates
		// 1. Deleted issues (Always archive)
		// 2. DONE issues older than X days, but keep at least Y most recent DONE issues.

		var deletedIDs []string
		var doneIssues []*model.Issue

		for _, i := range issues {
			if i.Deleted {
				deletedIDs = append(deletedIDs, i.ID)
			} else if i.Status == model.StatusDone {
				doneIssues = append(doneIssues, i)
			}
		}

		// Sort DONE issues by UpdatedAt (descending - newest first)
		sort.Slice(doneIssues, func(i, j int) bool {
			return doneIssues[i].UpdatedAt.After(doneIssues[j].UpdatedAt)
		})

		// Determine which DONE issues to archive
		var doneIDsToArchive []string
		cutoff := time.Now().AddDate(0, 0, -archiveDays)

		// Iterate through done issues
		// Keep at least `archiveKeep` count.
		// Archive if older than cutoff AND we have already kept enough.
		for idx, issue := range doneIssues {
			if idx < archiveKeep {
				// Keep this issue (it's one of the most recent ones)
				continue
			}
			// Past the keep limit. Check age.
			if issue.UpdatedAt.Before(cutoff) {
				doneIDsToArchive = append(doneIDsToArchive, issue.ID)
			}
		}

		idsToArchive := make(map[string]bool)
		for _, id := range deletedIDs {
			idsToArchive[id] = true
		}
		for _, id := range doneIDsToArchive {
			idsToArchive[id] = true
		}

		if len(idsToArchive) == 0 {
			fmt.Println("No issues to archive.")
			return
		}

		// Filter events
		var activeEvents []model.Event
		var archivedEvents []model.Event

		for _, evt := range events {
			if idsToArchive[evt.ID] {
				archivedEvents = append(archivedEvents, evt)
			} else {
				activeEvents = append(activeEvents, evt)
			}
		}

		// Calculate stats
		doneArchivedCount := len(doneIDsToArchive)
		deletedArchivedCount := len(deletedIDs)
		keptDoneCount := len(doneIssues) - doneArchivedCount

		fmt.Printf("This will move %d issues to archive.db (%d DONE, %d DELETED).\n", len(idsToArchive), doneArchivedCount, deletedArchivedCount)
		fmt.Printf("Keeping the last %d DONE issues active.\n", keptDoneCount)

		if !archiveYes {
			fmt.Print("Proceed? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return
			}
		}

		if err := storage.ArchiveEvents(activeEvents, archivedEvents); err != nil {
			fmt.Printf("Error archiving events: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Archive complete.")
	},
}

func init() {
	archiveCmd.Flags().IntVar(&archiveDays, "days", 30, "Archive DONE issues older than this many days")
	archiveCmd.Flags().IntVar(&archiveKeep, "keep", 50, "Number of recent DONE issues to keep in active db")
	archiveCmd.Flags().BoolVarP(&archiveYes, "yes", "y", false, "Skip confirmation prompt")
	rootCmd.AddCommand(archiveCmd)
}
