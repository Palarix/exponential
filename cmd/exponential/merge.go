package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	mergeSquash      bool
	mergeFF          bool
	mergeDeleteBranch bool
	mergeKeepBranch  bool
)

var mergeCmd = &cobra.Command{
	Use:               "merge [id]",
	Short:             "Merge an issue's branch and close the issue",
	Long:              "Merge the branch into the default branch, record a MERGE event, and transition to DONE. If no issue ID is given, infers from the current branch.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		id := ""
		if len(args) > 0 {
			id = args[0]
		}
		runMerge(id)
	},
}

func runMerge(id string) {
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

	// Fail early if working tree is dirty
	if !exponential.IsWorkingTreeClean() {
		fmt.Println("Error: working tree is not clean — commit or stash your changes first.")
		os.Exit(1)
	}

	// Show compact summary
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termWidth <= 0 {
		termWidth = 100
	}

	base := exponential.DefaultBranch()
	bs := issue.BranchStats
	headerStyle := ui.BoldStyle
	fmt.Printf("%s\n", headerStyle.Render(issue.Title))
	fmt.Printf("%s → %s\n", bs.Branch, ui.MutedStyle.Render(base))

	commitWord := "commits"
	if bs.Commits == 1 {
		commitWord = "commit"
	}
	fmt.Printf("⊙ %s  %d %s, %d files, +%d -%d\n\n",
		bs.HeadSHA, bs.Commits, commitWord, bs.FilesChanged, bs.Insertions, bs.Deletions)

	// Determine strategy
	var strategy exponential.MergeStrategy
	switch {
	case mergeSquash:
		strategy = exponential.MergeStrategySquash
	case mergeFF:
		strategy = exponential.MergeStrategyFF
	default:
		strategy = promptStrategy()
	}

	fmt.Printf("Strategy: %s\n", ui.AccentStyle.Render(string(strategy)))

	// Determine branch deletion
	deleteBranch := mergeDeleteBranch
	if !mergeDeleteBranch && !mergeKeepBranch {
		fmt.Print("Delete branch after merge? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		deleteBranch = strings.TrimSpace(strings.ToLower(answer)) == "y"
	}

	opts := exponential.MergeOptions{
		Strategy:     strategy,
		DeleteBranch: deleteBranch,
		KeepBranch:   mergeKeepBranch,
	}

	result, err := client.MergeIssue(issue.ID, opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	for _, msg := range result.Messages {
		fmt.Println(msg)
	}
}

func promptStrategy() exponential.MergeStrategy {
	fmt.Println("Merge strategy:")
	fmt.Println("  [1] squash  — single commit on main")
	fmt.Println("  [2] merge   — merge commit (--no-ff)")
	fmt.Println("  [3] ff      — fast-forward only")
	fmt.Print("Choose [1]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	switch strings.TrimSpace(answer) {
	case "2":
		return exponential.MergeStrategyMerge
	case "3":
		return exponential.MergeStrategyFF
	default:
		return exponential.MergeStrategySquash
	}
}

func init() {
	mergeCmd.Flags().BoolVar(&mergeSquash, "squash", false, "Squash merge into a single commit")
	mergeCmd.Flags().BoolVar(&mergeFF, "ff", false, "Fast-forward only (fails if not possible)")
	mergeCmd.Flags().BoolVarP(&mergeDeleteBranch, "delete-branch", "d", false, "Delete branch after merge")
	mergeCmd.Flags().BoolVar(&mergeKeepBranch, "keep-branch", false, "Keep branch after merge (skip prompt)")
	rootCmd.AddCommand(mergeCmd)
}
