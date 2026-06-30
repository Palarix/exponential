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
	Long:  "Check for common issues and optionally fix them, including setting up AI agent instruction files.",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if .xpo directory exists
		if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
			fmt.Println("Error: .xpo directory not found. Run 'xpo init' first.")
			os.Exit(1)
		}

		// Get prefix from config
		prefix := "issue-"
		if cfg != nil && cfg.Prefix != "" {
			prefix = cfg.Prefix
		}

		fmt.Println("xpo doctor - Checking configuration...")
		fmt.Println()

		// Check Data Model Version
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
		fmt.Println()

		// Check for git repo
		if !exponential.CheckGitRepo() { // Using internal logic
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Not a git repository\n", ui.NotePrefix)))
			fmt.Print(ui.Stylize("    You risk losing your issues database if you delete this directory.\n"))

			if isInteractive() {
				fmt.Print(ui.Stylize("\nInitialize git repository? [y/N]: "))
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" || response == "yes" {
					cmd := exec.Command("git", "init")
					if out, err := cmd.CombinedOutput(); err != nil {
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
		fmt.Println()

		// Check for issues.db
		issuesDB := filepath.Join(".xpo", "issues.db")
		if _, err := os.Stat(issuesDB); os.IsNotExist(err) {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Exponential database not found\n", ui.ErrorPrefix)))
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Exponential database exists\n", ui.OKPrefix)))
		}

		// Check .gitignore for required xpo entries
		requiredIgnores := []string{".xpo/issues.snapshot.json", ".xpo/git.lock"}
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

		// Check .mcp.json
		mcpStatus := exponential.DetectMCPConfig()
		if mcpStatus.HasExponential {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s MCP config (`.mcp.json`) has xpo entry\n", ui.OKPrefix)))
		} else {
			if mcpStatus.Exists {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s `.mcp.json` exists but has no xpo entry\n", ui.NotePrefix)))
			} else {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s `.mcp.json` not found\n", ui.NotePrefix)))
			}

			if isInteractive() {
				prompt := "Add xpo MCP server entry to `.mcp.json`?"
				if !mcpStatus.Exists {
					prompt = "Create `.mcp.json` with xpo MCP server entry?"
				}
				fmt.Print(ui.Stylize(fmt.Sprintf("\n%s [y/N]: ", prompt)))
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" || response == "yes" {
					if err := exponential.EnsureMCPConfig(); err != nil {
						fmt.Print(ui.Stylize(fmt.Sprintf("%s %v\n", ui.ErrorPrefix, err)))
					} else {
						action := "Added xpo entry to"
						if !mcpStatus.Exists {
							action = "Created"
						}
						fmt.Print(ui.Stylize(fmt.Sprintf("%s %s `.mcp.json`\n", ui.OKPrefix, action)))
					}
				} else {
					fmt.Println("Skipped.")
				}
			} else {
				fmt.Print(ui.Stylize("    Run `xpo doctor` in an interactive terminal to configure.\n"))
			}
		}

		// Detect AI agent files
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

		// Report configured agents
		if len(configured) > 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Agent files with xpo config:\n", ui.OKPrefix)))
			for _, r := range configured {
				fmt.Print(ui.Stylize(fmt.Sprintf("    • %s (`%s`)\n", r.Agent.Name, r.Agent.File)))
			}
		}

		// Report agents needing config
		if len(needsConfig) > 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Agent files needing xpo config:\n", ui.NotePrefix)))
			for _, r := range needsConfig {
				fmt.Print(ui.Stylize(fmt.Sprintf("    • %s (`%s`)\n", r.Agent.Name, r.Agent.File)))
			}

			// Prompt to fix if interactive
			if isInteractive() {
				fmt.Print(ui.Stylize("\nWould you like to add xpo instructions to these files? [y/N]: "))
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" || response == "yes" {
					for _, r := range needsConfig {
						if err := exponential.AppendAgentInstructions(r.Agent, prefix); err != nil {
							fmt.Print(ui.Stylize(fmt.Sprintf(" %s `%s`: %v\n", ui.ErrorPrefix, r.Agent.File, err)))
						} else {
							fmt.Print(ui.Stylize(fmt.Sprintf(" %s Added xpo instructions to `%s`\n", ui.OKPrefix, r.Agent.File)))
						}
					}
				} else {
					fmt.Println("Skipped.")
				}
			} else {
				fmt.Print(ui.Stylize("\nRun `xpo doctor` in an interactive terminal to add xpo instructions.\n"))
			}
		} else if len(configured) == 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s No agent instruction files detected\n", ui.NotePrefix)))
			fmt.Print(ui.Stylize("    Agent instruction files help AI assistants understand your xpo issues and workflow.\n"))

			if isInteractive() {
				fmt.Print(ui.Stylize("\nCreate `AGENTS.md` with xpo instructions? [y/N]: "))
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" || response == "yes" {
					agent := exponential.AgentConfig{Name: "Generic Agent", File: "AGENTS.md", Format: "markdown"}
					if err := exponential.AppendAgentInstructions(agent, prefix); err != nil {
						fmt.Print(ui.Stylize(fmt.Sprintf("%s Error creating `AGENTS.md`: %v\n", ui.ErrorPrefix, err)))
					} else {
						fmt.Print(ui.Stylize(fmt.Sprintf("%s Created `AGENTS.md` with instructions\n", ui.OKPrefix)))
					}
				} else {
					fmt.Println("Skipped.")
				}
			} else {
				fmt.Print(ui.Stylize("    Run `xpo doctor` to create a default configuration.\n"))
			}
		}

		// Check shell completion
		compRes := exponential.CheckCompletionConfig()
		if compRes.Shell != "unknown" {
			if compRes.Configured {
				fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Shell completion for `%s` is configured\n", ui.OKPrefix, compRes.Shell)))
			} else {
				fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Shell completion for `%s` is not configured\n", ui.NotePrefix, compRes.Shell)))
				cmd := exponential.GetCompletionInstallCmd(compRes.Shell)
				fmt.Println("    To enable completion, run this command and reload your shell:")
				fmt.Print(ui.Stylize(fmt.Sprintf("    `%s`\n", cmd)))
			}
		}

		fmt.Println("\nCheck complete!")
	},
}

// isInteractive checks if stdin is a terminal
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
