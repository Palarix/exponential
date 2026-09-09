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

type doctorCounts struct {
	ok    int
	warn  int
	fail  int
}

func (c *doctorCounts) pass(format string, a ...interface{}) {
	c.ok++
	fmt.Printf("%s %s\n", ui.OKPrefix, fmt.Sprintf(format, a...))
}

func (c *doctorCounts) note(format string, a ...interface{}) {
	c.warn++
	fmt.Printf("%s %s\n", ui.NotePrefix, fmt.Sprintf(format, a...))
}

func (c *doctorCounts) err(format string, a ...interface{}) {
	c.fail++
	fmt.Printf("%s %s\n", ui.ErrorPrefix, fmt.Sprintf(format, a...))
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and fix xpo configuration issues",
	Long: `Check for common issues across project configuration, MCP setup, and agent
integration, and optionally fix them.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("xpo doctor")
		fmt.Println()

		prefix := "issue-"
		if cfg != nil && cfg.Prefix != "" {
			prefix = cfg.Prefix
		}

		counts := &doctorCounts{}

		doctorProjectConfig(counts)
		doctorMCPConfig(counts)
		doctorAgentIntegration(counts, prefix)
		doctorShellCompletion(counts)

		fmt.Println()
		if counts.fail > 0 {
			fmt.Printf("%d passed, %d warnings, %d errors\n", counts.ok, counts.warn, counts.fail)
		} else if counts.warn > 0 {
			fmt.Printf("%d passed, %d warnings\n", counts.ok, counts.warn)
		} else {
			fmt.Printf("All %d checks passed.\n", counts.ok)
		}
	},
}

func doctorProjectConfig(c *doctorCounts) {
	fmt.Println("Project                             fix: xpo init")
	fmt.Println()

	if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
		c.err(".xpo directory not found — run xpo init")
		fmt.Println()
		return
	}
	c.pass(".xpo directory")

	if cfg != nil {
		if cfg.Version < version.DataModelVersion {
			c.err("Data model v%d (expected v%d) — run xpo migrate", cfg.Version, version.DataModelVersion)
		} else if cfg.Version > version.DataModelVersion {
			c.err("Data model v%d (CLI expects v%d) — upgrade xpo", cfg.Version, version.DataModelVersion)
		} else {
			c.pass("Data model v%d", cfg.Version)
		}
	} else {
		c.note("Could not verify data model version")
	}

	if !exponential.CheckGitRepo() {
		c.note("Not a git repository")
		if isInteractive() {
			fmt.Print("\n  Initialize git repository? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response == "y" || response == "yes" {
				gitCmd := exec.Command("git", "init")
				if out, err := gitCmd.CombinedOutput(); err != nil {
					c.err("git init failed: %v\n%s", err, string(out))
				} else {
					c.pass("Initialized git repository")
				}
			}
		}
	} else {
		c.pass("Git repository")
	}

	issuesDB := filepath.Join(".xpo", "issues.db")
	if _, err := os.Stat(issuesDB); os.IsNotExist(err) {
		c.err("Issues database not found")
	} else {
		c.pass("Issues database")
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
		f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			c.err(".gitignore: could not add missing entries: %v", err)
		} else {
			if len(gitignoreStr) > 0 && !strings.HasSuffix(gitignoreStr, "\n") {
				f.WriteString("\n")
			}
			for _, entry := range missingIgnores {
				f.WriteString(entry + "\n")
			}
			f.Close()
			c.pass(".gitignore (added missing entries)")
		}
	} else {
		c.pass(".gitignore")
	}

	gitattrsEntry := ".xpo/issues.db merge=union"
	gitattrsContent, _ := os.ReadFile(".gitattributes")
	if !strings.Contains(string(gitattrsContent), gitattrsEntry) {
		exponential.EnsureGitattributesEntry(gitattrsEntry)
		c.pass(".gitattributes (added merge=union)")
	} else {
		c.pass(".gitattributes")
	}

	fmt.Println()
}

func doctorMCPConfig(c *doctorCounts) {
	fmt.Println("MCP                                 fix: xpo init mcp")
	fmt.Println()

	agents := exponential.DetectInstalledAgents()
	hasMCPAgent := false
	needsFix := false

	for _, agent := range agents {
		if !agent.MCPConfig.HasMCPConfig() {
			continue
		}
		hasMCPAgent = true

		status := exponential.DetectMCPConfigFor(agent.MCPConfig)
		if status.HasExponential {
			c.pass("%s (%s)", agent.MCPConfig.File, agent.Name)
		} else {
			if status.Exists {
				c.note("%s — missing xpo entry (%s)", agent.MCPConfig.File, agent.Name)
			} else {
				c.note("%s — not found (%s)", agent.MCPConfig.File, agent.Name)
			}
			needsFix = true
		}
	}

	if !hasMCPAgent {
		c.note("No agent harnesses with MCP support detected")
	} else if needsFix {
		fmt.Println("    Run xpo init mcp to fix.")
	}

	fmt.Println()
}

func doctorAgentIntegration(c *doctorCounts, prefix string) {
	fmt.Println("Agent Integration                   fix: xpo init skill")
	fmt.Println()

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
		for _, r := range configured {
			c.pass("%s (%s)", r.Agent.File, r.Agent.Name)
		}
	}

	if len(needsConfig) > 0 {
		for _, r := range needsConfig {
			c.note("%s — missing xpo instructions (%s)", r.Agent.File, r.Agent.Name)
		}
	}

	if len(configured) == 0 && len(needsConfig) == 0 {
		c.note("No agent instruction files detected")
	}

	agents := exponential.DetectInstalledAgents()
	needsSkillFix := false
	for _, agent := range agents {
		if agent.SkillDir == "" {
			continue
		}

		status := exponential.DetectSkillInstall(agent)
		switch {
		case status.BrokenSymlink:
			c.err("%s skill — symlink broken", agent.Name)
			needsSkillFix = true
		case status.Global:
			c.pass("%s skill (global)", agent.Name)
		case status.Local:
			c.pass("%s skill (local)", agent.Name)
		default:
			c.note("%s skill — not installed", agent.Name)
			needsSkillFix = true
		}
	}

	if needsSkillFix || len(needsConfig) > 0 {
		fmt.Println("    Run xpo init skill to fix.")
	}

	fmt.Println()
}

func doctorShellCompletion(c *doctorCounts) {
	compRes := exponential.CheckCompletionConfig()
	if compRes.Shell == "unknown" {
		return
	}

	if compRes.Configured {
		c.pass("Shell completion (%s)", compRes.Shell)
	} else {
		c.note("Shell completion for %s not configured", compRes.Shell)
		cmd := exponential.GetCompletionInstallCmd(compRes.Shell)
		fmt.Printf("    %s\n", cmd)
	}
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
