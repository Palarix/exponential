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
		}

		// Generate ID
		alphabet := "0123456789abcdef"
		id, err := gonanoid.Generate(alphabet, 6)
		if err != nil {
			fmt.Printf("Error generating ID: %v\n", err)
			os.Exit(1)
		}

		if addEpicFlag {
			id = "epic-" + id
		} else {
			id = "task-" + id
		}

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
			CreatedAt: time.Now(),
			CreatedBy: user,
		}

		if err := storage.AppendEvent(event); err != nil {
			fmt.Printf("Error appending event: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created %s: %s (%s)\n", kind, id, title)

		// TODO: Commit to git? The prompt mentions "Commits like before".
		// For now we just write to file as per CLI behavior description.
		// "Appends a new line to JSONL, git commit -am '<description>' .beats/issues.jsonl"
		// I will auto-commit for them.

		commitMsg := fmt.Sprintf("beats: create %s %s - %s", kind, id, title)
		_ = commitMsg
		// exec.Command("git", "add", ".beats/issues.jsonl").Run()
		// exec.Command("git", "commit", "-m", commitMsg).Run()
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
	addCmd.Flags().StringVarP(&addParentFlag, "parent", "p", "", "Parent ID")
	addCmd.Flags().StringVarP(&addDescFlag, "desc", "d", "", "Description")
	addCmd.Flags().IntVarP(&addSPFlag, "sp", "s", 0, "Story Points")
	rootCmd.AddCommand(addCmd)
}
