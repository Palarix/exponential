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

// Update flags
var (
	updateTitle    string
	updateDesc     string
	updateStatus   string
	updateParent   string
	updateEstimate int
)

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		// Construct payload
		payload := model.UpdatePayload{}
		hasUpdate := false

		if cmd.Flags().Changed("title") {
			payload.Title = &updateTitle
			hasUpdate = true
		}
		if cmd.Flags().Changed("desc") {
			payload.Description = &updateDesc
			hasUpdate = true
		}
		if cmd.Flags().Changed("status") {
			// Validate status?
			payload.Status = &updateStatus
			hasUpdate = true
		}
		if cmd.Flags().Changed("parent") {
			payload.ParentID = &updateParent
			hasUpdate = true
		}
		if cmd.Flags().Changed("sp") {
			payload.Estimate = &updateEstimate
			hasUpdate = true
		}

		if !hasUpdate {
			fmt.Println("No updates provided")
			return
		}

		runUpdate(id, payload, "update")
	},
}

// Shortcuts
var startCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start working on an issue (Status: DOING)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		status := string(model.StatusDoing)
		payload := model.UpdatePayload{Status: &status}
		runUpdate(args[0], payload, "start")
	},
}

var doneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Complete an issue (Status: DONE)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		status := string(model.StatusDone)
		payload := model.UpdatePayload{Status: &status}
		runUpdate(args[0], payload, "done")
	},
}

var plannedCmd = &cobra.Command{
	Use:   "planned [id]",
	Short: "Mark issue as PLANNED",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		status := string(model.StatusPlanned)
		payload := model.UpdatePayload{Status: &status}
		runUpdate(args[0], payload, "planned")
	},
}

func runUpdate(id string, payload model.UpdatePayload, action string) {
	user := getUser()
	event := model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := storage.AppendEvent(event); err != nil {
		fmt.Printf("Error updating issue: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated %s\n", id)

	if cfg.AutoCommit {
		commitMsg := fmt.Sprintf("beats: %s %s", action, id)
		fmt.Println("Auto-committing...")
		if err := exec.Command("git", "add", ".beats/issues.jsonl").Run(); err != nil {
			fmt.Printf("Error adding to git: %v\n", err)
		} else if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
			fmt.Printf("Error committing: %v\n", err)
		}
	}
}

func init() {
	// Update flags
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "New title")
	updateCmd.Flags().StringVar(&updateDesc, "desc", "", "New description")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "New status")
	updateCmd.Flags().StringVar(&updateParent, "parent", "", "New parent ID")
	updateCmd.Flags().IntVar(&updateEstimate, "sp", 0, "New estimate")

	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(plannedCmd)
}
