package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	historyJSON    bool
	historyReverse bool
	historySince   string
	historyLimit   int
	historyNoPager bool
)

var historyCmd = &cobra.Command{
	Use:               "history [id]",
	Short:             "Show activity timeline for an issue or the whole project",
	Long:              `Without an issue ID, shows a project-wide timeline of recent activity including issue events and git commits. With an issue ID, shows the full activity timeline for that issue plus commits on its branch.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeIssueIDs,
	Run: func(cmd *cobra.Command, args []string) {
		client := exponential.NewClient(cfg)

		var entries []model.TimelineEntry

		if len(args) == 1 {
			issue, _, archived, err := client.FindIssue(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			if archived {
				fmt.Println("Note: This issue is archived.")
			}

			client.FillLocalBranchStats(issue)

			for _, evt := range issue.Events {
				if !exponential.IsMeaningfulActivityEvent(evt) {
					continue
				}
				entries = append(entries, model.TimelineEntry{
					Kind:       "issue_event",
					Timestamp:  evt.CreatedAt,
					IssueID:    evt.ID,
					EventType:  string(evt.Type),
					Payload:    evt.Payload,
					CreatedBy:  evt.CreatedBy,
					OnBehalfOf: evt.OnBehalfOf,
					Source:     evt.Source,
				})
			}

			if issue.BranchStats != nil && issue.BranchStats.Branch != "" {
				base := exponential.DefaultBranch()
				commits := exponential.ListBranchCommitsDetailed(issue.BranchStats.Branch, base)
				for _, c := range commits {
					ts, _ := time.Parse(time.RFC3339, c.Date)
					entries = append(entries, model.TimelineEntry{
						Kind:      "commit",
						Timestamp: ts,
						IssueID:   issue.ID,
						SHA:       c.SHA,
						Message:   c.Message,
						Author:    c.Author,
						Branch:    issue.BranchStats.Branch,
					})
				}
			}

			sort.SliceStable(entries, func(i, j int) bool {
				return entries[i].Timestamp.Before(entries[j].Timestamp)
			})
		} else {
			issues, err := client.ListIssues(exponential.FilterOptions{})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			var allEvents []model.Event
			issueMap := make(map[string]*model.Issue, len(issues))
			for _, iss := range issues {
				allEvents = append(allEvents, iss.Events...)
				issueMap[iss.ID] = iss
			}

			entries = exponential.BuildTimeline(allEvents, issueMap, historyLimit, "")
		}

		if historySince != "" {
			d, err := parseDuration(historySince)
			if err != nil {
				fmt.Printf("Error: invalid --since value %q: %v\n", historySince, err)
				os.Exit(1)
			}
			cutoff := time.Now().Add(-d)
			filtered := entries[:0]
			for _, e := range entries {
				if !e.Timestamp.Before(cutoff) {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}

		// Default: newest-first. --reverse: oldest-first.
		// BuildTimeline returns newest-first; single-issue sort is oldest-first.
		if len(args) == 1 && !historyReverse {
			// single-issue: flip to newest-first
			for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
				entries[i], entries[j] = entries[j], entries[i]
			}
		} else if len(args) == 0 && historyReverse {
			// global: flip to oldest-first
			for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}

		if historyJSON {
			ui.RenderTimelineJSON(entries, os.Stdout)
			return
		}

		termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || termWidth <= 0 {
			termWidth = 100
		}
		termWidth -= 4
		if termWidth > 116 {
			termWidth = 116
		}

		w := pagerWriter()
		ui.RenderTimeline(w, entries, termWidth)
		closePager(w)
	},
}

func pagerWriter() io.WriteCloser {
	if historyNoPager || historyJSON || !isatty.IsTerminal(os.Stdout.Fd()) {
		return nopWriteCloser{os.Stdout}
	}

	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}

	cmd := exec.Command(pager, "-R")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	pw, err := cmd.StdinPipe()
	if err != nil {
		return nopWriteCloser{os.Stdout}
	}

	if err := cmd.Start(); err != nil {
		return nopWriteCloser{os.Stdout}
	}

	return &pagerPipe{pipe: pw, cmd: cmd}
}

func closePager(w io.WriteCloser) {
	w.Close()
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

type pagerPipe struct {
	pipe io.WriteCloser
	cmd  *exec.Cmd
}

func (p *pagerPipe) Write(b []byte) (int, error) { return p.pipe.Write(b) }

func (p *pagerPipe) Close() error {
	p.pipe.Close()
	return p.cmd.Wait()
}

func init() {
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "Output events as JSONL (one JSON object per line)")
	historyCmd.Flags().BoolVar(&historyReverse, "reverse", false, "Show oldest events first (chronological)")
	historyCmd.Flags().StringVar(&historySince, "since", "", "Show events within a duration (e.g. 24h, 7d, 2w)")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 50, "Maximum events to show (global mode only)")
	historyCmd.Flags().BoolVar(&historyNoPager, "no-pager", false, "Do not pipe output through a pager")
	rootCmd.AddCommand(historyCmd)
}
