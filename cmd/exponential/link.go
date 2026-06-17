package main

import (
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
	"github.com/spf13/cobra"
)

var (
	linkTypeFlag string
)

var linkCmd = &cobra.Command{
	Use:   "link <source-id> <target-id> -t <type>",
	Short: "Add a dependency or relationship between two issues",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)

		sourceID := args[0]
		targetID := args[1]

		// Validate issue IDs
		sourceIssue, err := client.GetIssue(sourceID)
		if err != nil {
			fmt.Printf("Error: Source issue '%s' not found: %v\n", sourceID, err)
			os.Exit(1)
		}
		targetIssue, err := client.GetIssue(targetID)
		if err != nil {
			fmt.Printf("Error: Target issue '%s' not found: %v\n", targetID, err)
			os.Exit(1)
		}

		// Validate dependency type
		depKind := model.NormalizeDependencyKind(linkTypeFlag)
		if depKind == "" {
			fmt.Printf("Error: Invalid dependency type '%s'\n", linkTypeFlag)
			cmd.Help()
			os.Exit(1)
		}

		dependency := model.Dependency{
			SourceID: sourceIssue.ID,
			TargetID: targetIssue.ID,
			Kind:     model.DependencyKind(depKind),
		}

		newDeps := append(sourceIssue.Dependencies, dependency)
		payload := model.UpdatePayload{
			Dependencies: newDeps,
		}

		msgs, err := client.UpdateIssue(sourceIssue.ID, payload, "link")
		if err != nil {
			fmt.Printf("Error linking issues: %v\n", err)
			os.Exit(1)
		}

		for _, msg := range msgs {
			fmt.Println(msg)
		}
		fmt.Printf("Linked %s %s %s\n", sourceIssue.ID, depKind, targetIssue.ID)
	},
}

func init() {
	linkCmd.Flags().StringVarP(&linkTypeFlag, "type", "t", "", "Type of relationship: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to (required)")
	linkCmd.MarkFlagRequired("type")
	rootCmd.AddCommand(linkCmd)
}
