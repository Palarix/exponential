package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	gonanoid "github.com/matoous/go-nanoid/v2"
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
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := args[0]
		kind := "TASK"

		if addEpicFlag {
			kind = "EPIC"
		} else if addBugFlag {
			kind = "BUG"
		}

		// Duplicate detection
		if !addForceFlag {
			events, err := storage.ReadEvents()
			if err != nil {
				// Warn but don't fail, this check is optional
				fmt.Fprintf(os.Stderr, "Warning: could not read events for duplicate check: %v\n", err)
			} else {
				issuesMap := model.ProjectIssues(events)
				var issues []*model.Issue
				for _, i := range issuesMap {
					issues = append(issues, i)
				}
				// Sort by creation time desc (newest first)
				sort.Slice(issues, func(i, j int) bool {
					return issues[i].CreatedAt.After(issues[j].CreatedAt)
				})

				titleTokens := tokenize(title)
				var duplicates []*model.Issue

				for _, issue := range issues {
					issueTokens := tokenize(issue.Title)

					intersection := 0
					for t := range titleTokens {
						if issueTokens[t] {
							intersection++
						}
					}

					// Criteria:
					// 1. Strict Subset: intersection == len(titleTokens) (all new words exist in old)
					// 2. High Overlap: intersection >= 75% of min length

					minLen := len(titleTokens)
					if len(issueTokens) < minLen {
						minLen = len(issueTokens)
					}

					if minLen > 0 {
						ratio := float64(intersection) / float64(minLen)
						if ratio >= 0.75 {
							duplicates = append(duplicates, issue)
						}
					}
				}

				if len(duplicates) > 0 {
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
		}

		// Generate ID
		alphabet := "0123456789abcdef"
		id, err := gonanoid.Generate(alphabet, 6)
		if err != nil {
			fmt.Printf("Error generating ID: %v\n", err)
			os.Exit(1)
		}
		id = "beat-" + id

		// Get User
		user := getUser()

		payload := model.CreatePayload{
			Kind:        kind,
			Title:       title,
			Description: addDescFlag,
			ParentID:    addParentFlag,
			Estimate:    addSPFlag,
		}

		event := model.Event{
			ID:        id,
			Type:      model.EventTypeCreate,
			Payload:   payload,
			CreatedAt: time.Now().UTC(),
			CreatedBy: user,
		}

		if err := storage.AppendEvent(event); err != nil {
			fmt.Printf("Error appending event: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created %s: %s (%s)\n", kind, id, title)

		if cfg.AutoCommit {
			commitMsg := fmt.Sprintf("beats: create %s %s - %s", kind, id, title)
			fmt.Println("Auto-committing...")
			if err := exec.Command("git", "add", ".beats/issues.jsonl").Run(); err != nil {
				fmt.Printf("Error adding to git: %v\n", err)
			} else if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
				fmt.Printf("Error committing: %v\n", err)
			}
		}
	},
}

func getUser() string {
	// Try to get from git config
	nameBytes, _ := exec.Command("git", "config", "user.name").Output()
	emailBytes, _ := exec.Command("git", "config", "user.email").Output()

	name := strings.TrimSpace(string(nameBytes))
	email := strings.TrimSpace(string(emailBytes))

	if name == "" {
		name = "Unknown"
	}
	if email == "" {
		email = "unknown@example.com"
	}

	return fmt.Sprintf("%s <%s>", name, email)
}

func tokenize(s string) map[string]bool {
	tokens := make(map[string]bool)
	fields := strings.Fields(strings.ToLower(s))
	for _, f := range fields {
		f = strings.Trim(f, "(),.:;!?")
		if len(f) > 0 {
			tokens[f] = true
		}
	}
	return tokens
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
