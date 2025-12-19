package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		// 2. Check for .github/workflows/*.yml or *.yaml
		matches, err := filepath.Glob(".github/workflows/*.y*ml")
		if err != nil || len(matches) == 0 {
			fmt.Println("Error: No workflow files found in .github/workflows. Please setup GitHub Actions first.")
			os.Exit(1)
		}

		// 3. Create .beats directory
		beatsDir := ".beats"
		if err := os.MkdirAll(beatsDir, 0755); err != nil {
			fmt.Printf("Error creating .beats directory: %v\n", err)
			os.Exit(1)
		}

		// 4. Create .beats/issues.db
		issuesFile := filepath.Join(beatsDir, "issues.db")
		f, err := os.OpenFile(issuesFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error creating issues.db: %v\n", err)
			os.Exit(1)
		}
		f.Close()

		// 5. Add .beats/issues.snapshot.json to .gitignore
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

		fmt.Println("Initialized .beats successfully!")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
