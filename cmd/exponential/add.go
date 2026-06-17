package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/inputs"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var (
	addParentFlag   string
	addEstimateFlag int
	addDescFlag     string
	addForceFlag    bool
	addLabelFlag    []string
	addAssigneeFlag string
	addCycleFlag    string
	addJSONFlag     bool
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new issue",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)

		// JSON payload mode: read full structured input from stdin. Other
		// field flags are ignored; --force still applies.
		if addJSONFlag {
			content, err := readStdinExplicit()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			var input inputs.AddInput
			if err := inputs.DecodeStrict(content, &input); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			payload, err := input.ToCreatePayload()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			if payload.Estimate > 0 {
				if err := config.ValidateEstimate(cfg.EstimationSystem, payload.Estimate); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
			}
			if !addForceFlag {
				if dupes, derr := client.CheckDuplicates(payload.Title); derr == nil && len(dupes) > 0 {
					fmt.Println(ui.WarningStyle.Render("⚠ Possible duplicate(s) found:"))
					for _, d := range dupes {
						fmt.Printf("  %s  %s\n", d.ID, d.Title)
					}
					fmt.Println("\nUse --force to create anyway, or choose a different title.")
					os.Exit(0)
				}
			}
			issue, err := client.AddIssue(payload)
			if err != nil {
				fmt.Printf("Error adding issue: %v\n", err)
				os.Exit(1)
			}
			showConfirmation(issue)
			return
		}

		var title string

		// Argument mode: title given directly
		if len(args) > 0 {
			title = strings.Join(args, " ")

			// Resolve description from stdin if requested.
			if cmd.Flags().Changed("desc") && addDescFlag == "-" {
				content, err := readStdinExplicit()
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				addDescFlag = content
			} else if !cmd.Flags().Changed("desc") && isStdinPiped() {
				content, err := readAllStdin()
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				addDescFlag = content
			}
		} else {
			fmt.Println("Interactive mode: Enter a title for the new issue.")
			fmt.Print("> ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
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

		payload := model.CreatePayload{
			Title:       title,
			Description: addDescFlag,
			ParentID:    addParentFlag,
			Estimate:    addEstimateFlag,
			Assignee:    addAssigneeFlag,
			CycleID:     resolveCycleID(addCycleFlag),
			Labels:      addLabelFlag,
		}

		issue, err := client.AddIssue(payload)
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
	addCmd.Flags().StringVar(&addCycleFlag, "cycle", "", "Assign to cycle (current, next, or YYYY-MM-DD)")
	addCmd.Flags().BoolVar(&addJSONFlag, "json", false, "Read a full issue payload as JSON from stdin")
	rootCmd.AddCommand(addCmd)
}
