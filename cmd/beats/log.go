package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kuyio/beats/internal/beats"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:               "log [id] [points]",
	Short:             "Log work burned on an issue",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		pointsStr := args[1]

		points, err := strconv.Atoi(pointsStr)
		if err != nil {
			fmt.Printf("Error: points must be an integer: %v\n", err)
			os.Exit(1)
		}

		client := beats.NewClient(cfg)
		if err := client.LogWork(id, points); err != nil {
			fmt.Printf("Error logging work: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Logged %d SP on %s\n", points, id)
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
