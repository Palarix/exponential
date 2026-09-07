package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/storage"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var (
	rationaleTopFlag  int
	rationaleJSONFlag bool
)

var rationaleCmd = &cobra.Command{
	Use:   "rationale <query>",
	Short: "Search specs and walkthroughs for design rationale",
	Long: `Full-text search across all spec and walkthrough artifacts.

Uses BM25 ranking with title/label boosting and proximity scoring to find
the most relevant design rationale for a given query.

Examples:
  xpo rationale "branch badge display"
  xpo rationale "merge strategy" --top 10
  xpo rationale "merge strategy" --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)
		result, err := client.SearchRationale(args[0], rationaleTopFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if rationaleJSONFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(result)
			return
		}

		data := ui.RationaleData{
			Query:        result.Query,
			TotalMatches: result.TotalMatches,
			Results:      make([]ui.RationaleHit, len(result.Results)),
		}
		for i, r := range result.Results {
			data.Results[i] = ui.RationaleHit{
				IssueID:      r.IssueID,
				Title:        r.Title,
				Status:       r.Status,
				Labels:       r.Labels,
				Document:     r.Document,
				Fragment:     r.Fragment,
				Score:        r.Score,
				ArtifactPath: filepath.Join(storage.XpoDir(), "artifacts", r.IssueID, r.Document+".md"),
			}
		}
		fmt.Print(ui.RenderRationale(data, ui.TerminalWidth()))
	},
}

func init() {
	rationaleCmd.Flags().IntVarP(&rationaleTopFlag, "top", "n", 5, "Maximum results to return (max 20)")
	rationaleCmd.Flags().BoolVar(&rationaleJSONFlag, "json", false, "Output results as JSON")
	rootCmd.AddCommand(rationaleCmd)
}
