package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/glamour"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/storage"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var walkthroughJSONFlag bool

var walkthroughCmd = &cobra.Command{
	Use:   "walkthrough [issue-id]",
	Short: "Display or manage a walkthrough for an issue",
	Long: `Display the implementation walkthrough for an issue with terminal rendering.

  xpo walkthrough <id>              Glamour-rendered walkthrough
  xpo walkthrough <id> --json       JSON output (WalkthroughOutput)
  xpo walkthrough --json            Structured I/O from stdin (all operations)
  xpo walkthrough write <id>        Write walkthrough content from stdin
  xpo walkthrough delete <id>       Delete the walkthrough`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if walkthroughJSONFlag && len(args) == 0 {
			runWalkthroughJSON()
			return nil
		}
		if len(args) == 0 {
			return cmd.Help()
		}

		issueID := args[0]
		client := exponential.NewClient(cfg)
		content, err := client.ReadWalkthrough(issueID)
		if err != nil {
			if walkthroughJSONFlag {
				exitJSONError(err)
			}
			return err
		}

		if walkthroughJSONFlag {
			path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "walkthrough.md")
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(jsonio.WalkthroughOutput{OK: true, IssueID: issueID, Path: path, Content: content})
			return nil
		}

		termWidth := ui.TerminalWidth()
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(min(termWidth-4, 76)),
		)
		rendered, err := renderer.Render(content)
		if err != nil {
			fmt.Print(content)
			return nil
		}
		fmt.Print(rendered)
		return nil
	},
}

var walkthroughWriteCmd = &cobra.Command{
	Use:               "write <issue-id>",
	Short:             "Write walkthrough content from stdin",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		var content string
		var err error
		if isStdinPiped() {
			content, err = readAllStdin()
		} else {
			return fmt.Errorf("content is required: pipe to stdin")
		}
		if err != nil {
			return err
		}

		client := exponential.NewClient(cfg)
		if err := client.WriteWalkthrough(issueID, content); err != nil {
			return err
		}

		fmt.Printf("Walkthrough written for %s\n", issueID)
		return nil
	},
}

var walkthroughDeleteCmd = &cobra.Command{
	Use:               "delete <issue-id>",
	Short:             "Delete the walkthrough from an issue",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		client := exponential.NewClient(cfg)
		if err := client.DeleteWalkthrough(issueID); err != nil {
			return err
		}

		fmt.Printf("Walkthrough deleted from %s\n", issueID)
		return nil
	},
}

func runWalkthroughJSON() {
	content, err := readStdinExplicit()
	if err != nil {
		exitJSONError(err)
	}
	var input jsonio.WalkthroughToolInput
	if err := jsonio.DecodeStrict(content, &input); err != nil {
		exitJSONError(err)
	}
	if input.IssueID == "" {
		exitJSONError(fmt.Errorf("'issue_id' is required"))
	}

	client := exponential.NewClient(cfg)
	issue, err := client.GetIssue(input.IssueID)
	if err != nil {
		exitJSONError(err)
	}
	issueID := issue.ID
	path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "walkthrough.md")

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	switch input.Operation {
	case "write":
		if input.Content == "" {
			exitJSONError(fmt.Errorf("'content' is required for write"))
		}
		if err := client.WriteWalkthrough(issueID, input.Content); err != nil {
			exitJSONError(err)
		}
		enc.Encode(jsonio.WalkthroughOutput{OK: true, IssueID: issueID, Path: path})

	case "read":
		content, err := client.ReadWalkthrough(issueID)
		if err != nil {
			exitJSONError(err)
		}
		enc.Encode(jsonio.WalkthroughOutput{OK: true, IssueID: issueID, Path: path, Content: content})

	case "delete":
		if err := client.DeleteWalkthrough(issueID); err != nil {
			exitJSONError(err)
		}
		enc.Encode(jsonio.WalkthroughOutput{OK: true, IssueID: issueID, Path: path})

	default:
		exitJSONError(fmt.Errorf("invalid operation %q: must be write, read, or delete", input.Operation))
	}
}

func init() {
	walkthroughCmd.Flags().BoolVar(&walkthroughJSONFlag, "json", false, "Read a structured payload as JSON from stdin")

	walkthroughCmd.AddCommand(walkthroughWriteCmd)
	walkthroughCmd.AddCommand(walkthroughDeleteCmd)
	rootCmd.AddCommand(walkthroughCmd)
}
