package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/ui"
	"github.com/spf13/cobra"
)

var (
	addParentFlag   string
	addEstimateFlag int
	addDescFlag     string
	addForceFlag    bool
	addLabelFlag    []string
	addAssigneeFlag string
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new issue",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := beats.NewClient(cfg)

		var title string

		// Argument mode: title given directly
		if len(args) > 0 {
			title = strings.Join(args, " ")
		} else {
			fmt.Println("Interactive mode: Enter a title for the new issue.")
			fmt.Print("> ")
			var input string
			fmt.Scanln(&input)
			title = strings.TrimSpace(input)
		}

		if title == "" {
			fmt.Println("Error: title cannot be empty")
			os.Exit(1)
		}

		// Check estimate against configured estimation system
		if addEstimateFlag > 0 {
			if err := config.ValidateEstimate(cfg.EstimationSystem, addEstimateFlag); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}

		// Duplicate Check
		if !addForceFlag {
			dupes, err := client.CheckDuplicates(title)
			if err == nil && len(dupes) > 0 {
				fmt.Println(ui.WarningStyle.Render("⚠ Possible duplicate(s) found:"))
				for _, d := range dupes {
					fmt.Printf("  %s  %s\n", d.ID, d.Title)
				}
				fmt.Println("\nUse --force to create anyway, or choose a different title.")
				os.Exit(0)
			}
		}

		opts := beats.AddOptions{
			Title:       title,
			Description: addDescFlag,
			ParentID:    addParentFlag,
			Estimate:    addEstimateFlag,
			Assignee:    addAssigneeFlag,
			Labels:      addLabelFlag,
		}

		issue, err := client.AddIssue(opts)
		if err != nil {
			fmt.Printf("Error adding issue: %v\n", err)
			os.Exit(1)
		}

		showConfirmation(issue)
	},
}

func showConfirmation(issue *model.Issue) {
	fmt.Printf("Created %s: %s\n", issue.ID, issue.Title)
	if len(issue.Labels) > 0 {
		fmt.Printf("  Labels: %s\n", strings.Join(issue.Labels, ", "))
	}
	if issue.Assignee != "" {
		fmt.Printf("  Assignee: %s\n", issue.Assignee)
	}
	if issue.Estimate > 0 {
		fmt.Printf("  Estimate: %d pts\n", issue.Estimate)
	}
	if issue.ParentID != "" {
		fmt.Printf("  Parent: %s\n", issue.ParentID)
	}
}

func init() {
	addCmd.Flags().StringVarP(&addParentFlag, "parent", "p", "", "Parent issue ID")
	addCmd.Flags().IntVar(&addEstimateFlag, "sp", 0, "Story points estimate")
	addCmd.Flags().StringVar(&addDescFlag, "desc", "", "Issue description")
	addCmd.Flags().BoolVar(&addForceFlag, "force", false, "Skip duplicate check")
	addCmd.Flags().StringSliceVar(&addLabelFlag, "label", nil, "Labels (can be specified multiple times)")
	addCmd.Flags().StringVar(&addAssigneeFlag, "assignee", "", "Issue assignee")
	rootCmd.AddCommand(addCmd)
}
