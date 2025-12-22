package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/model"
	"github.com/spf13/cobra"
)

var estimateCmd = &cobra.Command{
	Use:               "estimate [id] [points]",
	Short:             "Estimate story points for an issue",
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
		estimate := points
		payload := model.UpdatePayload{
			Estimate: &estimate,
		}

		msgs, err := client.UpdateIssue(id, payload, fmt.Sprintf("estimate %d", points))
		if err != nil {
			fmt.Printf("Error estimating issue: %v\n", err)
			os.Exit(1)
		}

		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

func init() {
	rootCmd.AddCommand(estimateCmd)
}
