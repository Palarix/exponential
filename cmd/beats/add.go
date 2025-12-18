package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/spf13/cobra"
)

var (
	addEpicFlag   bool
	addBugFlag    bool
	addParentFlag string
	addDescFlag   string
	addSPFlag     int
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

func init() {
	addCmd.Flags().BoolVarP(&addEpicFlag, "epic", "e", false, "Create an Epic")
	addCmd.Flags().BoolVarP(&addBugFlag, "bug", "b", false, "Create a Bug")
	addCmd.Flags().StringVarP(&addParentFlag, "parent", "p", "", "Parent ID")
	addCmd.Flags().StringVarP(&addDescFlag, "desc", "d", "", "Description")
	addCmd.Flags().IntVarP(&addSPFlag, "sp", "s", 0, "Story Points")
	rootCmd.AddCommand(addCmd)
}
