package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/spf13/cobra"
)

var (
	inboxClear    bool
	inboxJSONFlag bool
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Show events relevant to you since your last check",
	Long: `Show a personal feed of recent events relevant to you — assignments,
comments on your issues, status changes, and merges — since you last
checked.

An event is relevant if it happened on an issue you created, are assigned
to, or have commented on. All merges are shown for team-wide visibility.
Your own actions are not shown.

Read state is tracked per-user in ~/.config/xpo/user.yaml. Run
` + "`xpo inbox --clear`" + ` to mark everything as read.

Examples:
  xpo inbox
  xpo inbox --clear
  xpo inbox --json`,
	RunE: runInbox,
}

func init() {
	inboxCmd.Flags().BoolVar(&inboxClear, "clear", false, "Mark all inbox items as read")
	inboxCmd.Flags().BoolVar(&inboxJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(inboxCmd)
}

func runInbox(cmd *cobra.Command, args []string) error {
	remoteURL := ""
	if cfg != nil {
		remoteURL = cfg.Remote.URL
	}

	if inboxClear {
		if err := config.SetInboxLastRead(remoteURL, time.Now()); err != nil {
			if inboxJSONFlag {
				exitJSONError(fmt.Errorf("failed to update read cursor: %w", err))
			}
			return fmt.Errorf("failed to update read cursor: %w", err)
		}
		fmt.Println("Inbox marked as read.")
		return nil
	}

	client := exponential.NewClient(cfg)
	me := client.GetUser()
	since := config.GetInboxLastRead(remoteURL)

	items, err := client.GetInbox(since)
	if err != nil {
		if inboxJSONFlag {
			exitJSONError(err)
		}
		return err
	}

	if inboxJSONFlag {
		entries := make([]jsonio.InboxEntry, len(items))
		for i, item := range items {
			entries[i] = jsonio.InboxEntry{
				IssueID:    item.IssueID,
				IssueTitle: item.IssueTitle,
				Type:       string(item.Type),
				Payload:    item.Payload,
				CreatedAt:  item.CreatedAt.Format(time.RFC3339),
				CreatedBy:  item.CreatedBy,
				OnBehalfOf: item.OnBehalfOf,
			}
		}
		out := jsonio.InboxOutput{Items: entries}
		if out.Items == nil {
			out.Items = []jsonio.InboxEntry{}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return nil
	}

	if len(items) == 0 {
		fmt.Println("No new activity.")
		return nil
	}

	for _, item := range items {
		ago := humanize.Time(item.CreatedAt)
		fmt.Printf("  %-18s %s\n", ago, exponential.FormatInboxItem(item, me))
	}

	noun := "items"
	if len(items) == 1 {
		noun = "item"
	}
	fmt.Printf("\n%d new %s since last check. Run `xpo inbox --clear` to mark as read.\n", len(items), noun)
	return nil
}
