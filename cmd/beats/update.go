package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/inputs"
	"github.com/palarix/beats/internal/model"
	"github.com/spf13/cobra"
)

// --- Update Command ---

var (
	updateStatusFlag   string
	updateParentFlag   string
	updateEstimateFlag int
	updateLabelFlag    []string
	updateAssigneeFlag string
	updateCycleFlag    string
	updateDescFlag     string
	updateJSONFlag     bool
)

var updateCmd = &cobra.Command{
	Use:               "update [id]",
	Short:             "Update an issue (flags or interactive editor)",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		client := beats.NewClient(cfg)

		// JSON payload mode: read a structured patch from stdin. Other
		// field flags are ignored when --json is set.
		if updateJSONFlag {
			content, err := readStdinExplicit()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			var input inputs.UpdateInput
			if err := inputs.DecodeStrict(content, &input); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			payload, err := input.ToUpdatePayload()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			if inputs.UpdatePayloadEmpty(payload) {
				fmt.Println("No changes in payload.")
				return
			}
			msgs, err := client.UpdateIssue(id, payload, "update")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			for _, msg := range msgs {
				fmt.Println(msg)
			}
			return
		}

		// Resolve --desc - explicit stdin opt-in before computing hasFlags.
		descFromStdin := false
		if cmd.Flags().Changed("desc") && updateDescFlag == "-" {
			content, err := readStdinExplicit()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			updateDescFlag = content
		}

		// Check if any flags were set
		hasFlags := cmd.Flags().Changed("status") || cmd.Flags().Changed("parent") ||
			cmd.Flags().Changed("sp") || cmd.Flags().Changed("desc") ||
			cmd.Flags().Changed("label") || cmd.Flags().Changed("assignee") ||
			cmd.Flags().Changed("cycle")

		// Auto-detect piped stdin when no flags were passed: treat as --desc.
		if !hasFlags && isStdinPiped() {
			content, err := readAllStdin()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			updateDescFlag = content
			descFromStdin = true
			hasFlags = true
		}

		if hasFlags {
			// Flag-based update
			payload := model.UpdatePayload{}

			if cmd.Flags().Changed("status") {
				payload.Status = &updateStatusFlag
			}
			if cmd.Flags().Changed("parent") {
				payload.ParentID = &updateParentFlag
			}
			if cmd.Flags().Changed("sp") {
				payload.Estimate = &updateEstimateFlag
			}
			if cmd.Flags().Changed("desc") || descFromStdin {
				payload.Description = &updateDescFlag
			}
			if cmd.Flags().Changed("label") {
				payload.Labels = updateLabelFlag
			}
			if cmd.Flags().Changed("assignee") {
				payload.Assignee = &updateAssigneeFlag
			}
			if cmd.Flags().Changed("cycle") {
				resolved := resolveCycleID(updateCycleFlag)
				payload.CycleID = &resolved
			}

			msgs, err := client.UpdateIssue(id, payload, "update")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			for _, msg := range msgs {
				fmt.Println(msg)
			}
		} else {
			// Interactive editor mode
			issue, err := client.GetIssue(id)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			template := beats.GenerateUpdateTemplate(issue)

			// Write to temp file
			tmpFile, err := os.CreateTemp("", "beats-edit-*.md")
			if err != nil {
				fmt.Printf("Error creating temp file: %v\n", err)
				os.Exit(1)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.WriteString(template)
			tmpFile.Close()

			// Open editor
			editor := cfg.Editor
			editorCmd := exec.Command(editor, tmpFile.Name())
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			if err := editorCmd.Run(); err != nil {
				fmt.Printf("Error running editor: %v\n", err)
				os.Exit(1)
			}

			// Read edited content
			editedContent, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				fmt.Printf("Error reading edited file: %v\n", err)
				os.Exit(1)
			}

			payload, err := beats.ParseUpdateContent(string(editedContent), issue)
			if err != nil {
				fmt.Printf("Error parsing: %v\n", err)
				os.Exit(1)
			}

			// Check for empty update
			if payload.Title == nil && payload.Description == nil &&
				payload.Status == nil && payload.Estimate == nil &&
				payload.ParentID == nil && payload.Assignee == nil &&
				payload.Labels == nil {
				fmt.Println("No changes detected.")
				return
			}

			msgs, err := client.UpdateIssue(id, *payload, "update")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			for _, msg := range msgs {
				fmt.Println(msg)
			}
		}
	},
}

// --- Shortcut Commands ---

var startCmd = &cobra.Command{
	Use:               "start [id]",
	Short:             "Start working on an issue (set to DOING)",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)
		status := string(model.StatusDoing)
		payload := model.UpdatePayload{Status: &status}
		msgs, err := client.UpdateIssue(args[0], payload, "start")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

var doneCmd = &cobra.Command{
	Use:               "done [id]",
	Short:             "Mark an issue as DONE",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)
		status := string(model.StatusDone)
		payload := model.UpdatePayload{Status: &status}
		msgs, err := client.UpdateIssue(args[0], payload, "done")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

var plannedCmd = &cobra.Command{
	Use:               "planned [id]",
	Short:             "Mark an issue as PLANNED",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)
		status := string(model.StatusPlanned)
		payload := model.UpdatePayload{Status: &status}
		msgs, err := client.UpdateIssue(args[0], payload, "planned")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateStatusFlag, "status", "", "New status")
	updateCmd.Flags().StringVarP(&updateParentFlag, "parent", "p", "", "Parent issue ID")
	updateCmd.Flags().IntVar(&updateEstimateFlag, "sp", 0, "Story points")
	updateCmd.Flags().StringVar(&updateDescFlag, "desc", "", "Description")
	updateCmd.Flags().StringSliceVar(&updateLabelFlag, "label", nil, "Labels")
	updateCmd.Flags().StringVar(&updateAssigneeFlag, "assignee", "", "Assignee")
	updateCmd.Flags().StringVar(&updateCycleFlag, "cycle", "", "Assign to cycle (current, next, none, or YYYY-MM-DD)")
	updateCmd.Flags().BoolVar(&updateJSONFlag, "json", false, "Read a structured update patch as JSON from stdin")

	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(plannedCmd)
}

func joinNonEmpty(parts ...string) string {
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, " ")
}
