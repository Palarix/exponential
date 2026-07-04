package main

import (
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/spf13/cobra"
)

var (
	driveFilter     string
	driveTestCmd    string
	driveDryRun     bool
	driveNoMerge    bool
	driveResume     bool
	driveMaxRetries int
	driveSupervisor string
	driveCoder      string
)

var driveCmd = &cobra.Command{
	Use:               "drive [issue-id]",
	Short:             "Autonomous agent execution loop for a single ticket",
	Long: `Pick a PLANNED ticket, hand it to an AI coding agent loop,
verify the result, and commit.

Configure defaults in .xpo/config.yaml:

  drive:
    supervisor: claude
    coder: claude
    max_retries: 3
    test_cmd: make test`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		runDrive(args)
	},
}

func runDrive(args []string) {
	id := ""
	if len(args) > 0 {
		id = args[0]
	}

	client := exponential.NewClient(cfg)
	client.OnBehalfOf = client.GetUser()

	opts := exponential.DriveOptions{
		IssueID:    id,
		Filter:     driveFilter,
		TestCmd:    driveTestCmd,
		DryRun:     driveDryRun,
		NoMerge:    driveNoMerge,
		Resume:     driveResume,
		MaxRetries: driveMaxRetries,
		Supervisor: driveSupervisor,
		Coder:      driveCoder,
	}

	result, err := client.DriveIssue(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch result.Status {
	case "no-work":
		fmt.Println("No eligible issues found.")
		os.Exit(2)
	case "blocked":
		os.Exit(1)
	}
}

func init() {
	driveCmd.Flags().StringVar(&driveFilter, "filter", "", "Filter issues by label")
	driveCmd.Flags().StringVar(&driveTestCmd, "test-cmd", "", "Override test command")
	driveCmd.Flags().BoolVar(&driveDryRun, "dry-run", false, "Show which ticket would be picked without executing")
	driveCmd.Flags().BoolVar(&driveNoMerge, "no-merge", false, "Skip merge — leave branch for human review")
	driveCmd.Flags().BoolVar(&driveResume, "resume", false, "Resume an in-progress issue from a previous drive")
	driveCmd.Flags().IntVar(&driveMaxRetries, "max-retries", 0, "Max implementation attempts (default: from config or 3)")
	driveCmd.Flags().StringVar(&driveSupervisor, "supervisor", "", "Supervisor agent (default: from config or claude)")
	driveCmd.Flags().StringVar(&driveCoder, "coder", "", "Coder agent (default: from config or claude)")
	rootCmd.AddCommand(driveCmd)
}
