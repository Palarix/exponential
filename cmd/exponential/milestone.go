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
	milestoneJSONFlag   bool
	milestoneLabelFlag  string
	milestoneGoalFlag   string
	milestoneTargetFlag string
	milestoneBeforeFlag string
	milestoneAfterFlag  string
	milestoneRemoveFlag bool
	milestoneToFlag     string
)

var milestoneCmd = &cobra.Command{
	Use:   "milestone",
	Short: "Manage milestones (list, show, create, update, reorder, complete, delete)",
	Long: `Milestones group issues that are done together. They are numbered #1, #2, ...
and may have a label. Current and Next are the first two open milestones in
order; an unsaved slot is a placeholder that the first write saves.

A milestone ref is #7 (number), a label, @current or @next.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runMilestoneList()
	},
}

var milestoneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Current, Next, upcoming and completed milestones",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runMilestoneList()
	},
}

var milestoneShowCmd = &cobra.Command{
	Use:   "show <milestone>",
	Short: "Show a milestone with its stats and issues",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)
		p, err := client.MilestoneProgress(args[0])
		if err != nil {
			milestoneFail(err)
		}
		if milestoneJSONFlag {
			s := jsonio.ToMilestoneSummary(p.Milestone, "", p.Counts, p.Total, p.Done, p.Points)
			printMilestoneJSON(jsonio.MilestoneOutput{Milestone: &s})
			return
		}
		r := p.Milestone
		fmt.Printf("%s · %s\n", milestoneHeading(r), strings.ToLower(string(r.State)))
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
		issues, err := client.ListIssues(exponential.FilterOptions{Milestone: r.ID, All: true})
		if err != nil {
			milestoneFail(err)
		}
		if len(issues) == 0 {
			fmt.Println("No issues in this milestone.")
			return
		}
		fmt.Print(ui.RenderIssueList(issues, ui.TerminalWidth(), cfg.Prefix))
	},
}

var milestoneCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an open milestone after the existing ones",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		r, err := exponential.NewClient(cfg).CreateMilestone(milestoneFields(cmd))
		if err != nil {
			milestoneFail(err)
		}
		printMilestone(r, []string{fmt.Sprintf("Created %s", milestoneHeading(r))})
	},
}

var milestoneUpdateCmd = &cobra.Command{
	Use:   "update <milestone>",
	Short: "Set a milestone's label, goal or target date",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r, err := exponential.NewClient(cfg).UpdateMilestone(args[0], milestoneFields(cmd))
		if err != nil {
			milestoneFail(err)
		}
		printMilestone(r, []string{fmt.Sprintf("Updated %s", milestoneHeading(r))})
	},
}

var milestoneReorderCmd = &cobra.Command{
	Use:   "reorder <milestone> (--before <milestone> | --after <milestone>)",
	Short: "Move an open milestone before or after another",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r, err := exponential.NewClient(cfg).ReorderMilestone(args[0], milestoneBeforeFlag, milestoneAfterFlag)
		if err != nil {
			milestoneFail(err)
		}
		printMilestone(r, []string{fmt.Sprintf("Moved %s", milestoneHeading(r))})
	},
}

var milestoneCompleteCmd = &cobra.Command{
	Use:   "complete <milestone>",
	Short: "Mark a milestone COMPLETED (final); open issues move to the next milestone",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r, msgs, err := exponential.NewClient(cfg).CompleteMilestone(args[0], milestoneMode(), milestoneToFlag)
		if err != nil {
			milestoneFail(err)
		}
		printMilestone(r, msgs)
	},
}

var milestoneDeleteCmd = &cobra.Command{
	Use:   "delete <milestone>",
	Short: "Delete an open milestone; all its issues move to the next milestone",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		msgs, err := exponential.NewClient(cfg).DeleteMilestone(args[0], milestoneMode(), milestoneToFlag)
		if err != nil {
			milestoneFail(err)
		}
		if milestoneJSONFlag {
			printMilestoneJSON(jsonio.MilestoneOutput{Messages: msgs})
			return
		}
		for _, m := range msgs {
			fmt.Println(m)
		}
	},
}

func runMilestoneList() {
	views, err := exponential.NewClient(cfg).ListMilestones()
	if err != nil {
		milestoneFail(err)
	}
	if milestoneJSONFlag {
		out := jsonio.MilestoneOutput{Milestones: make([]jsonio.MilestoneSummary, 0, len(views))}
		for _, v := range views {
			out.Milestones = append(out.Milestones, jsonio.ToMilestoneSummary(v.Milestone, v.Slot, v.Counts, v.Total, v.Done, v.Points))
		}
		printMilestoneJSON(out)
		return
	}
	for _, v := range views {
		line := fmt.Sprintf("%-9s %s", strings.ToUpper(v.Slot[:1])+v.Slot[1:], milestoneHeading(v.Milestone))
		switch {
		case v.Locked:
			line += ui.MutedStyle.Render("  locked until Current is saved")
		case v.Placeholder:
			line += ui.MutedStyle.Render("  placeholder")
		default:
			line += fmt.Sprintf("  %d/%d done", v.Done, v.Total)
		}
		if v.CompletedAt != nil {
			line += ui.MutedStyle.Render("  completed " + v.CompletedAt.Format("Jan 2, 2006"))
		} else if v.TargetDate != "" {
			line += ui.MutedStyle.Render("  target " + v.TargetDate)
		}
		if v.Goal != "" {
			line += "  " + v.Goal
		}
		fmt.Println(line)
	}
}

// milestoneHeading is "1.5 (#7)", or "#7" for a milestone without a label.
func milestoneHeading(r *model.Milestone) string {
	if r.Label == "" {
		return r.Display()
	}
	return fmt.Sprintf("%s (%s)", r.Label, r.Ref())
}

func milestoneFields(cmd *cobra.Command) exponential.MilestoneInput {
	var in exponential.MilestoneInput
	if cmd.Flags().Changed("label") {
		in.Label = &milestoneLabelFlag
	}
	if cmd.Flags().Changed("goal") {
		in.Goal = &milestoneGoalFlag
	}
	if cmd.Flags().Changed("target") {
		in.TargetDate = &milestoneTargetFlag
	}
	return in
}

func milestoneMode() string {
	if milestoneRemoveFlag {
		return model.MilestoneRemove
	}
	return model.MilestoneMoveToNext
}

func printMilestone(r *model.Milestone, msgs []string) {
	if milestoneJSONFlag {
		s := jsonio.ToMilestoneSummary(r, "", nil, 0, 0, nil)
		printMilestoneJSON(jsonio.MilestoneOutput{Milestone: &s, Messages: msgs})
		return
	}
	for _, m := range msgs {
		fmt.Println(m)
	}
}

func printMilestoneJSON(out jsonio.MilestoneOutput) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

func milestoneFail(err error) {
	if milestoneJSONFlag {
		exitJSONError(err)
	}
	fmt.Printf("Error: %v\n", err)
	os.Exit(1)
}

func init() {
	milestoneCmd.PersistentFlags().BoolVar(&milestoneJSONFlag, "json", false, "Output as JSON")
	for _, c := range []*cobra.Command{milestoneCreateCmd, milestoneUpdateCmd} {
		c.Flags().StringVar(&milestoneLabelFlag, "label", "", "Label, such as a version (1.5.0) or a name (Team Mode)")
		c.Flags().StringVar(&milestoneGoalFlag, "goal", "", "One-line goal")
		c.Flags().StringVar(&milestoneTargetFlag, "target", "", "Target date (YYYY-MM-DD), for display only")
	}
	milestoneReorderCmd.Flags().StringVar(&milestoneBeforeFlag, "before", "", "Move directly above this open milestone")
	milestoneReorderCmd.Flags().StringVar(&milestoneAfterFlag, "after", "", "Move directly below this open milestone")
	milestoneCompleteCmd.Flags().BoolVar(&milestoneRemoveFlag, "remove", false, "Remove open issues from the milestone instead of moving them to Next")
	milestoneDeleteCmd.Flags().BoolVar(&milestoneRemoveFlag, "remove", false, "Remove the issues from the milestone instead of moving them to Next")
	for _, c := range []*cobra.Command{milestoneCompleteCmd, milestoneDeleteCmd} {
		c.Flags().StringVar(&milestoneToFlag, "to", "", "Move the issues to this open milestone (default: the milestone after it, else the one before it)")
	}

	milestoneCmd.AddCommand(milestoneListCmd, milestoneShowCmd, milestoneCreateCmd, milestoneUpdateCmd,
		milestoneReorderCmd, milestoneCompleteCmd, milestoneDeleteCmd)
	rootCmd.AddCommand(milestoneCmd)
}
