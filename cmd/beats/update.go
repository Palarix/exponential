package main

import (
	"fmt"
	"os"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/ui"
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
	client := beats.NewClient(cfg) // cfg global from main.go/init?

	msgs, err := client.UpdateIssue(id, payload, action)
	if err != nil {
		fmt.Printf("Error updating issue: %v\n", err)
		os.Exit(1)
	}

	for _, msg := range msgs {
		fmt.Println(msg)
	}
}

func runInteractiveUpdate(id string) {
	client := beats.NewClient(cfg)

	// 1. Get current issue
	issue, err := client.GetIssue(id)
	if err != nil {
		fmt.Printf("Error getting issue: %v\n", err)
		os.Exit(1)
	}

	// 2. Generate Template
	template := beats.GenerateUpdateTemplate(issue)

	// 3. Open Editor
	content, err := ui.EditInteractive(template)
	if err != nil {
		fmt.Printf("Error opening editor: %v\n", err)
		os.Exit(1)
	}

	// 4. Parse Content
	payload, err := beats.ParseUpdateContent(content, issue)
	if err != nil {
		fmt.Printf("Error parsing content: %v\n", err)
		os.Exit(1)
	}

	// 5. Check if empty (no changes)
	// Actually ParseUpdateContent returns pointer to payload.
	// If fields are nil, no changes?
	// The implementation checks changes against original and sets field only if changed.
	// So if all fields are nil, then no changes.
	if payload.Title == nil && payload.Description == nil && payload.Status == nil &&
		payload.ParentID == nil && payload.Estimate == nil &&
		payload.BlockedBy == nil && payload.BlockReason == nil {
		fmt.Println("No changes detected.")
		return
	}

	// 6. Run Update
	// Note: dereference payload because runUpdate takes value? Or pointer?
	// runUpdate below takes model.UpdatePayload (struct), but Parse returns *UpdatePayload.
	// Let's defer to signature.
	// internal/beats/update.go: UpdateIssue(..., payload model.UpdatePayload, ...)
	// So we need to dereference: *payload.
	runUpdate(id, *payload, "update")
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
