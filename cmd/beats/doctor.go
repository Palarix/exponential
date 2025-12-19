package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/kuyio/beats/cmd/beats/ui"
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

		// Define styled prefixes (width 7 chars for alignment)
		// [OK]    = 7 chars
		// [ERROR] = 7 chars
		// [NOTE]  = 6 chars -> needs 1 space padding
		okPrefix := ui.GreenStyle.Render("[✔]")
		errPrefix := ui.RedStyle.Render("[x]")
		notePrefix := ui.YellowStyle.Render("[!]")

		fmt.Println("beats doctor - Checking configuration...\n")

		// Check for issues.db
		issuesDB := filepath.Join(".beats", "issues.db")
		if _, err := os.Stat(issuesDB); os.IsNotExist(err) {
			fmt.Printf("%s Beats database not found\n", errPrefix)
		} else {
			fmt.Printf("%s Beats database exists\n", okPrefix)
		}

		// Detect AI agent files
		results := DetectAgentFiles()
		var configured, needsConfig []AgentDetectionResult

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
			fmt.Printf("\n%s Agent files with beats config:\n\n", okPrefix)
			for _, r := range configured {
				fmt.Printf("    - %s (%s)\n", r.Agent.Name, r.Agent.File)
			}
		}

		// Report agents needing config
		if len(needsConfig) > 0 {
			fmt.Printf("\n%s Agent files needing beats config:\n\n", notePrefix)
			for _, r := range needsConfig {
				fmt.Printf("    - %s (%s)\n", r.Agent.Name, r.Agent.File)
			}

			// Prompt to fix if interactive
			if isInteractive() {
				fmt.Print("\nWould you like to add beats instructions to these files? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" || response == "yes" {
					for _, r := range needsConfig {
						if err := AppendBeatsToAgentFile(r.Agent, prefix); err != nil {
							fmt.Printf(" %s %s: %v\n", errPrefix, r.Agent.File, err)
						} else {
							fmt.Printf(" %s Added beats instructions to %s\n", okPrefix, r.Agent.File)
						}
					}
				} else {
					fmt.Println("Skipped.")
				}
			} else {
				fmt.Println("\nRun 'beats doctor' in an interactive terminal to add beats instructions.")
			}
		} else if len(configured) == 0 {
			fmt.Printf("\n%s No agent instruction files detected\n", notePrefix)
		}

		// Check shell completion
		compRes := CheckCompletionConfig()
		if compRes.Shell != "unknown" {
			if compRes.Configured {
				fmt.Printf("\n%s Shell completion for %s is configured\n", okPrefix, compRes.Shell)
			} else {
				fmt.Printf("\n%s Shell completion for %s is not configured\n\n", notePrefix, compRes.Shell)
				cmd := GetCompletionInstallCmd(compRes.Shell)
				fmt.Println("    To enable completion, run the following command and reload your shell:")
				fmt.Printf("\n    %s\n", cmd)
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
