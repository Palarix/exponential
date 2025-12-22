package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Column defines a table column
type Column struct {
	Title string
	Width int  // Fixed width or minimum width if Flex is true
	Flex  bool // If true, expands to fill available space
}

// HeaderStyle matches the list command header style
var HeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Border(lipgloss.NormalBorder(), false, false, true, false).
	BorderForeground(lipgloss.Color("240"))

// CalculateColumnWidths helper
func CalculateColumnWidths(totalWidth int, columns []Column) []int {
	usedWidth := 0
	flexCount := 0
	paddingPerCol := 2

	for _, col := range columns {
		usedWidth += col.Width + paddingPerCol
		if col.Flex {
			flexCount++
		}
	}

	remaining := totalWidth - usedWidth
	extra := 0
	if flexCount > 0 && remaining > 0 {
		extra = remaining / flexCount
	}

	widths := make([]int, len(columns))
	for i, col := range columns {
		w := col.Width
		if col.Flex {
			w += extra
		}
		widths[i] = w
	}
	return widths
}
