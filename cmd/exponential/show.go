package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	showJSONFlag            bool
	showWithSpecFlag        bool
	showWithWalkthroughFlag bool
)

var showCmd = &cobra.Command{
	Use:               "show [id]",
	Short:             "Show issue details and history",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		showIssue(args[0])
	},
}

func showIssue(id string) {
	client := exponential.NewClient(cfg)
	issue, children, archived, err := client.FindIssue(id)
	if err != nil {
		if showJSONFlag {
			exitJSONError(err)
		}
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if showJSONFlag {
		out := jsonio.ShowOutput{
			ID:               issue.ID,
			Title:            issue.Title,
			Status:           string(issue.Status),
			IsInferred:       issue.InferredStatus,
			Description:      issue.Description,
			Labels:           issue.Labels,
			ParentID:         issue.ParentID,
			StoryPoints:      issue.Estimate,
			Assignee:         issue.Assignee,
			CycleID:          issue.CycleID,
			EffectiveCycleID: issue.EffectiveCycleID,
			BranchStats:      issue.BranchStats,
			CreatedBy:        issue.CreatedBy,
			CreatedAt:        issue.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        issue.UpdatedAt.Format(time.RFC3339),
			Dependencies:     issue.Dependencies,
			Artifacts:        jsonio.ToArtifactEntries(issue.Artifacts),
			Comments:         jsonio.ToCommentSummaries(issue.Comments),
			Events:           jsonio.ToEventSummaries(issue.Events),
			Archived:         archived,
		}
		if len(children) > 0 {
			out.Children = make([]jsonio.IssueSummary, len(children))
			for i, c := range children {
				out.Children[i] = jsonio.ToIssueSummary(c)
			}
		}
		if showWithSpecFlag {
			if content, err := client.ReadSpec(issue.ID); err == nil {
				out.Spec = content
			}
		}
		if showWithWalkthroughFlag {
			if content, err := client.ReadWalkthrough(issue.ID); err == nil {
				out.Walkthrough = content
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termWidth <= 0 {
		termWidth = 100
	}

	var opts ui.DetailOptions
	if showWithSpecFlag {
		if content, err := client.ReadSpec(issue.ID); err == nil {
			opts.Spec = content
		}
	}
	if showWithWalkthroughFlag {
		if content, err := client.ReadWalkthrough(issue.ID); err == nil {
			opts.Walkthrough = content
		}
	}

	fmt.Println(ui.RenderIssueDetails(issue, children, archived, termWidth, opts))
}

func init() {
	showCmd.Flags().BoolVar(&showJSONFlag, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showWithSpecFlag, "with-spec", false, "Include spec content")
	showCmd.Flags().BoolVar(&showWithWalkthroughFlag, "with-walkthrough", false, "Include walkthrough content")
	rootCmd.AddCommand(showCmd)
}
