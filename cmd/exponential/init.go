package main

import (
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the .xpo directory",
	Long: `Initialize the .xpo directory with project configuration.

This creates the .xpo directory, config.yaml, issues database, and git configuration.
It does NOT install agent skills or MCP configuration — use the subcommands for that:

  xpo init mcp     Configure MCP server for your agent harness
  xpo init skill   Install workflow skill and agent instructions`,
	Run: func(cmd *cobra.Command, args []string) {
		var notes []string

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
		} else if !initForce {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s `.xpo` directory already exists, configuration verified\n", ui.OKPrefix)))
		}

		// --- Checks & Warnings ---

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

		// --- Summary ---
		fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Initialized `.xpo` successfully!\n", ui.OKPrefix)))
		fmt.Print(ui.Stylize("Customize your project configuration in `.xpo/config.yaml`\n"))

		// --- Hints for next steps ---
		mcpStatus := exponential.DetectMCPConfig()
		agents := exponential.DetectInstalledAgents()
		hasSkills := false
		for _, a := range agents {
			if s := exponential.DetectSkillInstall(a); s.Local || s.Global {
				hasSkills = true
				break
			}
		}

		if !mcpStatus.HasExponential || !hasSkills {
			fmt.Println()
			fmt.Print(ui.Stylize("Next steps:\n"))
			if !mcpStatus.HasExponential {
				fmt.Print(ui.Stylize(fmt.Sprintf("  %s Run `xpo init mcp` to configure the MCP server for your agent harness\n", ui.NotePrefix)))
			}
			if !hasSkills {
				fmt.Print(ui.Stylize(fmt.Sprintf("  %s Run `xpo init skill` to install the workflow skill and agent instructions\n", ui.NotePrefix)))
			}
		}
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Re-initialize even if .xpo already exists")
	rootCmd.AddCommand(initCmd)
}
