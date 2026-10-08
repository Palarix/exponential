package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/spf13/cobra"
)

var (
	unlinkTypeFlag string
	unlinkJSONFlag bool
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink <source-id> <target-id> -t <type>",
	Short: "Remove one relationship between two issues",
	Long:  `Remove one relationship between two issues, the counterpart of link. Links are bidirectional, so "unlink B A -t blocked_by" also removes "A blocks B" stored on A. A link to a deleted issue can be removed by its ID. Pass --json to read a structured payload {"source": "...", "target": "...", "type": "..."} from stdin.`,
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)

		if unlinkJSONFlag {
			content, err := readStdinExplicit()
			if err != nil {
				exitJSONError(err)
			}
			var input jsonio.LinkToolInput
			if err := jsonio.DecodeStrict(content, &input); err != nil {
				exitJSONError(err)
			}
			dep, _, err := client.Unlink(input.Source, input.Target, input.Type)
			if err != nil {
				exitJSONError(err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(jsonio.LinkOutput{Source: dep.SourceID, Target: dep.TargetID, Kind: string(dep.Kind)})
			return
		}

		if len(args) != 2 {
			fmt.Println("Error: exactly 2 arguments required: <source-id> <target-id>")
			cmd.Help()
			os.Exit(1)
		}

		dep, msgs, err := client.Unlink(args[0], args[1], unlinkTypeFlag)
		if err != nil {
			fmt.Printf("Error unlinking issues: %v\n", err)
			os.Exit(1)
		}
		for _, msg := range msgs {
			fmt.Println(msg)
		}
		fmt.Printf("Unlinked %s %s %s\n", dep.SourceID, dep.Kind, dep.TargetID)
	},
}

func init() {
	unlinkCmd.Flags().StringVarP(&unlinkTypeFlag, "type", "t", "", "Type of relationship: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to (required)")
	unlinkCmd.Flags().BoolVar(&unlinkJSONFlag, "json", false, "Read a structured payload as JSON from stdin")
	rootCmd.AddCommand(unlinkCmd)
}
