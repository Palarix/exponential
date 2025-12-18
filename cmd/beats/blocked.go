package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/spf13/cobra"
)

var (
	blockedByFlag     string
	blockedReasonFlag string
)

var blockedCmd = &cobra.Command{
	Use:   "blocked [id]",
	Short: "Mark an issue as BLOCKED",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		user := getUser()

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

		event := model.Event{
			ID:        id,
			Type:      model.EventTypeUpdate,
			Payload:   payload,
			CreatedAt: time.Now().UTC(),
			CreatedBy: user,
		}

		if err := storage.AppendEvent(event); err != nil {
			fmt.Printf("Error appending event: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Marked %s as BLOCKED\n", id)

		if cfg.AutoCommit {
			commitMsg := fmt.Sprintf("beats: block %s", id)
			fmt.Println("Auto-committing...")
			if err := exec.Command("git", "add", ".beats/issues.jsonl").Run(); err != nil {
				fmt.Printf("Error adding to git: %v\n", err)
			} else if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
				fmt.Printf("Error committing: %v\n", err)
			}
		}
	},
}

func init() {
	blockedCmd.Flags().StringVar(&blockedByFlag, "by", "", "ID of the issue blocking this one")
	blockedCmd.Flags().StringVar(&blockedReasonFlag, "reason", "", "Reason for blocking")
	rootCmd.AddCommand(blockedCmd)
}
