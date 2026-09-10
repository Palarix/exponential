package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/storage"
	"github.com/palarix/exponential/internal/ui"
	"github.com/palarix/exponential/internal/version"
)

type attentionItem struct {
	message  string
	isError  bool // true = error (✗), false = warning (!)
}

type doctorCounts struct {
	ok         int
	warn       int
	fail       int
	info       int
	fixed      int
	attention  []attentionItem
}

func (c *doctorCounts) pass(format string, a ...interface{}) {
	c.ok++
	fmt.Printf("  %s %s\n", ui.OKPrefix, fmt.Sprintf(format, a...))
}

func (c *doctorCounts) note(format string, a ...interface{}) {
	c.warn++
	fmt.Printf("  %s %s\n", ui.NotePrefix, fmt.Sprintf(format, a...))
}

func (c *doctorCounts) err(format string, a ...interface{}) {
	c.fail++
	fmt.Printf("  %s %s\n", ui.ErrorPrefix, fmt.Sprintf(format, a...))
}

func (c *doctorCounts) hint(format string, a ...interface{}) {
	c.info++
	fmt.Printf("  %s %s\n", ui.InfoPrefix, fmt.Sprintf(format, a...))
}

var (
	doctorFix    bool
	doctorStrict bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and fix xpo configuration",
	Long: `Check project configuration, MCP setup, agent integration, and machine
setup for common issues. Use --fix to resolve what can be auto-fixed.`,
	Run: func(cmd *cobra.Command, args []string) {
		prefix := "issue"
		if cfg != nil && cfg.Prefix != "" {
			prefix = cfg.Prefix
		}
		integrationVer := exponential.DetectIntegrationVersion()

		counts := &doctorCounts{}
		var localFixes []string

		// Section: xpo
		fmt.Printf("\nxpo\n")
		counts.pass("Version %s", version.CLIVersion)

		// Section: Project
		cwd, _ := os.Getwd()
		home, _ := os.UserHomeDir()
		displayPath := cwd
		if home != "" && strings.HasPrefix(cwd, home) {
			displayPath = "~" + cwd[len(home):]
		}
		fmt.Printf("\nProject  %s\n", displayPath)

		if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
			counts.err(".xpo directory not found")
			fmt.Printf("    Run xpo init to set up\n")
			printDoctorSummary(counts)
			if counts.fail > 0 {
				os.Exit(1)
			}
			return
		}
		counts.pass("Settings are valid (prefix %s)", prefix)

		if cfg != nil {
			if cfg.Version < version.DataModelVersion {
				counts.err("Data model v%d (expected v%d)", cfg.Version, version.DataModelVersion)
				fmt.Printf("    Run xpo migrate to update\n")
			} else if cfg.Version > version.DataModelVersion {
				counts.err("Data model v%d (CLI expects v%d)", cfg.Version, version.DataModelVersion)
				fmt.Printf("    Upgrade xpo to match\n")
			}
		}

		if !exponential.CheckGitRepo() {
			counts.note("Not a git repository")
			if doctorFix && isInteractive() {
				if promptConfirm("Initialize git repository?", false) {
					gitCmd := exec.Command("git", "init")
					if out, err := gitCmd.CombinedOutput(); err != nil {
						counts.err("git init failed: %v\n%s", err, string(out))
					} else {
						counts.pass("Initialized git repository")
					}
				}
			}
		} else {
			counts.pass("Git repository")
		}

		// .gitignore
		requiredIgnores := []string{".xpo/git.lock", ".xpo/worktrees/"}
		gitignoreContent, _ := os.ReadFile(".gitignore")
		gitignoreStr := string(gitignoreContent)
		var missingIgnores []string
		for _, entry := range requiredIgnores {
			if !strings.Contains(gitignoreStr, entry) {
				missingIgnores = append(missingIgnores, entry)
			}
		}
		if len(missingIgnores) > 0 {
			if doctorFix {
				f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					counts.err(".gitignore: could not add missing entries: %v", err)
				} else {
					if len(gitignoreStr) > 0 && !strings.HasSuffix(gitignoreStr, "\n") {
						f.WriteString("\n")
					}
					for _, entry := range missingIgnores {
						f.WriteString(entry + "\n")
					}
					f.Close()
					localFixes = append(localFixes, "Added missing .gitignore entries")
				}
			} else {
				counts.note(".gitignore missing entries")
				fmt.Printf("    Run xpo doctor --fix to add them\n")
			}
		} else {
			counts.pass(".gitignore")
		}

		if !storage.RefStoreReady() {
			counts.err("refs/xpo/data not found — run xpo init")
		} else {
			counts.pass("Storage: refs/xpo/data")

			lineNum, err := storage.ValidateEvents()
			if err != nil {
				counts.err("Issues database has an invalid entry at line %d", lineNum)
				fmt.Printf("    %s\n", err)
				counts.attention = append(counts.attention, attentionItem{
					message: fmt.Sprintf("Issues database has an invalid entry at line %d\n    Use 'xpo db edit' to inspect and fix", lineNum),
					isError: true,
				})
			} else {
				counts.pass("Issues database (%d events)", lineNum)
			}
		}

		// Section: Integrations
		fmt.Printf("\nIntegrations\n")
		doctorIntegrations(counts, prefix, integrationVer)

		// --fix mode: show fix summary and skip "This machine"
		if doctorFix {
			if len(localFixes) > 0 {
				fmt.Printf("\nFixed locally\n")
				for _, fix := range localFixes {
					fmt.Printf("  %s %s\n", ui.OKPrefix, fix)
				}
			}

			printDoctorFixSummary(counts, localFixes)

			if counts.fail > 0 || len(counts.attention) > 0 {
				os.Exit(1)
			}
			return
		}

		// Section: This machine (diagnostic mode only)
		fmt.Printf("\nThis machine\n")
		agents := exponential.DetectInstalledAgents()
		for _, agent := range agents {
			if agent.Binary != "" {
				counts.pass("%s CLI found", agent.Name)
			}
		}
		doctorShellCompletion(counts)

		printDoctorSummary(counts)

		if counts.fail > 0 {
			os.Exit(1)
		}
		if doctorStrict && counts.warn > 0 {
			os.Exit(1)
		}
	},
}

// checkAgentHealth delegates to the testable internal function.
func checkAgentHealth(agent exponential.AgentConfig, integrationVer string) exponential.AgentHealth {
	return exponential.CheckAgentHealth(agent, integrationVer)
}

func doctorIntegrations(c *doctorCounts, prefix string, integrationVer string) {
	agents := exponential.DetectInstalledAgents()

	maxName := 0
	for _, agent := range agents {
		if len(agent.Name) > maxName {
			maxName = len(agent.Name)
		}
	}

	var autoFixAgents []exponential.AgentConfig
	var interactiveAgents []exponential.AgentConfig

	for _, agent := range agents {
		h := checkAgentHealth(agent, integrationVer)
		padded := fmt.Sprintf("%-*s", maxName, agent.Name)

		if !h.Configured {
			c.hint("%s   Found, not set up. Add it with xpo init", padded)
			continue
		}

		if h.HasProblems() {
			c.note("%s   %s", padded, strings.Join(h.AllProblems(), ", "))
			if len(h.AutoFix) > 0 {
				autoFixAgents = append(autoFixAgents, agent)
			}
			if len(h.Interactive) > 0 {
				interactiveAgents = append(interactiveAgents, agent)
			}
		} else {
			var status []string
			if h.HasMCP {
				status = append(status, "MCP configured")
			}
			status = append(status, "Up to date")
			c.pass("%s   %s", padded, strings.Join(status, ", "))
		}
	}

	// Also show agents on PATH but not detected by DetectInstalledAgents
	allAgents := exponential.AgentRegistry
	detectedSet := map[string]bool{}
	for _, d := range agents {
		detectedSet[d.Name] = true
	}
	for _, agent := range allAgents {
		if agent.Name == "Generic Agent" || agent.Binary == "" || detectedSet[agent.Name] {
			continue
		}
		if _, err := exec.LookPath(agent.Binary); err == nil {
			c.hint("%-*s   Found, not set up. Add it with xpo init", maxName, agent.Name)
		}
	}

	if !doctorFix {
		return
	}

	// Auto-fixable agents: show change plan and apply
	if len(autoFixAgents) > 0 {
		plan := exponential.ComputeInitPlan(autoFixAgents)
		if len(plan.Changes) > 0 {
			fmt.Printf("\nChanges\n")
			for _, ch := range plan.Changes {
				marker := "~"
				if ch.Action == "create" {
					marker = "+"
				}
				fmt.Printf("  %s %s\n", marker, ch.Path)
			}

			if isInteractive() {
				if !promptConfirm("Apply changes?", true) {
					fmt.Printf("\nNo changes were made. Run %s again when you're ready.\n\n", ui.AccentStyle.Render("xpo doctor --fix"))
					os.Exit(1)
				}
			}
			applyInit(prefix, autoFixAgents)
			c.fixed += len(autoFixAgents)
		}
	}

	// Interactive agents: offer replace/keep/diff per file
	for _, agent := range interactiveAgents {
		promptEditedBlock(agent, prefix, true)
		c.fixed++
	}
}

func doctorShellCompletion(c *doctorCounts) {
	compRes := exponential.CheckCompletionConfig()
	if compRes.Shell == "unknown" {
		return
	}

	if compRes.Configured {
		c.pass("Shell completions installed (%s)", compRes.Shell)
	} else {
		c.note("Shell completion for %s not configured", compRes.Shell)
		cmd := exponential.GetCompletionInstallCmd(compRes.Shell)
		fmt.Printf("    %s\n", cmd)
	}
}

func renderAttention(items []attentionItem) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("\nCannot fix automatically\n")
	for _, item := range items {
		prefix := ui.NotePrefix
		if item.isError {
			prefix = ui.ErrorPrefix
		}
		fmt.Printf("  %s %s\n", prefix, item.message)
	}
}

func printDoctorFixSummary(counts *doctorCounts, localFixes []string) {
	renderAttention(counts.attention)

	totalFixed := counts.fixed + len(localFixes)
	totalProblems := totalFixed + len(counts.attention)
	fmt.Println()
	if totalProblems == 0 {
		fmt.Println("Nothing to fix.")
	} else if len(counts.attention) == 0 {
		fmt.Printf("%s Fixed %d of %d problems\n", ui.OKPrefix, totalFixed, totalProblems)
	} else {
		fmt.Printf("Fixed %d of %d problems\n", totalFixed, totalProblems)
	}
}

func printDoctorSummary(counts *doctorCounts) {
	renderAttention(counts.attention)

	fmt.Println()
	problems := counts.fail + counts.warn
	fixable := problems - len(counts.attention)
	if counts.fail > 0 {
		fmt.Printf("%d passed, %d warnings, %d errors\n", counts.ok, counts.warn, counts.fail)
	} else if counts.warn > 0 {
		fmt.Printf("%d passed, %d warnings\n", counts.ok, counts.warn)
	} else {
		fmt.Printf("All %d checks passed.\n", counts.ok)
	}
	if fixable > 0 && !doctorFix {
		fmt.Printf("Run %s to resolve %d of them.\n", ui.AccentStyle.Render("xpo doctor --fix"), fixable)
	}
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Auto-fix problems where possible")
	doctorCmd.Flags().BoolVar(&doctorStrict, "strict", false, "Exit non-zero on warnings (for CI)")
	rootCmd.AddCommand(doctorCmd)
}
