package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	mergeSquash       bool
	mergeFF           bool
	mergeDeleteBranch bool
	mergeKeepBranch   bool
	mergeNoWorktree   bool
	mergeJSONFlag     bool
)

var mergeCmd = &cobra.Command{
	Use:               "merge [id]",
	Short:             "Merge an issue's branch and close the issue",
	Long:              "Merge the branch into the default branch, record a MERGE event, and transition to DONE. When worktrees are enabled (default), the merge runs from the hub checkout on main and the worktree is removed automatically. Use --no-wt for the classic checkout-based flow. If no issue ID is given, infers from the current branch.",
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
	if mergeJSONFlag {
		runMergeJSON()
		return
	}

	if mergeNoWorktree {
		cfg.Worktrees = false
	}
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

	if err := exponential.HubCleanForMerge(issue.BranchStats.Branch); err != nil {
		fmt.Printf("Error: %v\n", err)
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

func runMergeJSON() {
	content, err := readStdinExplicit()
	if err != nil {
		exitJSONError(err)
	}
	var input jsonio.MergeToolInput
	if err := jsonio.DecodeStrict(content, &input); err != nil {
		exitJSONError(err)
	}
	if input.ID == "" {
		exitJSONError(fmt.Errorf("'id' is required"))
	}

	strategy := exponential.MergeStrategySquash
	switch input.Strategy {
	case "", "squash":
		// default
	case "merge":
		strategy = exponential.MergeStrategyMerge
	case "ff":
		strategy = exponential.MergeStrategyFF
	default:
		exitJSONError(fmt.Errorf("invalid merge strategy %q: must be one of squash, merge, ff", input.Strategy))
	}

	client := exponential.NewClient(cfg)
	issue, err := client.ResolveReviewIssue(input.ID)
	if err != nil {
		exitJSONError(err)
	}
	if issue.BranchStats == nil {
		exitJSONError(fmt.Errorf("no branch found for %s", issue.ID))
	}

	if err := exponential.HubCleanForMerge(issue.BranchStats.Branch); err != nil {
		exitJSONError(err)
	}

	result, err := client.MergeIssue(issue.ID, exponential.MergeOptions{
		Strategy:      strategy,
		CommitMessage: input.CommitMessage,
		DeleteBranch:  !input.KeepBranch,
	})
	if err != nil {
		exitJSONError(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(jsonio.MergeOutput{ID: issue.ID, MergeSHA: result.MergeSHA, Messages: result.Messages})
}

func init() {
	mergeCmd.Flags().BoolVar(&mergeSquash, "squash", false, "Squash merge into a single commit")
	mergeCmd.Flags().BoolVar(&mergeFF, "ff", false, "Fast-forward only (fails if not possible)")
	mergeCmd.Flags().BoolVarP(&mergeDeleteBranch, "delete-branch", "d", false, "Delete branch after merge")
	mergeCmd.Flags().BoolVar(&mergeKeepBranch, "keep-branch", false, "Keep branch after merge (skip prompt)")
	mergeCmd.Flags().BoolVar(&mergeNoWorktree, "no-wt", false, "Use checkout-based merge instead of worktree-aware merge")
	mergeCmd.Flags().BoolVar(&mergeJSONFlag, "json", false, "Read a structured merge payload as JSON from stdin")
	rootCmd.AddCommand(mergeCmd)
}
