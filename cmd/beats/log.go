package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log [id] [points]",
	Short: "Log work burned on an issue",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		pointsStr := args[1]

		points, err := strconv.Atoi(pointsStr)
		if err != nil {
			fmt.Printf("Error: points must be an integer: %v\n", err)
			os.Exit(1)
		}

		user := getUser()

		payload := model.WorkLogPayload{
			Amount: points,
		}

		event := model.Event{
			ID:        id,
			Type:      model.EventTypeWorkLog,
			Payload:   payload,
			CreatedAt: time.Now().UTC(),
			CreatedBy: user,
		}

		if err := storage.AppendEvent(event); err != nil {
			fmt.Printf("Error appending event: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Logged %d SP on %s\n", points, id)

		if cfg.AutoCommit {
			commitMsg := fmt.Sprintf("beats: log %s %d", id, points)
			fmt.Println("Auto-committing...")
			if err := exec.Command("git", "add", ".beats/issues.db").Run(); err != nil {
				fmt.Printf("Error adding to git: %v\n", err)
			} else if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
				fmt.Printf("Error committing: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
