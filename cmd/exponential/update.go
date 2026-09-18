package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
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
		client := exponential.NewClient(cfg)

		// JSON payload mode: read a structured patch from stdin. Other
		// field flags are ignored when --json is set.
		if updateJSONFlag {
			content, err := readStdinExplicit()
			if err != nil {
				exitJSONError(err)
			}
			var input jsonio.UpdateInput
			if err := jsonio.DecodeStrict(content, &input); err != nil {
				exitJSONError(err)
			}
			payload, err := input.ToUpdatePayload()
			if err != nil {
				exitJSONError(err)
			}
			if jsonio.UpdatePayloadEmpty(payload) {
				exitJSONError(fmt.Errorf("no fields set: provide at least one field to update"))
			}
			if err := client.ValidateUpdatePayload(&payload); err != nil {
				exitJSONError(err)
			}
			msgs, err := client.UpdateIssue(id, payload, "update")
			if err != nil {
				exitJSONError(err)
			}
			out := jsonio.UpdateOutput{ID: id, Messages: msgs}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)
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

			template := exponential.GenerateUpdateTemplate(issue)

			// Write to temp file
			tmpFile, err := os.CreateTemp("", "xpo-edit-*.md")
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

			payload, err := exponential.ParseUpdateContent(string(editedContent), issue)
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

var startForce bool
var startMode string
var startJSONFlag bool

var startCmd = &cobra.Command{
	Use:               "start [id]",
	Short:             "Start working on an issue (set to DOING + create worktree)",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		if startJSONFlag {
			runStartJSON()
			return
		}
		if len(args) == 0 {
			fmt.Println("Error: requires exactly 1 arg(s)")
			os.Exit(1)
		}
		switch startMode {
		case "":
			// no override — use global config
		case "worktree":
			cfg.Worktrees = true
		case "branch":
			cfg.Worktrees = false
		default:
			fmt.Printf("Error: invalid --mode %q: must be \"worktree\" or \"branch\"\n", startMode)
			os.Exit(1)
		}
		client := exponential.NewClient(cfg)
		_, _, msgs, err := client.StartWork(args[0], startForce)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

func runStartJSON() {
	content, err := readStdinExplicit()
	if err != nil {
		exitJSONError(err)
	}
	var input jsonio.StartToolInput
	if err := jsonio.DecodeStrict(content, &input); err != nil {
		exitJSONError(err)
	}
	if input.ID == "" {
		exitJSONError(fmt.Errorf("'id' is required"))
	}

	switch input.Mode {
	case "":
		// no override
	case "worktree":
		cfg.Worktrees = true
	case "branch":
		cfg.Worktrees = false
	default:
		exitJSONError(fmt.Errorf("invalid mode %q: must be \"worktree\" or \"branch\"", input.Mode))
	}

	client := exponential.NewClient(cfg)
	branch, wtPath, msgs, err := client.StartWork(input.ID, input.Force)
	if err != nil {
		exitJSONError(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(jsonio.StartOutput{ID: input.ID, Branch: branch, WorktreePath: wtPath, Messages: msgs})
}

var doneJSONFlag bool

var doneCmd = &cobra.Command{
	Use:               "done [id]",
	Short:             "Mark an issue as DONE",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)
		status := string(model.StatusDone)
		payload := model.UpdatePayload{Status: &status}
		msgs, err := client.UpdateIssue(args[0], payload, "done")
		if err != nil {
			if doneJSONFlag {
				exitJSONError(err)
			}
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if doneJSONFlag {
			out := jsonio.UpdateOutput{ID: args[0], Messages: msgs}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)
			return
		}
		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

var plannedJSONFlag bool

var plannedCmd = &cobra.Command{
	Use:               "planned [id]",
	Short:             "Mark an issue as PLANNED",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)
		status := string(model.StatusPlanned)
		payload := model.UpdatePayload{Status: &status}
		msgs, err := client.UpdateIssue(args[0], payload, "planned")
		if err != nil {
			if plannedJSONFlag {
				exitJSONError(err)
			}
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if plannedJSONFlag {
			out := jsonio.UpdateOutput{ID: args[0], Messages: msgs}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)
			return
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
	updateCmd.Flags().BoolVar(&updateJSONFlag, "json", false, "Read a structured payload as JSON from stdin")

	rootCmd.AddCommand(updateCmd)
	startCmd.Flags().BoolVar(&startForce, "force", false, "Take over an issue already in progress or with an existing branch")
	startCmd.Flags().StringVar(&startMode, "mode", "", "Create a \"worktree\" or a \"branch\" (overrides config)")
	startCmd.Flags().BoolVar(&startJSONFlag, "json", false, "Read a structured payload as JSON from stdin")
	rootCmd.AddCommand(startCmd)
	doneCmd.Flags().BoolVar(&doneJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(doneCmd)
	plannedCmd.Flags().BoolVar(&plannedJSONFlag, "json", false, "Output as JSON")
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
