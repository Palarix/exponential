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

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/ui"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and fix beats configuration issues",
	Long:  "Check for common issues and optionally fix them, including setting up AI agent instruction files.",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if .beats directory exists
		if _, err := os.Stat(".beats"); os.IsNotExist(err) {
			fmt.Println("Error: .beats directory not found. Run 'beats init' first.")
			os.Exit(1)
		}

		// Get prefix from config
		prefix := "beats-"
		if cfg != nil && cfg.Prefix != "" {
			prefix = cfg.Prefix
		}

		fmt.Println("beats doctor - Checking configuration...")
		fmt.Println()

		// Check for git repo
		if !beats.CheckGitRepo() { // Using internal logic
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
		issuesDB := filepath.Join(".beats", "issues.db")
		if _, err := os.Stat(issuesDB); os.IsNotExist(err) {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Beats database not found\n", ui.ErrorPrefix)))
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Beats database exists\n", ui.OKPrefix)))
		}

		// Detect AI agent files
		results := beats.DetectAgentFiles()
		var configured, needsConfig []beats.AgentDetectionResult

		for _, r := range results {
			if r.Exists {
				if r.HasBeatsConfig {
					configured = append(configured, r)
				} else {
					needsConfig = append(needsConfig, r)
				}
			}
		}

		// Report configured agents
		if len(configured) > 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Agent files with beats config:\n", ui.OKPrefix)))
			for _, r := range configured {
				fmt.Print(ui.Stylize(fmt.Sprintf("    • %s (`%s`)\n", r.Agent.Name, r.Agent.File)))
			}
		}

		// Report agents needing config
		if len(needsConfig) > 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Agent files needing beats config:\n", ui.NotePrefix)))
			for _, r := range needsConfig {
				fmt.Print(ui.Stylize(fmt.Sprintf("    • %s (`%s`)\n", r.Agent.Name, r.Agent.File)))
			}

			// Prompt to fix if interactive
			if isInteractive() {
				fmt.Print(ui.Stylize("\nWould you like to add beats instructions to these files? [y/N]: "))
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" || response == "yes" {
					for _, r := range needsConfig {
						if err := beats.AppendBeatsToAgentFile(r.Agent, prefix); err != nil {
							fmt.Print(ui.Stylize(fmt.Sprintf(" %s `%s`: %v\n", ui.ErrorPrefix, r.Agent.File, err)))
						} else {
							fmt.Print(ui.Stylize(fmt.Sprintf(" %s Added beats instructions to `%s`\n", ui.OKPrefix, r.Agent.File)))
						}
					}
				} else {
					fmt.Println("Skipped.")
				}
			} else {
				fmt.Print(ui.Stylize("\nRun `beats doctor` in an interactive terminal to add beats instructions.\n"))
			}
		} else if len(configured) == 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s No agent instruction files detected\n", ui.NotePrefix)))
			fmt.Print(ui.Stylize("    Agent instruction files help AI assistants understand your beats issues and workflow.\n"))

			if isInteractive() {
				fmt.Print(ui.Stylize("\nCreate `AGENTS.md` with beats instructions? [y/N]: "))
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" || response == "yes" {
					agent := beats.AgentConfig{Name: "Generic Agent", File: "AGENTS.md", Format: "markdown"}
					if err := beats.AppendBeatsToAgentFile(agent, prefix); err != nil {
						fmt.Print(ui.Stylize(fmt.Sprintf("%s Error creating `AGENTS.md`: %v\n", ui.ErrorPrefix, err)))
					} else {
						fmt.Print(ui.Stylize(fmt.Sprintf("%s Created `AGENTS.md` with instructions\n", ui.OKPrefix)))
					}
				} else {
					fmt.Println("Skipped.")
				}
			} else {
				fmt.Print(ui.Stylize("    Run `beats doctor` to create a default configuration.\n"))
			}
		}

		// Check shell completion
		compRes := beats.CheckCompletionConfig()
		if compRes.Shell != "unknown" {
			if compRes.Configured {
				fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Shell completion for `%s` is configured\n", ui.OKPrefix, compRes.Shell)))
			} else {
				fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Shell completion for `%s` is not configured\n", ui.NotePrefix, compRes.Shell)))
				cmd := beats.GetCompletionInstallCmd(compRes.Shell)
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
