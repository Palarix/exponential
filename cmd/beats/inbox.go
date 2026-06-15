package main

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/config"
	"github.com/spf13/cobra"
)

var inboxClear bool

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Show events relevant to you since your last check",
	Long: `Show a personal feed of recent events relevant to you — assignments,
comments on your issues, status changes, and merges — since you last
checked.

An event is relevant if it happened on an issue you created, are assigned
to, or have commented on. All merges are shown for team-wide visibility.
Your own actions are not shown.

Read state is tracked per-user in ~/.config/beats/user.yaml. Run
` + "`beats inbox --clear`" + ` to mark everything as read.

Examples:
  beats inbox
  beats inbox --clear`,
	RunE: runInbox,
}

func init() {
	inboxCmd.Flags().BoolVar(&inboxClear, "clear", false, "Mark all inbox items as read")
	rootCmd.AddCommand(inboxCmd)
}

func runInbox(cmd *cobra.Command, args []string) error {
	remoteURL := ""
	if cfg != nil {
		remoteURL = cfg.Remote.URL
	}

	if inboxClear {
		if err := config.SetInboxLastRead(remoteURL, time.Now()); err != nil {
			return fmt.Errorf("failed to update read cursor: %w", err)
		}
		fmt.Println("Inbox marked as read.")
		return nil
	}

	client := beats.NewClient(cfg)
	me := client.GetUser()
	since := config.GetInboxLastRead(remoteURL)

	items, err := client.GetInbox(since)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Println("No new activity.")
		return nil
	}

	for _, item := range items {
		ago := humanize.Time(item.CreatedAt)
		fmt.Printf("  %-18s %s\n", ago, beats.FormatInboxItem(item, me))
	}

	noun := "items"
	if len(items) == 1 {
		noun = "item"
	}
	fmt.Printf("\n%d new %s since last check. Run `beats inbox --clear` to mark as read.\n", len(items), noun)
	return nil
}
