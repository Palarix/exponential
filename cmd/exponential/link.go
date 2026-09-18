package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
	"github.com/spf13/cobra"
)

var (
	linkTypeFlag string
	linkJSONFlag bool
)

var linkCmd = &cobra.Command{
	Use:   "link <source-id> <target-id> -t <type>",
	Short: "Add a dependency or relationship between two issues",
	Long:  `Add a dependency or relationship between two issues. Pass --json to read a structured payload {"source": "...", "target": "...", "type": "..."} from stdin.`,
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)

		if linkJSONFlag {
			content, err := readStdinExplicit()
			if err != nil {
				exitJSONError(err)
			}
			var input jsonio.LinkToolInput
			if err := jsonio.DecodeStrict(content, &input); err != nil {
				exitJSONError(err)
			}
			if input.Source == "" || input.Target == "" {
				exitJSONError(fmt.Errorf("'source' and 'target' are required"))
			}
			kind := model.NormalizeDependencyKind(input.Type)
			if kind == "" {
				exitJSONError(fmt.Errorf("invalid link type %q", input.Type))
			}
			src, err := client.GetIssue(input.Source)
			if err != nil {
				exitJSONError(fmt.Errorf("source: %w", err))
			}
			tgt, err := client.GetIssue(input.Target)
			if err != nil {
				exitJSONError(fmt.Errorf("target: %w", err))
			}
			if src.ID == tgt.ID {
				exitJSONError(fmt.Errorf("cannot link an issue to itself"))
			}
			for _, dep := range src.Dependencies {
				if dep.TargetID == tgt.ID && string(dep.Kind) == kind {
					exitJSONError(fmt.Errorf("link %s %s already exists on %s", kind, tgt.ID, src.ID))
				}
			}
			newDeps := append(src.Dependencies, model.Dependency{
				SourceID: src.ID,
				TargetID: tgt.ID,
				Kind:     model.DependencyKind(kind),
			})
			if _, err := client.UpdateIssue(src.ID, model.UpdatePayload{Dependencies: newDeps}, "link"); err != nil {
				exitJSONError(err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(jsonio.LinkOutput{Source: src.ID, Target: tgt.ID, Kind: kind})
			return
		}

		if len(args) != 2 {
			fmt.Println("Error: exactly 2 arguments required: <source-id> <target-id>")
			cmd.Help()
			os.Exit(1)
		}

		sourceID := args[0]
		targetID := args[1]

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
	linkCmd.Flags().BoolVar(&linkJSONFlag, "json", false, "Read a structured link payload as JSON from stdin")
	rootCmd.AddCommand(linkCmd)
}
