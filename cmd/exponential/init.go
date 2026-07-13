package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the .xpo directory",
	Run: func(cmd *cobra.Command, args []string) {
		var notes []string

		// --- PHASE 1: Initialization ---
		res, err := exponential.InitProject(initForce)
		if err != nil {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s %v\n", ui.ErrorPrefix, err)))
			os.Exit(1)
		}

		for _, note := range res.Notes {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s %s\n", ui.OKPrefix, note)))
		}
		if res.Created {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Created `.xpo` directory\n", ui.OKPrefix)))
		}

		// --- PHASE 2: Create .mcp.json and agent instructions ---

		prefix := "issue-"
		if freshCfg, err := config.LoadConfig(); err == nil && freshCfg.Prefix != "" {
			prefix = freshCfg.Prefix
		}

		if err := exponential.EnsureMCPConfig(); err != nil {
			notes = append(notes, fmt.Sprintf("Could not configure `.mcp.json`: %v", err))
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Configured `.mcp.json` with xpo MCP server\n", ui.OKPrefix)))
		}

		agents := exponential.DetectInstalledAgents()
		for _, agent := range agents {
			agentContent, _ := os.ReadFile(agent.File)
			if strings.Contains(string(agentContent), "# Exponential Agent Instructions") {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s `%s` already has xpo instructions\n", ui.OKPrefix, agent.File)))
			} else if err := exponential.AppendAgentInstructions(agent, prefix); err != nil {
				notes = append(notes, fmt.Sprintf("Could not create `%s`: %v", agent.File, err))
			} else {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Created `%s` with agent instructions (%s)\n", ui.OKPrefix, agent.File, agent.Name)))
			}

			if skillDir, err := exponential.WriteAgentSkill(agent); err != nil {
				notes = append(notes, fmt.Sprintf("Could not write skill to `%s`: %v", agent.SkillDir, err))
			} else if skillDir != "" {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Created `%s/` workflow skill (%s)\n", ui.OKPrefix, skillDir, agent.Name)))
			}
		}

		// --- PHASE 3: Checks & Warnings ---

		if !exponential.CheckGitRepo() {
			notes = append(notes, "Not a git repository")
		}

		if exponential.CheckGithubWorkflows() {
			notes = append(notes, "Github workflows detected")
		}

		activeHooks := exponential.CheckGitHooks()
		if len(activeHooks) > 0 {
			notes = append(notes, "Existing git hooks found")
		}

		compRes := exponential.CheckCompletionConfig()
		if !compRes.Configured && compRes.Shell != "unknown" {
			notes = append(notes, ui.Stylize(fmt.Sprintf("Shell completion for `%s` is not configured", compRes.Shell)))
		}

		if len(notes) > 0 {
			for _, note := range notes {
				fmt.Printf("%s %s\n", ui.NotePrefix, note)
			}
			fmt.Print(ui.Stylize("\nRun `xpo doctor` to see details and fix these issues.\n"))
		}

		// --- PHASE 4: Summary ---
		fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Initialized `.xpo` successfully!\n", ui.OKPrefix)))
		fmt.Print(ui.Stylize("Customize your project configuration in `.xpo/config.yaml`\n"))
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Re-initialize even if .xpo already exists")
	rootCmd.AddCommand(initCmd)
}
