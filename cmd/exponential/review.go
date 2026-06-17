package main

import (
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var reviewCmd = &cobra.Command{
	Use:               "review [id]",
	Short:             "Review changes for an issue's branch",
	Long:              "Show a unified review of the branch changes: commits, description, files changed, and full diff. If no issue ID is given, infers from the current branch.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := ""
		if len(args) > 0 {
			id = args[0]
		}
		runReview(id)
	},
}

func runReview(id string) {
	client := exponential.NewClient(cfg)
	issue, err := client.ResolveReviewIssue(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if issue.BranchStats == nil {
		fmt.Printf("No branch found for %s.\n", issue.ID)
		fmt.Println("Create a branch with 'xpo start', or check that the branch name contains the issue ID.")
		os.Exit(1)
	}

	if issue.BranchStats.Commits == 0 {
		fmt.Printf("Branch %s has no commits ahead of %s.\n", issue.BranchStats.Branch, exponential.DefaultBranch())
		os.Exit(0)
	}

	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termWidth <= 0 {
		termWidth = 100
	}

	base := exponential.DefaultBranch()
	branch := issue.BranchStats.Branch

	// Gather review data
	commits := exponential.ListBranchCommits(branch, base)
	files := exponential.ListFilesChanged(branch, base)

	reviewCommits := make([]ui.ReviewCommit, len(commits))
	for i, c := range commits {
		reviewCommits[i] = ui.ReviewCommit{SHA: c.SHA, Message: c.Message}
	}
	reviewFiles := make([]ui.ReviewFile, len(files))
	for i, f := range files {
		reviewFiles[i] = ui.ReviewFile{
			Status:     f.Status,
			Path:       f.Path,
			Insertions: f.Insertions,
			Deletions:  f.Deletions,
		}
	}

	data := ui.ReviewData{
		Issue:   issue,
		Base:    base,
		Commits: reviewCommits,
		Files:   reviewFiles,
	}

	fmt.Print(ui.RenderReview(data, termWidth))

	fmt.Println()
	fmt.Printf("%s\n", ui.MutedStyle.Render(fmt.Sprintf("To view the full diff: git diff %s...%s", base, branch)))
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}
