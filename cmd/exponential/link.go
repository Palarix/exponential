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
	linkTypeFlag      string
	linkJSONFlag      bool
	linkOverwriteFlag bool
)

var linkCmd = &cobra.Command{
	Use:   "link <source-id> <target-id> -t <type>",
	Short: "Add a dependency or relationship between two issues",
	Long:  `Add a dependency or relationship between two issues. Links are bidirectional: A blocks B also means B blocked_by A, whichever side stores it. Two issues can have at most one link: linking an already-linked pair fails unless --overwrite is set, which replaces the existing link. Only blocked_by prevents start; all other kinds are informational. Pass --json to read a structured payload {"source": "...", "target": "...", "type": "...", "overwrite": false} from stdin.`,
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
			added, replaced, _, err := client.Link(input.Source, input.Target, input.Type, input.Overwrite)
			if err != nil {
				exitJSONError(err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(jsonio.NewLinkOutput(added, replaced))
			return
		}

		if len(args) != 2 {
			fmt.Println("Error: exactly 2 arguments required: <source-id> <target-id>")
			cmd.Help()
			os.Exit(1)
		}

		added, replaced, msgs, err := client.Link(args[0], args[1], linkTypeFlag, linkOverwriteFlag)
		if err != nil {
			fmt.Printf("Error linking issues: %v\n", err)
			os.Exit(1)
		}

		for _, msg := range msgs {
			fmt.Println(msg)
		}
		fmt.Println(jsonio.NewLinkOutput(added, replaced).LinkSummary())
	},
}

func init() {
	linkCmd.Flags().StringVarP(&linkTypeFlag, "type", "t", "", "Type of relationship: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to (required)")
	linkCmd.Flags().BoolVar(&linkJSONFlag, "json", false, "Read a structured payload as JSON from stdin")
	linkCmd.Flags().BoolVar(&linkOverwriteFlag, "overwrite", false, "Replace an existing link of a different kind between the two issues")
	rootCmd.AddCommand(linkCmd)
}
