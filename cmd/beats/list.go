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
var listMineFlag bool
var listEpicFlag string

func parseTimeFilter(input string) (time.Time, error) {
	// Handle custom suffixes for days (d) and weeks (w)
	if len(input) > 1 {
		lastChar := input[len(input)-1]
		if lastChar == 'd' || lastChar == 'w' {
			valStr := input[:len(input)-1]
			// We can try to convert the prefix to an int/float to verify it's a number
			// But simpler might be to just string replace if we want to rely on ParseDuration validation later
			// However, ParseDuration doesn't take 24*val + "h", we need to compute.

			// Let's assume valid input like "1d", "2w".
			// To be robust, we essentially want to map "1d" -> "24h", "1w" -> "168h"
			// But complex things like "1d2h" are harder with this simple check.
			// Let's stick to simple single unit check for now as requested.

			// Actually simpler: just replace suffix with hour equivalent?
			// "1d" -> "24h" ?? No, 2d -> 48h.

			// So parsing the number is needed.
			// But "1.5d" is valid conceptual duration.

			// Let's write a helper or just do it inline.

			// Try to parse the numeric part
			// We use a small trick: standard ParseDuration supports fractional hours.
			// So we can say: "1d" -> parse "1" -> 1.0 * 24h
			// "1.5w" -> parse "1.5" -> 1.5 * 168h

			// But we need to use strconv or similar.
			// Or we can leverage ParseDuration itself!
			// "1d" is not valid. But if we replace "d" with "h" -> "1h", parse it, get 1 hour, then multiply by 24?
			// YES. "1.5d" -> "1.5h" -> 1.5 hours. 1.5 hours * 24 = 36 hours (1.5 days).
			// This works for simple scalar + unit inputs.

			modifiedInput := valStr + "h"
			if d, err := time.ParseDuration(modifiedInput); err == nil {
				// d is now X hours. We want X days/weeks.
				// Since we parsed it as hours, the value 'd' represents X hours.
				// If unit was 'd', we want X * 24 hours.
				// d is (X * time.Hour). We want (X * 24 * time.Hour).
				// So actualDuration = d * 24 (if days) or d * 24 * 7 (if weeks).

				// However, 'd' is already time.Duration (int64 nanoseconds).
				// So d * 24 works.

				var factor int64
				if lastChar == 'd' {
					factor = 24
				} else {
					factor = 24 * 7
				}

				finalDuration := d * time.Duration(factor)
				return time.Now().Add(-finalDuration), nil
			}
		}
	}

	// Try parsing as standard duration (relative to now)
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
			{"Title", 50},
			{"Parent", 16},
			{"Created", 15},
			{"Updated", 15},
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

		// Find recent DONE issues to always show (unless filtered out by other means)
		recentDoneIDs := make(map[string]bool)
		if !listAllFlag {
			var doneIssues []*model.Issue
			for _, i := range issues {
				if i.Status == model.StatusDone {
					doneIssues = append(doneIssues, i)
				}
			}
			// Sort by UpdatedAt Desc
			// Note: issues are already sorted by CreatedAt in SortIssues, but we want UpdatedAt for "recent"
			// But SortIssues flattens the hierarchy, so we should just sort doneIssues locally.
			// Actually, SortIssues sorts by Root CreatedAt, then children.
			// Let's do a simple sort here.
			// We need to import "sort" if not available? It is not imported in list.go (only fmt, os, strings, time - wait, check imports)
			// list.go imports: fmt, os, strings, time, lipgloss, humanize, model, storage, cobra.
			// Need to add "sort" import? No, I can't add imports with replace_file_content block if not already there easily without seeing top.
			// But I can use bubble sort or just simple loop since it's small, OR I can assume "sort" is needed.
			// Wait, simple manual selection of top 3 is O(N), easier than importing sort if I don't want to touch imports.
			// But N is small.
			// Let's check imports of list.go first? I probably should have checked imports.
			// View file showed imports: fmt, os, strings, time. (See Step 21).
			// So I cannot use `sort` package without adding it to imports.
			// Adding imports with replace_file_content at top is annoying if I'm doing a block in the middle.
			// I'll stick to a simple strategy: Find top 3.

			type simpleIssue struct {
				ID      string
				Updated time.Time
			}
			var candidates []simpleIssue
			for _, i := range doneIssues {
				candidates = append(candidates, simpleIssue{i.ID, i.UpdatedAt})
			}

			// Simple sort or select top 3
			// Selection sort 3 times
			for k := 0; k < 3 && k < len(candidates); k++ {
				maxIdx := k
				for j := k + 1; j < len(candidates); j++ {
					if candidates[j].Updated.After(candidates[maxIdx].Updated) {
						maxIdx = j
					}
				}
				// Swap
				candidates[k], candidates[maxIdx] = candidates[maxIdx], candidates[k]
				recentDoneIDs[candidates[k].ID] = true
			}
		}

		for _, i := range issues {
			// Check against validStatuses if set
			if len(validStatuses) > 0 {
				if !validStatuses[string(i.Status)] {
					continue
				}
			}

			// Time filters
			if !sinceTime.IsZero() && i.UpdatedAt.Before(sinceTime) {
				continue
			}
			if !beforeTime.IsZero() && i.UpdatedAt.After(beforeTime) {
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

			// Mine filter
			if listMineFlag {
				currentUser := getUser()
				// Basic check: CreatedBy contains "Name <email>" or just "Name"
				// getUser returns "Name <email>"
				// Issue.CreatedBy might match that.
				// Let's do a loose containment check for safety, or strict if we are confident.
				// Since getUser() returns exact string stored in CreatedBy (usually), strict might work,
				// but let's be safer with contains since one might have "Name" and other "Name <email>".
				// Actually, add.go stores exactly what getUser() returns.
				// So strict equality or Contains is fine.
				if !strings.Contains(i.CreatedBy, currentUser) && !strings.Contains(currentUser, i.CreatedBy) {
					// Fallback: Check emails
					// If i.CreatedBy has <email>, extract it.
					// If currentUser has <email>, extract it.
					// Compare emails.
					userEmail := extractEmail(currentUser)
					issueEmail := extractEmail(i.CreatedBy)
					if userEmail != "" && issueEmail != "" {
						if userEmail != issueEmail {
							continue
						}
					} else {
						// No emails, fallback to string compare which failed
						continue
					}
				}
			}

			// Epic filter
			if listEpicFlag != "" {
				if i.ParentID != listEpicFlag {
					continue
				}
			}

			// Hide DONE tasks unless:
			// 1. --all is passed
			// 2. Status is explicitly DONE in filter
			// 3. It is one of the recent DONE tasks
			if i.Status == model.StatusDone {
				show := false
				if listAllFlag {
					show = true
				} else if validStatuses[string(model.StatusDone)] {
					show = true
				} else if recentDoneIDs[i.ID] {
					show = true
				}

				if !show {
					continue
				}
			}

			relTime := humanize.Time(i.CreatedAt)
			updTime := humanize.Time(i.UpdatedAt)

			// Format 'By' column to show only email if available
			byStr := i.CreatedBy
			if start := strings.Index(byStr, "<"); start != -1 {
				if end := strings.LastIndex(byStr, ">"); end != -1 && start < end {
					byStr = byStr[start+1 : end]
				}
			}

			// Base logic: Determine base style for the row
			var idS, stS, tiS, paS, crS, upS, byS lipgloss.Style

			switch i.Status {
			case model.StatusBacklog:
				idS, stS, tiS, paS, crS, upS, byS = whiteStyle, mutedStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			case model.StatusDoing:
				idS, stS, tiS, paS, crS, upS, byS = whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			case model.StatusDone:
				idS, stS, tiS, paS, crS, upS, byS = whiteStyle, greenStyle, strikeStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			case model.StatusBlocked:
				idS, stS, tiS, paS, crS, upS, byS = whiteStyle, redStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
			default:
				idS, stS, tiS, paS, crS, upS, byS = whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle, whiteStyle
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
			c7 := renderCell(updTime, upS, cols[6].Width)
			c8 := renderCell(byStr, byS, cols[7].Width)

			row := lipgloss.JoinHorizontal(lipgloss.Left, c1, c2, c3, c4, c5, c6, c7, c8)
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
	listCmd.Flags().BoolVar(&listMineFlag, "mine", false, "Show issues created by current user")
	listCmd.Flags().StringVar(&listEpicFlag, "epic", "", "Filter by child of epic ID")
	rootCmd.AddCommand(listCmd)
}

func extractEmail(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start != -1 && end != -1 && start < end {
		return s[start+1 : end]
	}
	return ""
}
