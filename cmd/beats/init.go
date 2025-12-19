package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/palarix/beats/cmd/beats/ui"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the .beats directory",
	Run: func(cmd *cobra.Command, args []string) {
		var notes []string

		// --- PHASE 1: Initialization & Informational Output ---

		// 1. Create .beats directory
		beatsDir := ".beats"
		if _, err := os.Stat(beatsDir); err == nil {
			if !initForce {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s `.beats` directory already exists. Use `--force` to re-initialize.\n", ui.ErrorPrefix)))
				os.Exit(1)
			}
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Re-initializing existing `.beats` directory...\n", ui.NotePrefix)))
		} else {
			if err := os.MkdirAll(beatsDir, 0755); err != nil {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Error creating `.beats` directory: %v\n", ui.ErrorPrefix, err)))
				os.Exit(1)
			}
		}

		// 2. Derive prefix from folder name and write config.yaml
		cwd, _ := os.Getwd()
		folderName := filepath.Base(cwd)
		prefix := sanitizePrefix(folderName) + "-"

		configPath := filepath.Join(beatsDir, "config.yaml")
		configContent := fmt.Sprintf("prefix: %s\n", prefix)
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			notes = append(notes, ui.Stylize(fmt.Sprintf("Could not write `config.yaml`: %v", err)))
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Configured issue prefix: `%s`\n", ui.OKPrefix, prefix)))
		}

		// 3. Create .beats/issues.db
		issuesFile := filepath.Join(beatsDir, "issues.db")
		f, err := os.OpenFile(issuesFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Error creating `issues.db`: %v\n", ui.ErrorPrefix, err)))
			os.Exit(1)
		}
		f.Close()
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Created `issues.db`\n", ui.OKPrefix)))

		// 4. Add .beats/issues.snapshot.json to .gitignore
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
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Added local beats artifacts to `.gitignore`\n", ui.OKPrefix)))
			} else {
				notes = append(notes, ui.Stylize(fmt.Sprintf("Could not write to `.gitignore`: %v", err)))
			}
		}

		// --- PHASE 2: Checks & Warnings ---

		// 5. Check if inside git repo
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			notes = append(notes, "Not a git repository")
		}

		// 6. Check for .github/workflows/*.yml or *.yaml
		matches, err := filepath.Glob(".github/workflows/*.y*ml")
		if err == nil && len(matches) > 0 {
			notes = append(notes, "Github workflows detected")
		}

		// 7. Check for existing git hooks (non-sample files)
		hookFiles, _ := filepath.Glob(".git/hooks/*")
		var activeHooks []string
		for _, h := range hookFiles {
			if !strings.HasSuffix(h, ".sample") {
				activeHooks = append(activeHooks, filepath.Base(h))
			}
		}
		if len(activeHooks) > 0 {
			notes = append(notes, "Existing git hooks found")
		}

		// 8. Detect AI agent instruction files
		results := DetectAgentFiles()
		var detected []string
		for _, r := range results {
			if r.Exists {
				if r.HasBeatsConfig {
					detected = append(detected, ui.Stylize(fmt.Sprintf("%s (`%s`)", r.Agent.Name, r.Agent.File)))
				} else {
					notes = append(notes, ui.Stylize(fmt.Sprintf("Agent file `%s` needs beats config", r.Agent.File)))
				}
			}
		}

		if len(detected) > 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Detected configured agents:\n", ui.OKPrefix)))
			for _, d := range detected {
				fmt.Printf("    - %s\n", d)
			}
		}

		// 9. Check shell completion
		compRes := CheckCompletionConfig()
		if !compRes.Configured && compRes.Shell != "unknown" {
			notes = append(notes, ui.Stylize(fmt.Sprintf("Shell completion for `%s` is not configured", compRes.Shell)))
		}

		if len(notes) > 0 {
			// fmt.Println("") // Spacing
			for _, note := range notes {
				fmt.Printf("%s %s\n", ui.NotePrefix, note)
			}
			fmt.Print(ui.Stylize("\nRun `beats doctor` to see details and fix these issues.\n"))
		}

		// --- PHASE 3: Summary and Warnings ---
		fmt.Print(ui.Stylize(fmt.Sprintf("\n%s Initialized `.beats` successfully!\n", ui.OKPrefix)))

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
