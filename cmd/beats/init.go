package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the .beats directory",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Check if inside git repo
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Println("Error: Not inside a git repository")
			os.Exit(1)
		}

		// 2. Check for .github/workflows
		if _, err := os.Stat(".github/workflows"); os.IsNotExist(err) {
			fmt.Println("Error: .github/workflows directory not found. Please setup GitHub Actions first.")
			os.Exit(1)
		}

		// 3. Create .beats directory
		beatsDir := ".beats"
		if err := os.MkdirAll(beatsDir, 0755); err != nil {
			fmt.Printf("Error creating .beats directory: %v\n", err)
			os.Exit(1)
		}

		// 4. Create .beats/issues.jsonl
		issuesFile := filepath.Join(beatsDir, "issues.jsonl")
		f, err := os.OpenFile(issuesFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error creating issues.jsonl: %v\n", err)
			os.Exit(1)
		}
		f.Close()

		// 5. Add .beats/ to .gitignore
		gitignorePath := ".gitignore"
		f, err = os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Don't fail hard if gitignore fails, just warn
			fmt.Printf("Warning: Could not open .gitignore: %v\n", err)
		} else {
			defer f.Close()
			// Check if already ignored (simplistic check)
			// For now, just append
			if _, err := f.WriteString("\n# Beats issue tracker\n.beats/\n"); err != nil {
				fmt.Printf("Warning: Could not write to .gitignore: %v\n", err)
			}
		}

		fmt.Println("Initialized .beats successfully!")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
