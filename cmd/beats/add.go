package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	addEpicFlag   bool
	addBugFlag    bool
	addParentFlag string
	addDescFlag   string
	addSPFlag     int
	addForceFlag  bool
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Create a new issue",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)
		var title string
		var description string

		// 1. Interactive Mode (No Title Provided)
		if len(args) == 0 {
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Fprintln(os.Stderr, "Error: title required in non-interactive mode")
				cmd.Help()
				os.Exit(1)
			}

			// Edit in temp file using UI helper
			template := "<Title>\n\n<Description>\n"
			content, err := ui.EditInteractive(template)
			if err != nil {
				fmt.Printf("Error editing: %v\n", err)
				os.Exit(1)
			}

			if content == "" {
				fmt.Println("Aborted by user (empty content).")
				os.Exit(0)
			}

			title, description = parseInteractiveContent(content)
			if title == "" {
				fmt.Println("Aborted: empty title")
				os.Exit(0)
			}
		} else {
			// 2. Argument Mode
			title = args[0]
			description = addDescFlag
		}
		kind := "TASK"

		if addEpicFlag {
			kind = "EPIC"
		} else if addBugFlag {
			kind = "BUG"
		}

		// Duplicate detection
		if !addForceFlag {
			duplicates, err := client.CheckDuplicates(title)
			if err != nil {
				// Warn but don't fail
				fmt.Fprintf(os.Stderr, "Warning: could not read events for duplicate check: %v\n", err)
			} else if len(duplicates) > 0 {
				fmt.Println("Potential duplicate(s) found:")
				for _, d := range duplicates {
					fmt.Printf("  %s [%s] %s (Status: %s)\n", d.ID, d.Kind, d.Title, d.Status)
				}
				fmt.Println()

				if !isatty.IsTerminal(os.Stdout.Fd()) {
					fmt.Println("Error: potential duplicates found in non-interactive mode. Use --force to override.")
					os.Exit(1)
				}

				fmt.Print("Create anyway? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Aborted.")
					os.Exit(0)
				}
			}
		}

		// Create Issue
		opts := beats.AddOptions{
			Kind:        kind,
			Title:       title,
			Description: description,
			ParentID:    addParentFlag,
			Estimate:    addSPFlag,
		}

		issue, err := client.AddIssue(opts)
		if err != nil {
			fmt.Printf("Error creating issue: %v\n", err)
			os.Exit(1)
		}

		showConfirmation(issue.ID, issue.Kind, issue.Title)
	},
}

func showConfirmation(id string, kind string, title string) {
	fmt.Println(ui.Stylize("\n Created `" + id + "` [" + kind + "] - " + title))
}

func parseInteractiveContent(content string) (string, string) {
	// Normalize newlines
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSpace(content)

	// Split by first double newline
	parts := strings.SplitN(content, "\n\n", 2)

	title := strings.TrimSpace(parts[0])
	description := ""
	if len(parts) > 1 {
		description = strings.TrimSpace(parts[1])
	}

	return title, description
}

func init() {
	addCmd.Flags().BoolVarP(&addEpicFlag, "epic", "e", false, "Create an Epic")
	addCmd.Flags().BoolVarP(&addBugFlag, "bug", "b", false, "Create a Bug")
	addCmd.Flags().StringVarP(&addParentFlag, "parent", "p", "", "Parent ID")
	addCmd.Flags().StringVarP(&addDescFlag, "desc", "d", "", "Description")
	addCmd.Flags().IntVarP(&addSPFlag, "sp", "s", 0, "Story Points")
	addCmd.Flags().BoolVarP(&addForceFlag, "force", "f", false, "Force create even if duplicates found")
	rootCmd.AddCommand(addCmd)
}
