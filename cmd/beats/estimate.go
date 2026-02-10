package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/model"
	"github.com/spf13/cobra"
)

var estimateCmd = &cobra.Command{
	Use:               "estimate [id] [points]",
	Short:             "Set the estimate for an issue",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		input := args[1]

		// Parse input (supports shirt-size labels for shirt system)
		points, err := config.ParseEstimateInput(cfg.EstimationSystem, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Validate against estimation system
		if err := config.ValidateEstimate(cfg.EstimationSystem, points); err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Allowed values for %s: %v\n", cfg.EstimationSystem, config.EstimationSystems[cfg.EstimationSystem])
			os.Exit(1)
		}

		client := beats.NewClient(cfg)
		estimate := points
		payload := model.UpdatePayload{
			Estimate: &estimate,
		}

		displayVal := config.EstimateDisplayValue(cfg.EstimationSystem, points)
		msgs, err := client.UpdateIssue(id, payload, fmt.Sprintf("estimate %s", displayVal))
		if err != nil {
			fmt.Printf("Error estimating issue: %v\n", err)
			os.Exit(1)
		}

		for _, msg := range msgs {
			fmt.Println(msg)
		}

		// Show confirmation using display value
		if cfg.EstimationSystem == "shirt" {
			fmt.Printf("Estimated %s at %s (%d pts)\n", id, displayVal, points)
		}
	},
}

func init() {
	rootCmd.AddCommand(estimateCmd)
}

// Unused but keeping for potential future use
var _ = strconv.Atoi
