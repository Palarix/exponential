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

var specJSONFlag bool

var specCmd = &cobra.Command{
	Use:   "spec [issue-id]",
	Short: "Display or manage a spec for an issue",
	Long: `Display the design spec for an issue with terminal rendering.

  xpo spec <id>              Glamour-rendered spec
  xpo spec <id> --json       JSON output (SpecOutput)
  xpo spec --json            Structured I/O from stdin (all operations)
  xpo spec write <id>        Write spec content from stdin
  xpo spec delete <id>       Delete the spec`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if specJSONFlag && len(args) == 0 {
			runSpecJSON()
			return nil
		}
		if len(args) == 0 {
			return cmd.Help()
		}

		issueID := args[0]
		client := exponential.NewClient(cfg)
		content, err := client.ReadSpec(issueID)
		if err != nil {
			if specJSONFlag {
				exitJSONError(err)
			}
			return err
		}

		if specJSONFlag {
			path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "spec.md")
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(jsonio.SpecOutput{OK: true, IssueID: issueID, Path: path, Content: content})
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

var specWriteCmd = &cobra.Command{
	Use:               "write <issue-id>",
	Short:             "Write spec content from stdin",
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
		if err := client.WriteSpec(issueID, content); err != nil {
			return err
		}

		fmt.Printf("Spec written for %s\n", issueID)
		return nil
	},
}

var specDeleteCmd = &cobra.Command{
	Use:               "delete <issue-id>",
	Short:             "Delete the spec from an issue",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		client := exponential.NewClient(cfg)
		if err := client.DeleteSpec(issueID); err != nil {
			return err
		}

		fmt.Printf("Spec deleted from %s\n", issueID)
		return nil
	},
}

func runSpecJSON() {
	content, err := readStdinExplicit()
	if err != nil {
		exitJSONError(err)
	}
	var input jsonio.SpecToolInput
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
	path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "spec.md")

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	switch input.Operation {
	case "write":
		if input.Content == "" {
			exitJSONError(fmt.Errorf("'content' is required for write"))
		}
		if err := client.WriteSpec(issueID, input.Content); err != nil {
			exitJSONError(err)
		}
		enc.Encode(jsonio.SpecOutput{OK: true, IssueID: issueID, Path: path})

	case "read":
		content, err := client.ReadSpec(issueID)
		if err != nil {
			exitJSONError(err)
		}
		enc.Encode(jsonio.SpecOutput{OK: true, IssueID: issueID, Path: path, Content: content})

	case "delete":
		if err := client.DeleteSpec(issueID); err != nil {
			exitJSONError(err)
		}
		enc.Encode(jsonio.SpecOutput{OK: true, IssueID: issueID, Path: path})

	default:
		exitJSONError(fmt.Errorf("invalid operation %q: must be write, read, or delete", input.Operation))
	}
}

func init() {
	specCmd.Flags().BoolVar(&specJSONFlag, "json", false, "Read a structured payload as JSON from stdin")

	specCmd.AddCommand(specWriteCmd)
	specCmd.AddCommand(specDeleteCmd)
	rootCmd.AddCommand(specCmd)
}
