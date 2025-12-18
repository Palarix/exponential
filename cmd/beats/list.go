package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	"github.com/spf13/cobra"
)

var listStatusFlag string

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
		strikeStyle := lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("255"))

		for _, i := range issues {
			if listStatusFlag != "" && string(i.Status) != listStatusFlag {
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

			// Render
			c1 := renderCell(i.ID, idS, cols[0].Width)
			c2 := renderCell(i.Kind, idS, cols[1].Width)
			c3 := renderCell(string(i.Status), stS, cols[2].Width)
			c4 := renderCell(i.Title, tiS, cols[3].Width)
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
	rootCmd.AddCommand(listCmd)
}
