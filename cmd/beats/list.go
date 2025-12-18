package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/spf13/cobra"
)

var listStatusFlag string
var listAllFlag bool
var listSinceFlag string
var listBeforeFlag string
var listMatchFlag string

func parseTimeFilter(input string) (time.Time, error) {
	// Try parsing as duration (relative to now)
	if d, err := time.ParseDuration(input); err == nil {
		return time.Now().Add(-d), nil
	}
	// Try parsing as date (End of day assumption for "since" might be wrong, but simple date usually means 00:00)
	// For "since 2024-01-01", we want from 2024-01-01 00:00:00.
	if t, err := time.Parse("2006-01-02", input); err == nil {
		return t, nil
	}
	// Try RFC3339
	return time.Parse(time.RFC3339, input)
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List issues",
	Run: func(cmd *cobra.Command, args []string) {
		events, err := storage.ReadEvents()
		if err != nil {
			fmt.Printf("Error reading events: %v\n", err)
			os.Exit(1)
		}

		issuesMap := model.ProjectIssues(events)
		issues := model.SortIssues(issuesMap)

		// Parse status flags
		validStatuses := make(map[string]bool)
		if listStatusFlag != "" {
			parts := strings.Split(listStatusFlag, ",")
			for _, p := range parts {
				validStatuses[strings.TrimSpace(p)] = true
			}
		}

		// Parse time flags
		var sinceTime, beforeTime time.Time
		if listSinceFlag != "" {
			t, err := parseTimeFilter(listSinceFlag)
			if err != nil {
				fmt.Printf("Error parsing --since: %v\n", err)
				os.Exit(1)
			}
			sinceTime = t
		}
		if listBeforeFlag != "" {
			t, err := parseTimeFilter(listBeforeFlag)
			if err != nil {
				fmt.Printf("Error parsing --before: %v\n", err)
				os.Exit(1)
			}
			beforeTime = t
		}

		// Prepare match query
		matchQuery := ""
		if listMatchFlag != "" {
			matchQuery = strings.ToLower(listMatchFlag)
		}

		// Column Config
		cols := []struct {
			Title string
			Width int
		}{
			{"ID", 14},
			{"Type", 10},
			{"Status", 12},
			{"Title", 60},
			{"Parent", 16},
			{"Created", 20},
			{"By", 25},
		}

		// Header Style
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("240"))

		// Render Header
		var headerCells []string
		for _, col := range cols {
			// Pad the header cell. Width includes padding.
			cell := lipgloss.NewStyle().Width(col.Width).Padding(0, 1).Render(col.Title)
			headerCells = append(headerCells, cell)
		}
		fmt.Println(headerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, headerCells...)))

		// Row Styles
		mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		whiteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		blueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
		purpleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
		strikeStyle := lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("255"))

		for _, i := range issues {
			// Check against validStatuses if set
			if len(validStatuses) > 0 {
				if !validStatuses[string(i.Status)] {
					continue
				}
			}

			// Time filters
			if !sinceTime.IsZero() && i.CreatedAt.Before(sinceTime) {
				continue
			}
			if !beforeTime.IsZero() && i.CreatedAt.After(beforeTime) {
				continue
			}

			// Search filter
			if matchQuery != "" {
				// Naive match against all fields requested: ID, Type, Status, Title, Parent, CreatedBy
				// Description is not in the list per user request, but simple enough to add later if needed.
				// User asked for: "id, type status, title, parent or created by"
				matchFound := false
				if strings.Contains(strings.ToLower(i.ID), matchQuery) {
					matchFound = true
				} else if strings.Contains(strings.ToLower(i.Kind), matchQuery) {
					matchFound = true
				} else if strings.Contains(strings.ToLower(string(i.Status)), matchQuery) {
					matchFound = true
				} else if strings.Contains(strings.ToLower(i.Title), matchQuery) {
					matchFound = true
				} else if strings.Contains(strings.ToLower(i.ParentID), matchQuery) {
					matchFound = true
				} else if strings.Contains(strings.ToLower(i.CreatedBy), matchQuery) {
					matchFound = true
				}

				if !matchFound {
					continue
				}
			}

			// Hide DONE tasks unless --all is passed or status is explicitly DONE in the filter
			if !listAllFlag && !validStatuses[string(model.StatusDone)] && i.Status == model.StatusDone {
				continue
			}

			relTime := humanize.Time(i.CreatedAt)

			// Format 'By' column to show only email if available
			byStr := i.CreatedBy
			if start := strings.Index(byStr, "<"); start != -1 {
				if end := strings.LastIndex(byStr, ">"); end != -1 && start < end {
					byStr = byStr[start+1 : end]
				}
			}

			// Base logic: Determine base style for the row
			var idS, stS, tiS, paS, crS, byS lipgloss.Style

			switch i.Status {
			case model.StatusBacklog:
				idS, stS, tiS, paS, crS, byS = whiteStyle, mutedStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			case model.StatusDoing:
				idS, stS, tiS, paS, crS, byS = whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			case model.StatusDone:
				idS, stS, tiS, paS, crS, byS = whiteStyle, greenStyle, strikeStyle, whiteStyle, whiteStyle, whiteStyle
			case model.StatusBlocked:
				idS, stS, tiS, paS, crS, byS = whiteStyle, redStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			default:
				idS, stS, tiS, paS, crS, byS = whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			}

			// Determine Type Style (tyS)
			var tyS lipgloss.Style
			switch i.Kind {
			case "BUG":
				tyS = redStyle
			case "EPIC":
				tyS = purpleStyle
			case "FEATURE":
				tyS = blueStyle
			default: // TASK and others
				tyS = whiteStyle
			}

			// Helper to render cell
			renderCell := func(content string, style lipgloss.Style, width int) string {
				// Calculate max content width (width - 2 for padding)
				maxW := width - 2
				if maxW < 0 {
					maxW = 0
				}

				// Truncate if necessary (naive rune-based)
				runes := []rune(content)
				if len(runes) > maxW {
					content = string(runes[:maxW-1]) + "…"
				}

				return style.Width(width).Padding(0, 1).Render(content)
			}

			// Check for ParentID and add visual prefix
			title := i.Title
			if i.ParentID != "" {
				title = " ↳ " + title
			}

			// Render
			c1 := renderCell(i.ID, idS, cols[0].Width)
			c2 := renderCell(i.Kind, tyS, cols[1].Width)
			c3 := renderCell(string(i.Status), stS, cols[2].Width)
			c4 := renderCell(title, tiS, cols[3].Width)
			c5 := renderCell(i.ParentID, paS, cols[4].Width)
			c6 := renderCell(relTime, crS, cols[5].Width)
			c7 := renderCell(byStr, byS, cols[6].Width)

			row := lipgloss.JoinHorizontal(lipgloss.Left, c1, c2, c3, c4, c5, c6, c7)
			fmt.Println(row)
		}
	},
}

func init() {
	listCmd.Flags().StringVar(&listStatusFlag, "status", "", "Filter by status")
	listCmd.Flags().BoolVarP(&listAllFlag, "all", "a", false, "Show all issues (including DONE)")
	listCmd.Flags().StringVar(&listSinceFlag, "since", "", "Show issues created since duration/date (e.g. 24h, 2024-01-01)")
	listCmd.Flags().StringVar(&listBeforeFlag, "before", "", "Show issues created before duration/date")
	listCmd.Flags().StringVarP(&listMatchFlag, "match", "m", "", "Search for string in ID, title, status, etc.")
	rootCmd.AddCommand(listCmd)
}
