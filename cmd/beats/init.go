package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the .beats directory",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Check if inside git repo
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Println("Error: Not inside a git repository")
			os.Exit(1)
		}

		// 2. Check for .github/workflows/*.yml or *.yaml
		matches, err := filepath.Glob(".github/workflows/*.y*ml")
		if err != nil || len(matches) == 0 {
			fmt.Println("Error: No workflow files found in .github/workflows. Please setup GitHub Actions first.")
			os.Exit(1)
		}

		// 3. Check if .beats already exists
		beatsDir := ".beats"
		if _, err := os.Stat(beatsDir); err == nil {
			if !initForce {
				fmt.Println("Error: .beats directory already exists. Use --force to re-initialize.")
				os.Exit(1)
			}
			fmt.Println("Warning: Re-initializing existing .beats directory...")
		}

		// 4. Check for existing git hooks (non-sample files)
		hookFiles, _ := filepath.Glob(".git/hooks/*")
		var activeHooks []string
		for _, h := range hookFiles {
			if !strings.HasSuffix(h, ".sample") {
				activeHooks = append(activeHooks, filepath.Base(h))
			}
		}
		if len(activeHooks) > 0 {
			fmt.Printf("Note: Found existing git hooks: %s\n", strings.Join(activeHooks, ", "))
			fmt.Println("      beats will not modify your existing hooks.")
		}

		// 5. Create .beats directory
		if err := os.MkdirAll(beatsDir, 0755); err != nil {
			fmt.Printf("Error creating .beats directory: %v\n", err)
			os.Exit(1)
		}

		// 6. Create .beats/issues.db
		issuesFile := filepath.Join(beatsDir, "issues.db")
		f, err := os.OpenFile(issuesFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error creating issues.db: %v\n", err)
			os.Exit(1)
		}
		f.Close()

		// 7. Add .beats/issues.snapshot.json to .gitignore
		gitignorePath := ".gitignore"
		content, err := os.ReadFile(gitignorePath)
		var contentStr string
		if err == nil {
			contentStr = string(content)
		}

		ignoreEntry := ".beats/issues.snapshot.json"
		if !strings.Contains(contentStr, ignoreEntry) {
			f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				defer f.Close()
				if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
					f.WriteString("\n")
				}
				f.WriteString(ignoreEntry + "\n")
			} else {
				fmt.Printf("Warning: Could not write to .gitignore: %v\n", err)
			}
		}

		// 8. Derive prefix from folder name and write config.yaml
		cwd, _ := os.Getwd()
		folderName := filepath.Base(cwd)
		prefix := sanitizePrefix(folderName) + "-"

		configPath := filepath.Join(beatsDir, "config.yaml")
		configContent := fmt.Sprintf("prefix: %s\n", prefix)
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			fmt.Printf("Warning: Could not write config.yaml: %v\n", err)
		} else {
			fmt.Printf("Configured issue prefix: %s\n", prefix)
		}

		// 9. Detect AI agent instruction files
		results := DetectAgentFiles()
		var detected, needsConfig []string
		for _, r := range results {
			if r.Exists {
				if r.HasBeatsConfig {
					detected = append(detected, fmt.Sprintf("%s (%s) ✓", r.Agent.Name, r.Agent.File))
				} else {
					needsConfig = append(needsConfig, fmt.Sprintf("%s (%s)", r.Agent.Name, r.Agent.File))
				}
			}
		}

		if len(detected) > 0 {
			fmt.Println("\nDetected agent files with beats config:")
			for _, d := range detected {
				fmt.Printf("  • %s\n", d)
			}
		}

		if len(needsConfig) > 0 {
			fmt.Println("\nAgent files needing beats config:")
			for _, n := range needsConfig {
				fmt.Printf("  ○ %s\n", n)
			}
			fmt.Println("\nRun 'beats doctor' to add beats instructions to these files.")
		}

		// 10. Check shell completion
		compRes := CheckCompletionConfig()
		if !compRes.Configured && compRes.Shell != "unknown" {
			fmt.Printf("\n[NOTE] Shell completion for %s is not configured.\n", compRes.Shell)
			fmt.Println("Run 'beats doctor' to get setup instructions.")
		}

		fmt.Println("\nInitialized .beats successfully!")
	},
}

// sanitizePrefix converts a folder name to a valid issue ID prefix
func sanitizePrefix(name string) string {
	name = strings.ToLower(name)
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	s := result.String()
	if s == "" {
		s = "beats" // Fallback if folder name has no valid chars
	}
	return s
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Re-initialize even if .beats already exists")
	rootCmd.AddCommand(initCmd)
}
