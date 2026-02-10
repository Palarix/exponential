package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/model"
	"github.com/spf13/cobra"
)

var blockedCmd = &cobra.Command{
	Use:               "blocked [id]",
	Short:             "Mark an issue as BLOCKED",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		client := beats.NewClient(cfg)

		status := string(model.StatusBlocked)
		payload := model.UpdatePayload{
			Status: &status,
		}

		msgs, err := client.UpdateIssue(id, payload, "block")
		if err != nil {
			fmt.Printf("Error marking blocked: %v\n", err)
			os.Exit(1)
		}

		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

func init() {
	rootCmd.AddCommand(blockedCmd)
}
