package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
		prefix := choosePrefix()

		res, err := exponential.InitProject(initForce, prefix)
		if err != nil {
			fmt.Printf("%s %v\n", ui.ErrorPrefix, err)
			os.Exit(1)
		}

		for _, note := range res.Notes {
			fmt.Printf("%s %s\n", ui.OKPrefix, note)
		}
		if res.Created {
			fmt.Printf("%s Created .xpo directory\n", ui.OKPrefix)
		} else if !initForce {
			fmt.Printf("%s .xpo directory verified\n", ui.OKPrefix)
		}

		// --- Quick health check ---
		var warnings int

		if !exponential.CheckGitRepo() {
			fmt.Printf("%s Not a git repository\n", ui.NotePrefix)
			warnings++
		}

		hasMCP := false
		agents := exponential.DetectInstalledAgents()
		for _, agent := range agents {
			if agent.MCPConfig.HasMCPConfig() {
				if status := exponential.DetectMCPConfigFor(agent.MCPConfig); status.HasExponential {
					hasMCP = true
					break
				}
			}
		}
		if !hasMCP {
			fmt.Printf("%s MCP server not configured\n", ui.NotePrefix)
			warnings++
		}

		hasSkills := false
		for _, agent := range agents {
			if agent.SkillDir == "" {
				continue
			}
			if status := exponential.DetectSkillInstall(agent); status.Local || status.Global {
				hasSkills = true
				break
			}
		}
		if !hasSkills {
			fmt.Printf("%s Agent skill not installed\n", ui.NotePrefix)
			warnings++
		}

		if compRes := exponential.CheckCompletionConfig(); !compRes.Configured && compRes.Shell != "unknown" {
			fmt.Printf("%s Shell completion not configured\n", ui.NotePrefix)
			warnings++
		}

		fmt.Printf("\nInitialized successfully. Edit .xpo/config.yaml to customize.\n")

		if warnings > 0 {
			fmt.Println("\nRun xpo doctor to address any warnings.")
		}
	},
}

func choosePrefix() string {
	suggested := exponential.DefaultPrefix()

	// Non-interactive or re-init: use the default
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return suggested
	}

	// If .xpo already exists and not forcing, don't prompt — config already has a prefix
	if _, err := os.Stat(".xpo"); err == nil && !initForce {
		return suggested
	}

	fmt.Printf("Issue ID prefix [%s]: ", suggested)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response == "" {
		return suggested
	}

	if !strings.HasSuffix(response, "-") {
		response += "-"
	}
	return response
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Re-initialize even if .xpo already exists")
	rootCmd.AddCommand(initCmd)
}
