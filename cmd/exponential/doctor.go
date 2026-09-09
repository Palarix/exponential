package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/palarix/exponential/internal/version"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and fix xpo configuration issues",
	Long: `Check for common issues across project configuration, MCP setup, and agent
integration, and optionally fix them.

Checks are organized into three sections:
  1. Project Configuration  (fix with: xpo init)
  2. MCP Configuration      (fix with: xpo init mcp)
  3. Agent Integration       (fix with: xpo init skill)`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("xpo doctor — checking configuration...")
		fmt.Println()

		prefix := "issue-"
		if cfg != nil && cfg.Prefix != "" {
			prefix = cfg.Prefix
		}

		doctorProjectConfig()
		doctorMCPConfig()
		doctorAgentIntegration(prefix)
		doctorShellCompletion()

		fmt.Println("\nCheck complete!")
	},
}

func doctorProjectConfig() {
	fmt.Print(ui.Stylize("── Project Configuration ── (fix with: `xpo init`)\n\n"))

	if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s `.xpo` directory not found\n", ui.ErrorPrefix)))
		fmt.Print(ui.Stylize("    Run `xpo init` to initialize your project.\n\n"))
		return
	}
	fmt.Print(ui.Stylize(fmt.Sprintf("%s `.xpo` directory exists\n", ui.OKPrefix)))

	if cfg != nil {
		if cfg.Version < version.DataModelVersion {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Data Model Version mismatch (Project: v%d, CLI expects: v%d)\n", ui.ErrorPrefix, cfg.Version, version.DataModelVersion)))
			fmt.Print(ui.Stylize("    Run `xpo migrate` to update your project data model.\n"))
		} else if cfg.Version > version.DataModelVersion {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Data Model Version mismatch (Project: v%d, CLI expects: v%d)\n", ui.ErrorPrefix, cfg.Version, version.DataModelVersion)))
			fmt.Print(ui.Stylize("    Your CLI version is too old. Please upgrade exponential.\n"))
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Data Model Version matches (v%d)\n", ui.OKPrefix, cfg.Version)))
		}
	} else {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Could not verify Data Model Version (config not loaded)\n", ui.NotePrefix)))
	}

	if !exponential.CheckGitRepo() {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Not a git repository\n", ui.NotePrefix)))
		fmt.Print(ui.Stylize("    You risk losing your issues database if you delete this directory.\n"))
		if isInteractive() {
			fmt.Print(ui.Stylize("\nInitialize git repository? [y/N]: "))
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response == "y" || response == "yes" {
				gitCmd := exec.Command("git", "init")
				if out, err := gitCmd.CombinedOutput(); err != nil {
					fmt.Print(ui.Stylize(fmt.Sprintf("%s Error initializing git: %v\n", ui.ErrorPrefix, err)))
					fmt.Println(string(out))
				} else {
					fmt.Print(ui.Stylize(fmt.Sprintf("%s Initialized git repository\n", ui.OKPrefix)))
				}
			} else {
				fmt.Println("Skipped.")
			}
		} else {
			fmt.Print(ui.Stylize("    Run `git init` to track your project.\n"))
		}
	} else {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Git repository detected\n", ui.OKPrefix)))
	}

	issuesDB := filepath.Join(".xpo", "issues.db")
	if _, err := os.Stat(issuesDB); os.IsNotExist(err) {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Exponential database not found\n", ui.ErrorPrefix)))
	} else {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Exponential database exists\n", ui.OKPrefix)))
	}

	requiredIgnores := []string{".xpo/issues.snapshot.json", ".xpo/git.lock", ".xpo/worktrees/"}
	gitignoreContent, _ := os.ReadFile(".gitignore")
	gitignoreStr := string(gitignoreContent)
	var missingIgnores []string
	for _, entry := range requiredIgnores {
		if !strings.Contains(gitignoreStr, entry) {
			missingIgnores = append(missingIgnores, entry)
		}
	}
	if len(missingIgnores) > 0 {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s `.gitignore` missing entries: %s\n", ui.NotePrefix, strings.Join(missingIgnores, ", "))))
		f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Could not update `.gitignore`: %v\n", ui.ErrorPrefix, err)))
		} else {
			if len(gitignoreStr) > 0 && !strings.HasSuffix(gitignoreStr, "\n") {
				f.WriteString("\n")
			}
			for _, entry := range missingIgnores {
				f.WriteString(entry + "\n")
			}
			f.Close()
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Added missing entries to `.gitignore`\n", ui.OKPrefix)))
		}
	} else {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s `.gitignore` has required xpo entries\n", ui.OKPrefix)))
	}

	gitattrsEntry := ".xpo/issues.db merge=union"
	gitattrsContent, _ := os.ReadFile(".gitattributes")
	gitattrsStr := string(gitattrsContent)
	if !strings.Contains(gitattrsStr, gitattrsEntry) {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s `.gitattributes` missing `merge=union` for issues.db\n", ui.NotePrefix)))
		exponential.EnsureGitattributesEntry(gitattrsEntry)
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Added `merge=union` rule to `.gitattributes`\n", ui.OKPrefix)))
	} else {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s `.gitattributes` has `merge=union` for issues.db\n", ui.OKPrefix)))
	}

	fmt.Println()
}

func doctorMCPConfig() {
	fmt.Print(ui.Stylize("── MCP Configuration ── (fix with: `xpo init mcp`)\n\n"))

	agents := exponential.DetectInstalledAgents()
	hasMCPAgent := false

	for _, agent := range agents {
		if !agent.MCPConfig.HasMCPConfig() {
			continue
		}
		hasMCPAgent = true

		status := exponential.DetectMCPConfigFor(agent.MCPConfig)
		if status.HasExponential {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s `%s` has xpo entry (%s)\n", ui.OKPrefix, agent.MCPConfig.File, agent.Name)))
		} else if status.Exists {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s `%s` exists but has no xpo entry (%s)\n", ui.NotePrefix, agent.MCPConfig.File, agent.Name)))
			fmt.Print(ui.Stylize("    Run `xpo init mcp` to add it.\n"))
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s `%s` not found (%s)\n", ui.NotePrefix, agent.MCPConfig.File, agent.Name)))
			fmt.Print(ui.Stylize("    Run `xpo init mcp` to create it.\n"))
		}
	}

	if !hasMCPAgent {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s No agent harnesses with MCP support detected\n", ui.NotePrefix)))
	}

	fmt.Println()
}

func doctorAgentIntegration(prefix string) {
	fmt.Print(ui.Stylize("── Agent Integration ── (fix with: `xpo init skill`)\n\n"))

	results := exponential.DetectAgentFiles()
	var configured, needsConfig []exponential.AgentDetectionResult

	for _, r := range results {
		if r.Exists {
			if r.HasExponentialConfig {
				configured = append(configured, r)
			} else {
				needsConfig = append(needsConfig, r)
			}
		}
	}

	if len(configured) > 0 {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Agent files with xpo config:\n", ui.OKPrefix)))
		for _, r := range configured {
			fmt.Print(ui.Stylize(fmt.Sprintf("    • %s (`%s`)\n", r.Agent.Name, r.Agent.File)))
		}
	}

	if len(needsConfig) > 0 {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Agent files needing xpo config:\n", ui.NotePrefix)))
		for _, r := range needsConfig {
			fmt.Print(ui.Stylize(fmt.Sprintf("    • %s (`%s`)\n", r.Agent.Name, r.Agent.File)))
		}
		fmt.Print(ui.Stylize("    Run `xpo init skill` to add xpo instructions.\n"))
	}

	if len(configured) == 0 && len(needsConfig) == 0 {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s No agent instruction files detected\n", ui.NotePrefix)))
		fmt.Print(ui.Stylize("    Run `xpo init skill` to create agent instructions.\n"))
	}

	agents := exponential.DetectInstalledAgents()
	hasSkillAgent := false
	for _, agent := range agents {
		if agent.SkillDir == "" {
			continue
		}
		hasSkillAgent = true

		status := exponential.DetectSkillInstall(agent)
		switch {
		case status.BrokenSymlink:
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Skill for %s: installed globally (symlink broken)\n", ui.ErrorPrefix, agent.Name)))
			fmt.Print(ui.Stylize("    Run `xpo init skill --global --force` to repair.\n"))
		case status.Global:
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Skill for %s: installed globally\n", ui.OKPrefix, agent.Name)))
		case status.Local:
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Skill for %s: installed locally\n", ui.OKPrefix, agent.Name)))
		default:
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Skill for %s: not installed\n", ui.NotePrefix, agent.Name)))
			fmt.Print(ui.Stylize("    Run `xpo init skill` to install.\n"))
		}
	}

	if !hasSkillAgent {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s No agent harnesses with skill support detected\n", ui.NotePrefix)))
	}

	fmt.Println()
}

func doctorShellCompletion() {
	compRes := exponential.CheckCompletionConfig()
	if compRes.Shell != "unknown" {
		if compRes.Configured {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Shell completion for `%s` is configured\n", ui.OKPrefix, compRes.Shell)))
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Shell completion for `%s` is not configured\n", ui.NotePrefix, compRes.Shell)))
			cmd := exponential.GetCompletionInstallCmd(compRes.Shell)
			fmt.Println("    To enable completion, run this command and reload your shell:")
			fmt.Print(ui.Stylize(fmt.Sprintf("    `%s`\n", cmd)))
		}
	}
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
