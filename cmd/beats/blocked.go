package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/model"
	"github.com/spf13/cobra"
)

var (
	blockedByFlag     string
	blockedReasonFlag string
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

		if blockedByFlag != "" {
			payload.BlockedBy = &blockedByFlag
		}
		if blockedReasonFlag != "" {
			payload.BlockReason = &blockedReasonFlag
		}

		// Use "block" action for commit message inside UpdateIssue?
		// UpdateIssue takes 'action' string (e.g. "update", "start").
		// blocking logic uses "block".
		msgs, err := client.UpdateIssue(id, payload, "block")
		if err != nil {
			fmt.Printf("Error marking blocked: %v\n", err)
			os.Exit(1)
		}

		// Print messages or custom confirmation?
		// UpdateIssue returns "Updated <id>" default.
		// Detailed messages might be "Auto-started parent" etc.
		// For blocked, we might want "Marked <id> as BLOCKED".
		// But let's trust UpdateIssue messages + custom print if needed.
		// Or loop logic?
		// Usually tools print what happened.
		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

func init() {
	blockedCmd.Flags().StringVar(&blockedByFlag, "by", "", "ID of the issue blocking this one")
	blockedCmd.Flags().StringVar(&blockedReasonFlag, "reason", "", "Reason for blocking")
	rootCmd.AddCommand(blockedCmd)
}
