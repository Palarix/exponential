package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kuyio/beats/internal/model"
)

// StatusIcons maps issue status to display icon
var StatusIcons = map[model.IssueStatus]string{
	model.StatusBacklog: "•",
	model.StatusPlanned: "●",
	model.StatusDoing:   "⋯",
	model.StatusBlocked: "x",
	model.StatusDone:    "✔",
}

// StatusIconsStr provides string-keyed version for show.go compatibility
var StatusIconsStr = map[string]string{
	"BACKLOG": "•",
	"PLANNED": "●",
	"DOING":   "⋯",
	"BLOCKED": "x",
	"DONE":    "✔",
}

// StatusStyles maps issue status to lipgloss style
var StatusStyles = map[model.IssueStatus]lipgloss.Style{
	model.StatusBacklog: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#626262"}),
	model.StatusPlanned: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#626262"}),
	model.StatusDoing:   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1524ffff", Dark: "#337effff"}),
	model.StatusBlocked: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#b30000ff", Dark: "#ff0000ff"}),
	model.StatusDone:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#009938ff", Dark: "#00cb55ff"}),
}

// StatusStylesStr provides string-keyed version for show.go compatibility
var StatusStylesStr = map[string]lipgloss.Style{
	"BACKLOG": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#626262"}),
	"PLANNED": lipgloss.NewStyle().Foreground(lipgloss.Color("67")),
	"DOING":   lipgloss.NewStyle().Foreground(lipgloss.Color("172")),
	"BLOCKED": lipgloss.NewStyle().Foreground(lipgloss.Color("160")),
	"DONE":    lipgloss.NewStyle().Foreground(lipgloss.Color("64")),
}

// Type styles
var (
	RedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	BlueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	PurpleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	WhiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	GreenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	YellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	StrikeStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("255"))
)

// TypeStyle returns the appropriate style for an issue kind
func TypeStyle(kind string) lipgloss.Style {
	switch kind {
	case "BUG":
		return RedStyle
	case "EPIC":
		return PurpleStyle
	case "FEATURE":
		return BlueStyle
	default:
		return WhiteStyle
	}
}

// FormatKindTag returns an abbreviated type tag like [T], [E], [B]
func FormatKindTag(kind string) string {
	if len(kind) > 0 {
		return fmt.Sprintf("[%c]", kind[0])
	}
	return ""
}

// RenderCell renders a cell with truncation and padding
func RenderCell(content string, style lipgloss.Style, width int) string {
	// Calculate max content width (width - 2 for padding)
	maxW := width - 2
	if maxW < 0 {
		maxW = 0
	}

	// Truncate if necessary (rune-based)
	runes := []rune(content)
	if len(runes) > maxW {
		content = string(runes[:maxW-1]) + "…"
	}

	return style.Width(width).Padding(0, 1).Render(content)
}

// ExtractEmail extracts email from a "Name <email>" format string
func ExtractEmail(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start != -1 && end != -1 && start < end {
		return s[start+1 : end]
	}
	return ""
}

// FormatByString extracts a display-friendly name from CreatedBy field
func FormatByString(createdBy string) string {
	if start := strings.Index(createdBy, "<"); start != -1 {
		if end := strings.LastIndex(createdBy, ">"); end != -1 && start < end {
			return createdBy[start+1 : end]
		}
	}
	return createdBy
}
