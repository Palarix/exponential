package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kuyio/beats/cmd/beats/ui"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

		// Empty line at start of output
		fmt.Println()

		// Prepare match query
		matchQuery := ""
		if listMatchFlag != "" {
			matchQuery = strings.ToLower(listMatchFlag)
		}

		// Detect Terminal Width
		termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || termWidth <= 0 {
			termWidth = 120 // Fallback
		}

		// Apply safety margin and cap for readability
		termWidth -= 4
		if termWidth > 116 {
			termWidth = 116 // Total effective width including padding
		}

		fixedWidth := 14 + 5 + 6 + 15 + 20 + 2*6 // Col widths + padding (1 left, 1 right per column)
		// ID(14), Type(5), Status(6), Updated(15), By(20) = 60 chars content
		// 6 columns * 2 padding = 12 chars padding. Total fixed = 72.

		titleWidth := termWidth - fixedWidth
		if titleWidth < 20 {
			titleWidth = 20
		}
		if titleWidth > 100 {
			titleWidth = 100 // Cap for readability
		}

		// Column Config
		cols := []struct {
			Title string
			Width int
		}{
			{"ID", 14},
			{" ", 5},
			{" ", 6},
			{"Title", titleWidth},
			{"Updated", 15},
			{"Added By", 20},
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

		// Row Styles using shared ui package
		strikeStyle := ui.StrikeStyle

		// Get recent DONE issues using shared utility
		recentDoneIDs := make(map[string]bool)
		if !listAllFlag {
			recentDoneIDs = ui.GetRecentDoneIDs(issues, 3)
		}

		// Use shared status icons and colors from ui package
		statusIcons := ui.StatusIcons
		statusColors := ui.StatusStyles

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
					userEmail := ui.ExtractEmail(currentUser)
					issueEmail := ui.ExtractEmail(i.CreatedBy)
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

			updTime := humanize.CustomRelTime(i.UpdatedAt, time.Now(), "ago", "from now", listMagnitudes)

			// Format 'By' column using shared utility
			byStr := ui.FormatByString(i.CreatedBy)

			// Base logic: Determine base style for the row
			var idS, stS, tiS, upS, byS lipgloss.Style

			switch i.Status {
			case model.StatusBacklog:
				idS, tiS, upS, byS = ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle
			case model.StatusDoing:
				idS, tiS, upS, byS = ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle
			case model.StatusDone:
				idS, tiS, upS, byS = ui.WhiteStyle, strikeStyle, ui.WhiteStyle, ui.WhiteStyle
			case model.StatusBlocked:
				idS, tiS, upS, byS = ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle
			default:
				idS, tiS, upS, byS = ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle, ui.WhiteStyle
			}

			if style, ok := statusColors[i.Status]; ok {
				stS = style
			} else {
				stS = ui.WhiteStyle
			}

			// Prepare Status String with Icon
			// User requested "just show the icons in the ls"
			stStr := string(i.Status) // Fallback
			if icon, ok := statusIcons[i.Status]; ok {
				stStr = fmt.Sprintf(" %s ", icon) // Center with spaces? Or just icon?
				// Width is 3, padding 1.
				// If content is " ● ", width 3.
			}

			// Determine Type Style using shared utility
			tyS := ui.TypeStyle(i.Kind)

			// Abbreviate Kind using shared utility
			kindStr := ui.FormatKindTag(i.Kind)

			// Render cells using shared utility
			renderCell := ui.RenderCell

			// Check for ParentID and add visual prefix
			title := i.Title
			if i.ParentID != "" {
				title = " ↳ " + title
			}

			// Render
			c1 := renderCell(i.ID, idS, cols[0].Width)
			c2 := renderCell(kindStr, tyS, cols[1].Width)
			c3 := renderCell(stStr, stS, cols[2].Width)
			c4 := renderCell(title, tiS, cols[3].Width)
			c7 := renderCell(updTime, upS, cols[4].Width)
			c8 := renderCell(byStr, byS, cols[5].Width)

			row := lipgloss.JoinHorizontal(lipgloss.Left, c1, c2, c3, c4, c7, c8)
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
