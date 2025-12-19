package main

import (
	// Actually bufio is not used I think? Let's check logic.
	// Logic uses strings.Split, exec, os, etc.
	// Oh, I added bufio but might not have used it.
	// Let's re-read code in memory.
	// `parseUpdateContent` uses strings.Split.
	// `openEditor` uses os, exec.
	// I don't see bufio usage in my added code.
	// `add.go` used bufio for confirmation.
	// `update.go` doesn't ask for confirmation in my added code.
	// So I can remove bufio.
	// But let's just make it compilable first.

	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	Use:               "update [id]",
	Short:             "Update an issue",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		// Check if any flags provided
		anyFlag := false
		if cmd.Flags().Changed("title") {
			anyFlag = true
		}
		if cmd.Flags().Changed("desc") {
			anyFlag = true
		}
		if cmd.Flags().Changed("status") {
			anyFlag = true
		}
		if cmd.Flags().Changed("parent") {
			anyFlag = true
		}
		if cmd.Flags().Changed("sp") {
			anyFlag = true
		}

		if !anyFlag {
			runInteractiveUpdate(id)
			return
		}

		// Construct payload from flags
		payload := model.UpdatePayload{}

		if cmd.Flags().Changed("title") {
			payload.Title = &updateTitle
		}
		if cmd.Flags().Changed("desc") {
			payload.Description = &updateDesc
		}
		if cmd.Flags().Changed("status") {
			payload.Status = &updateStatus
		}
		if cmd.Flags().Changed("parent") {
			payload.ParentID = &updateParent
		}
		if cmd.Flags().Changed("sp") {
			payload.Estimate = &updateEstimate
		}

		runUpdate(id, payload, "update")
	},
}

// Shortcuts
var startCmd = &cobra.Command{
	Use:               "start [id]",
	Short:             "Start working on an issue (Status: DOING)",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		status := string(model.StatusDoing)
		payload := model.UpdatePayload{Status: &status}
		runUpdate(args[0], payload, "start")
	},
}

var doneCmd = &cobra.Command{
	Use:               "done [id]",
	Short:             "Complete an issue (Status: DONE)",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		status := string(model.StatusDone)
		payload := model.UpdatePayload{Status: &status}
		runUpdate(args[0], payload, "done")
	},
}

var plannedCmd = &cobra.Command{
	Use:               "planned [id]",
	Short:             "Mark issue as PLANNED",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
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
		if err := exec.Command("git", "add", ".beats/issues.db").Run(); err != nil {
			fmt.Printf("Error adding to git: %v\n", err)
		} else if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
			fmt.Printf("Error committing: %v\n", err)
		}
	}
}

func runInteractiveUpdate(id string) {
	// 1. Read current issue state
	events, err := storage.ReadEvents()
	if err != nil {
		fmt.Printf("Error reading events: %v\n", err)
		os.Exit(1)
	}
	issues := model.ProjectIssues(events)
	issue, exists := issues[id]
	if !exists {
		fmt.Printf("Issue %s not found\n", id)
		os.Exit(1)
	}

	// 2. Generate Template
	template := generateUpdateTemplate(issue)

	// 3. Open Editor
	content, err := openEditor(template)
	if err != nil {
		fmt.Printf("Error opening editor: %v\n", err)
		os.Exit(1)
	}

	// 4. Parse Content
	payload, err := parseUpdateContent(content, issue)
	if err != nil {
		fmt.Printf("Error parsing content: %v\n", err)
		os.Exit(1)
	}

	// 5. Check if empty (no changes)
	if payload.Title == nil && payload.Description == nil && payload.Status == nil &&
		payload.ParentID == nil && payload.Estimate == nil &&
		payload.BlockedBy == nil && payload.BlockReason == nil {
		fmt.Println("No changes detected.")
		return
	}

	// 6. Run Update
	runUpdate(id, payload, "update")
}

func generateUpdateTemplate(i *model.Issue) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Title: %s\n", i.Title))
	sb.WriteString(fmt.Sprintf("Status: %s\n", i.Status))
	sb.WriteString(fmt.Sprintf("Parent: %s\n", i.ParentID))
	sb.WriteString(fmt.Sprintf("Estimate: %d\n", i.Estimate))
	sb.WriteString(fmt.Sprintf("Blocked By: %s\n", i.BlockedBy))
	sb.WriteString(fmt.Sprintf("Block Reason: %s\n", i.BlockReason))
	sb.WriteString("\n") // Double newline separates headers from description
	sb.WriteString(i.Description)

	return sb.String()
}

func openEditor(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpFile, err := os.CreateTemp("", "beats-update-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(initialContent); err != nil {
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func parseUpdateContent(content string, original *model.Issue) (model.UpdatePayload, error) {
	// Split into Headers and Body
	parts := strings.SplitN(content, "\n\n", 2)

	headerBlock := parts[0]
	descriptionBlock := ""
	if len(parts) > 1 {
		descriptionBlock = parts[1]
	}

	// Parse Headers
	meta := make(map[string]string)
	headerLines := strings.Split(headerBlock, "\n")
	for _, line := range headerLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Expect "Key: Value"
		kp := strings.SplitN(line, ":", 2)
		if len(kp) == 2 {
			key := strings.TrimSpace(strings.ToLower(kp[0]))
			val := strings.TrimSpace(kp[1])
			meta[key] = val
		}
	}

	payload := model.UpdatePayload{}

	// Map headers to payload
	if val, ok := meta["title"]; ok {
		if val != original.Title {
			payload.Title = &val
		}
	}

	if val, ok := meta["status"]; ok {
		if val != string(original.Status) {
			payload.Status = &val
		}
	}
	if val, ok := meta["parent"]; ok {
		if val != original.ParentID {
			payload.ParentID = &val
		}
	}
	if val, ok := meta["estimate"]; ok {
		est, err := strconv.Atoi(val)
		if err == nil {
			if est != original.Estimate {
				payload.Estimate = &est
			}
		}
	}
	if val, ok := meta["blocked by"]; ok {
		if val != original.BlockedBy {
			payload.BlockedBy = &val
		}
	}
	if val, ok := meta["block reason"]; ok {
		if val != original.BlockReason {
			payload.BlockReason = &val
		}
	}

	// Handle Description (No comment stripping logic applied anymore)
	newDesc := strings.TrimSpace(descriptionBlock)
	if newDesc != original.Description {
		payload.Description = &newDesc
	}

	return payload, nil
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
