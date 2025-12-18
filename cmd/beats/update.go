package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
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
	// 1. Read and Project State
	events, err := storage.ReadEvents()
	if err != nil {
		fmt.Printf("Error reading events: %v\n", err)
		os.Exit(1)
	}
	issues := model.ProjectIssues(events)

	targetIssue, exists := issues[id]
	if !exists {
		fmt.Printf("Issue %s not found\n", id)
		os.Exit(1)
	}

	user := getUser()
	timestamp := time.Now().UTC()
	var eventsToAppend []model.Event
	var messages []string

	// 2. Prepare Primary Update
	primaryEvent := model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   payload,
		CreatedAt: timestamp,
		CreatedBy: user,
	}
	eventsToAppend = append(eventsToAppend, primaryEvent)
	messages = append(messages, fmt.Sprintf("Updated %s", id))

	// 3. Logic & Side Effects based on Status Change
	if payload.Status != nil {
		newStatus := model.IssueStatus(*payload.Status)

		// --- Scenario: Start Child -> Auto-Start Parent ---
		if newStatus == model.StatusDoing && targetIssue.ParentID != "" {
			parent, pExists := issues[targetIssue.ParentID]
			if pExists && parent.Status != model.StatusDoing && parent.Status != model.StatusDone {
				// Auto-start parent
				pStatus := string(model.StatusDoing)
				parentEvent := model.Event{
					ID:        parent.ID,
					Type:      model.EventTypeUpdate,
					Payload:   model.UpdatePayload{Status: &pStatus},
					CreatedAt: timestamp, // Logical simultaneity
					CreatedBy: user,
				}
				eventsToAppend = append(eventsToAppend, parentEvent)
				messages = append(messages, fmt.Sprintf("Auto-started parent epic %s", parent.ID))
			}
		}

		// --- Scenario: Manual Complete Epic -> Validation ---
		if newStatus == model.StatusDone {
			// Check if this issue is a parent with incomplete children
			hasIncompleteChildren := false
			for _, child := range issues {
				if child.ParentID == id && child.Status != model.StatusDone {
					hasIncompleteChildren = true
					break
				}
			}
			if hasIncompleteChildren {
				fmt.Printf("Error: Cannot complete epic %s because it has unfinished child tasks.\n", id)
				os.Exit(1)
			}
		}

		// --- Scenario: Complete Child -> Auto-Complete Parent ---
		if newStatus == model.StatusDone && targetIssue.ParentID != "" {
			parent, pExists := issues[targetIssue.ParentID]
			if pExists && parent.Status != model.StatusDone {
				// Check if ALL OTHER children are done
				allSiblingsDone := true
				for _, other := range issues {
					if other.ParentID == parent.ID && other.ID != id { // Skip self (we are becoming done)
						if other.Status != model.StatusDone {
							allSiblingsDone = false
							break
						}
					}
				}

				if allSiblingsDone {
					// Auto-complete parent
					pStatus := string(model.StatusDone)
					parentEvent := model.Event{
						ID:        parent.ID,
						Type:      model.EventTypeUpdate,
						Payload:   model.UpdatePayload{Status: &pStatus},
						CreatedAt: timestamp,
						CreatedBy: user,
					}
					eventsToAppend = append(eventsToAppend, parentEvent)
					messages = append(messages, fmt.Sprintf("Auto-completed parent epic %s (all children done)", parent.ID))
				}
			}
		}
	}

	// 4. Commit Changes
	for _, evt := range eventsToAppend {
		if err := storage.AppendEvent(evt); err != nil {
			fmt.Printf("Error appending event for %s: %v\n", evt.ID, err)
			os.Exit(1)
		}
	}

	// 5. Output Messages
	for _, msg := range messages {
		fmt.Println(msg)
	}

	// 6. Git Commit (if enabled)
	if cfg.AutoCommit {
		// Just simple commit message for the main action, maybe mentions side effects?
		// Keeping it simple: "beats: <action> <id>"
		commitMsg := fmt.Sprintf("beats: %s %s", action, id)

		// If side effects occurred, maybe append to msg?
		if len(eventsToAppend) > 1 {
			commitMsg += " (with cascading updates)"
		}

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
