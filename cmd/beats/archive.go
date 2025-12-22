package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/beats/internal/beats"
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
		client := beats.NewClient(cfg)

		stats, err := client.GetArchiveStats(archiveDays, archiveKeep)
		if err != nil {
			fmt.Printf("Error calculating archive stats: %v\n", err)
			os.Exit(1)
		}

		if stats.TotalArchive == 0 {
			fmt.Println("No issues to archive.")
			return
		}

		fmt.Printf("This will move %d issues to archive.db (%d DONE, %d DELETED).\n", stats.TotalArchive, stats.DoneCount, stats.DeletedCount)
		fmt.Printf("Keeping the last %d DONE issues active.\n", stats.KeptCount)

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

		if err := client.PerformArchive(stats); err != nil {
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
