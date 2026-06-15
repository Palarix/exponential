package main

import (
	"fmt"
	"os"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/ui"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the .beats directory",
	Run: func(cmd *cobra.Command, args []string) {
		var notes []string

		// --- PHASE 1: Initialization ---
		res, err := beats.InitBeats(initForce)
		if err != nil {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s %v\n", ui.ErrorPrefix, err)))
			os.Exit(1)
		}

		if !res.Created {
			// It existed (and force was used/checked inside? No, checks force inside)
			// Wait, InitBeats returns error if exists and !force.
			// If res.Created is false but no error, it means we re-initialized (force=true).
		}

		// Print notes from initialization
		for _, note := range res.Notes {
			// Heuristic: if note starts with "Configured", it's OK. If "Could not", it's checks.
			// Or just print them?
			// The original code printed specific prefixed messages.
			// Let's just print them as info or warnings.
			// Actually InitBeats returns notes string list.
			// We can categorize them or just print.
			fmt.Print(ui.Stylize(fmt.Sprintf("%s %s\n", ui.OKPrefix, note)))
		}
		if res.Created {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Created `.beats` directory\n", ui.OKPrefix)))
		}

		// --- PHASE 2: Checks & Warnings ---

		// 5. Check if inside git repo
		if !beats.CheckGitRepo() {
			notes = append(notes, "Not a git repository")
		}

		// 6. Check for .github/workflows
		if beats.CheckGithubWorkflows() {
			notes = append(notes, "Github workflows detected")
		}

		// 7. Check for existing git hooks
		activeHooks := beats.CheckGitHooks()
		if len(activeHooks) > 0 {
			notes = append(notes, "Existing git hooks found")
		}

		// 8. Check .mcp.json
		mcpStatus := beats.DetectMCPConfig()
		if mcpStatus.HasBeats {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s MCP config (`%s`) has beats entry\n", ui.OKPrefix, ".mcp.json")))
		} else if mcpStatus.Exists {
			notes = append(notes, ui.Stylize("`.mcp.json` exists but has no beats entry"))
		} else {
			notes = append(notes, "`.mcp.json` not found")
		}

		// 9. Detect AI agent instruction files
		results := beats.DetectAgentFiles()
		var detected []string
		for _, r := range results {
			if r.Exists {
				if r.HasBeatsConfig {
					detected = append(detected, ui.Stylize(fmt.Sprintf("%s (`%s`)", r.Agent.Name, r.Agent.File)))
				} else {
					notes = append(notes, ui.Stylize(fmt.Sprintf("Agent file `%s` needs beats config", r.Agent.File)))
				}
			}
		}

		if len(detected) > 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Detected configured agents:\n", ui.OKPrefix)))
			for _, d := range detected {
				fmt.Printf("    - %s\n", d)
			}
		}

		// 10. Check shell completion
		compRes := beats.CheckCompletionConfig()
		if !compRes.Configured && compRes.Shell != "unknown" {
			notes = append(notes, ui.Stylize(fmt.Sprintf("Shell completion for `%s` is not configured", compRes.Shell)))
		}

		if len(notes) > 0 {
			for _, note := range notes {
				fmt.Printf("%s %s\n", ui.NotePrefix, note)
			}
			fmt.Print(ui.Stylize("\nRun `beats doctor` to see details and fix these issues.\n"))
		}

		// --- PHASE 3: Summary ---
		fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Initialized `.beats` successfully!\n", ui.OKPrefix)))
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Re-initialize even if .beats already exists")
	rootCmd.AddCommand(initCmd)
}
