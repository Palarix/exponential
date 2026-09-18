package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var blockedJSONFlag bool

var blockedCmd = &cobra.Command{
	Use:               "blocked [id]",
	Short:             "List blocked issues, or mark an issue as BLOCKED",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)

		if len(args) == 0 {
			issues, err := client.ListIssues(exponential.FilterOptions{
				Statuses: []string{"BLOCKED"},
			})
			if err != nil {
				if blockedJSONFlag {
					exitJSONError(err)
				}
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if blockedJSONFlag {
				out := jsonio.ListOutput{Issues: make([]jsonio.IssueSummary, 0, len(issues))}
				for _, i := range issues {
					out.Issues = append(out.Issues, jsonio.ToIssueSummary(i))
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(out)
				return
			}

			if len(issues) == 0 {
				fmt.Println("No blocked issues.")
				return
			}

			width := ui.TerminalWidth()
			fmt.Print(ui.RenderIssueList(issues, width, cfg.Prefix))
			return
		}

		if blockedJSONFlag {
			issue, err := client.GetIssue(args[0])
			if err != nil {
				exitJSONError(err)
			}
			status := string(model.StatusBlocked)
			payload := model.UpdatePayload{Status: &status}
			msgs, err := client.UpdateIssue(issue.ID, payload, "block")
			if err != nil {
				exitJSONError(err)
			}
			out := jsonio.UpdateOutput{ID: issue.ID, Messages: msgs}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)
			return
		}

		status := string(model.StatusBlocked)
		payload := model.UpdatePayload{Status: &status}
		msgs, err := client.UpdateIssue(args[0], payload, "block")
		if err != nil {
			fmt.Printf("Error marking blocked: %v\n", err)
			os.Exit(1)
		}
		for _, msg := range msgs {
			fmt.Println(msg)
		}
	},
}

func init() {
	blockedCmd.Flags().BoolVar(&blockedJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(blockedCmd)
}
