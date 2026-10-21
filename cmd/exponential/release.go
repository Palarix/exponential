package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var (
	releaseJSONFlag   bool
	releaseLabelFlag  string
	releaseGoalFlag   string
	releaseTargetFlag string
	releaseBeforeFlag string
	releaseAfterFlag  string
	releaseRemoveFlag bool
	releaseToFlag     string
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Manage releases (list, show, create, update, reorder, complete, delete)",
	Long: `Releases group issues that ship together. They are numbered #1, #2, ...
and may have a label. Current and Next are the first two open releases in
order; an unsaved slot is a placeholder that the first write saves.

A release ref is #7 (number), a label, @current or @next.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runReleaseList()
	},
}

var releaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Current, Next, upcoming and past releases",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runReleaseList()
	},
}

var releaseShowCmd = &cobra.Command{
	Use:   "show <release>",
	Short: "Show a release with its stats and issues",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)
		p, err := client.ReleaseProgress(args[0])
		if err != nil {
			releaseFail(err)
		}
		if releaseJSONFlag {
			s := jsonio.ToReleaseSummary(p.Release, "", p.Counts, p.Total, p.Done)
			printReleaseJSON(jsonio.ReleaseOutput{Release: &s})
			return
		}
		r := p.Release
		fmt.Printf("%s · %s\n", releaseHeading(r), strings.ToLower(string(r.State)))
		if r.Goal != "" {
			fmt.Printf("Goal: %s\n", r.Goal)
		}
		if r.TargetDate != "" {
			fmt.Printf("Target: %s\n", r.TargetDate)
		}
		fmt.Printf("%d/%d done\n\n", p.Done, p.Total)
		if r.Placeholder {
			fmt.Println("Not saved yet: the first write saves it.")
			return
		}
		issues, err := client.ListIssues(exponential.FilterOptions{Release: r.ID, All: true})
		if err != nil {
			releaseFail(err)
		}
		if len(issues) == 0 {
			fmt.Println("No issues in this release.")
			return
		}
		fmt.Print(ui.RenderIssueList(issues, ui.TerminalWidth(), cfg.Prefix))
	},
}

var releaseCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an open release after the existing ones",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		r, err := exponential.NewClient(cfg).CreateRelease(releaseFields(cmd))
		if err != nil {
			releaseFail(err)
		}
		printRelease(r, []string{fmt.Sprintf("Created %s", releaseHeading(r))})
	},
}

var releaseUpdateCmd = &cobra.Command{
	Use:   "update <release>",
	Short: "Set a release's label, goal or target date",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r, err := exponential.NewClient(cfg).UpdateRelease(args[0], releaseFields(cmd))
		if err != nil {
			releaseFail(err)
		}
		printRelease(r, []string{fmt.Sprintf("Updated %s", releaseHeading(r))})
	},
}

var releaseReorderCmd = &cobra.Command{
	Use:   "reorder <release> (--before <release> | --after <release>)",
	Short: "Move an open release before or after another",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r, err := exponential.NewClient(cfg).ReorderRelease(args[0], releaseBeforeFlag, releaseAfterFlag)
		if err != nil {
			releaseFail(err)
		}
		printRelease(r, []string{fmt.Sprintf("Moved %s", releaseHeading(r))})
	},
}

var releaseCompleteCmd = &cobra.Command{
	Use:   "complete <release>",
	Short: "Mark a release RELEASED (final); open issues move to the next release",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r, msgs, err := exponential.NewClient(cfg).CompleteRelease(args[0], releaseMode(), releaseToFlag)
		if err != nil {
			releaseFail(err)
		}
		printRelease(r, msgs)
	},
}

var releaseDeleteCmd = &cobra.Command{
	Use:   "delete <release>",
	Short: "Delete an open release; all its issues move to the next release",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		msgs, err := exponential.NewClient(cfg).DeleteRelease(args[0], releaseMode(), releaseToFlag)
		if err != nil {
			releaseFail(err)
		}
		if releaseJSONFlag {
			printReleaseJSON(jsonio.ReleaseOutput{Messages: msgs})
			return
		}
		for _, m := range msgs {
			fmt.Println(m)
		}
	},
}

func runReleaseList() {
	views, err := exponential.NewClient(cfg).ListReleases()
	if err != nil {
		releaseFail(err)
	}
	if releaseJSONFlag {
		out := jsonio.ReleaseOutput{Releases: make([]jsonio.ReleaseSummary, 0, len(views))}
		for _, v := range views {
			out.Releases = append(out.Releases, jsonio.ToReleaseSummary(v.Release, v.Slot, v.Counts, v.Total, v.Done))
		}
		printReleaseJSON(out)
		return
	}
	for _, v := range views {
		line := fmt.Sprintf("%-9s %s", strings.ToUpper(v.Slot[:1])+v.Slot[1:], releaseHeading(v.Release))
		switch {
		case v.Locked:
			line += ui.MutedStyle.Render("  locked until Current is saved")
		case v.Placeholder:
			line += ui.MutedStyle.Render("  placeholder")
		default:
			line += fmt.Sprintf("  %d/%d done", v.Done, v.Total)
		}
		if v.ReleasedAt != nil {
			line += ui.MutedStyle.Render("  released " + v.ReleasedAt.Format("Jan 2, 2006"))
		} else if v.TargetDate != "" {
			line += ui.MutedStyle.Render("  target " + v.TargetDate)
		}
		if v.Goal != "" {
			line += "  " + v.Goal
		}
		fmt.Println(line)
	}
}

// releaseHeading is "1.5 (#7)", or "#7" for a release without a label.
func releaseHeading(r *model.Release) string {
	if r.Label == "" {
		return r.Display()
	}
	return fmt.Sprintf("%s (%s)", r.Label, r.Ref())
}

func releaseFields(cmd *cobra.Command) exponential.ReleaseInput {
	var in exponential.ReleaseInput
	if cmd.Flags().Changed("label") {
		in.Label = &releaseLabelFlag
	}
	if cmd.Flags().Changed("goal") {
		in.Goal = &releaseGoalFlag
	}
	if cmd.Flags().Changed("target") {
		in.TargetDate = &releaseTargetFlag
	}
	return in
}

func releaseMode() string {
	if releaseRemoveFlag {
		return model.ReleaseRemove
	}
	return model.ReleaseMoveToNext
}

func printRelease(r *model.Release, msgs []string) {
	if releaseJSONFlag {
		s := jsonio.ToReleaseSummary(r, "", nil, 0, 0)
		printReleaseJSON(jsonio.ReleaseOutput{Release: &s, Messages: msgs})
		return
	}
	for _, m := range msgs {
		fmt.Println(m)
	}
}

func printReleaseJSON(out jsonio.ReleaseOutput) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

func releaseFail(err error) {
	if releaseJSONFlag {
		exitJSONError(err)
	}
	fmt.Printf("Error: %v\n", err)
	os.Exit(1)
}

func init() {
	releaseCmd.PersistentFlags().BoolVar(&releaseJSONFlag, "json", false, "Output as JSON")
	for _, c := range []*cobra.Command{releaseCreateCmd, releaseUpdateCmd} {
		c.Flags().StringVar(&releaseLabelFlag, "label", "", "Label, such as a version (1.5.0) or a name (Team Mode)")
		c.Flags().StringVar(&releaseGoalFlag, "goal", "", "One-line goal")
		c.Flags().StringVar(&releaseTargetFlag, "target", "", "Target date (YYYY-MM-DD), for display only")
	}
	releaseReorderCmd.Flags().StringVar(&releaseBeforeFlag, "before", "", "Move directly above this open release")
	releaseReorderCmd.Flags().StringVar(&releaseAfterFlag, "after", "", "Move directly below this open release")
	releaseCompleteCmd.Flags().BoolVar(&releaseRemoveFlag, "remove", false, "Remove open issues from the release instead of moving them to Next")
	releaseDeleteCmd.Flags().BoolVar(&releaseRemoveFlag, "remove", false, "Remove the issues from the release instead of moving them to Next")
	for _, c := range []*cobra.Command{releaseCompleteCmd, releaseDeleteCmd} {
		c.Flags().StringVar(&releaseToFlag, "to", "", "Move the issues to this open release (default: the release after it, else the one before it)")
	}

	releaseCmd.AddCommand(releaseListCmd, releaseShowCmd, releaseCreateCmd, releaseUpdateCmd,
		releaseReorderCmd, releaseCompleteCmd, releaseDeleteCmd)
	rootCmd.AddCommand(releaseCmd)
}
