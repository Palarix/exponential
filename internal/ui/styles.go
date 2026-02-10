package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kuyio/beats/internal/model"
	"golang.org/x/term"
)

// --- Color Constants ---
var (
	BacklogColor = lipgloss.Color("#6c757d") // Gray
	PlannedColor = lipgloss.Color("#ffffff") // White
	DoingColor   = lipgloss.Color("#198754") // Green
	BlockedColor = lipgloss.Color("#dc3545") // Red
	DoneColor    = lipgloss.Color("#adb5bd") // Light Gray

	AccentColor = lipgloss.Color("#6f42c1") // Purple
	MutedColor  = lipgloss.Color("#6c757d") // Gray
	WhiteColor  = lipgloss.Color("#ffffff")
)

// --- Predefined Label Colors ---
var labelColors = []lipgloss.Color{
	lipgloss.Color("#e91e63"), // Pink
	lipgloss.Color("#9c27b0"), // Purple
	lipgloss.Color("#673ab7"), // Deep Purple
	lipgloss.Color("#3f51b5"), // Indigo
	lipgloss.Color("#2196f3"), // Blue
	lipgloss.Color("#009688"), // Teal
	lipgloss.Color("#4caf50"), // Green
	lipgloss.Color("#ff9800"), // Orange
	lipgloss.Color("#ff5722"), // Deep Orange
	lipgloss.Color("#795548"), // Brown
}

// --- Base Styles ---
var (
	WhiteStyle  = lipgloss.NewStyle().Foreground(WhiteColor)
	AccentStyle = lipgloss.NewStyle().Foreground(AccentColor)
	MutedStyle  = lipgloss.NewStyle().Foreground(MutedColor)
	BoldStyle   = lipgloss.NewStyle().Bold(true)
)

// StatusStyle returns the style for a given status.
func StatusStyle(status model.IssueStatus) lipgloss.Style {
	switch status {
	case model.StatusBacklog:
		return lipgloss.NewStyle().Foreground(BacklogColor)
	case model.StatusPlanned:
		return lipgloss.NewStyle().Foreground(PlannedColor)
	case model.StatusDoing:
		return lipgloss.NewStyle().Foreground(DoingColor).Bold(true)
	case model.StatusBlocked:
		return lipgloss.NewStyle().Foreground(BlockedColor).Bold(true)
	case model.StatusDone:
		return lipgloss.NewStyle().Foreground(DoneColor).Strikethrough(true)
	default:
		return lipgloss.NewStyle()
	}
}

// StatusIcon returns the icon for a given status.
func StatusIcon(status model.IssueStatus) string {
	switch status {
	case model.StatusBacklog:
		return "󱥸"
	case model.StatusPlanned:
		return "○"
	case model.StatusDoing:
		return "●"
	case model.StatusBlocked:
		return "⊘"
	case model.StatusDone:
		return "✓"
	default:
		return " "
	}
}

// LabelColor returns a consistent color for a label based on its name.
func LabelColor(label string) lipgloss.Color {
	l := strings.ToLower(label)
	if strings.Contains(l, "bug") {
		return BlockedColor // Red
	}
	if strings.Contains(l, "feature") {
		return PlannedColor // Blue
	}
	if strings.Contains(l, "epic") {
		return AccentColor // Purple
	}
	// Default to white for everything else
	return WhiteColor
}

// FormatLabels renders a list of labels as styled tags.
func FormatLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	var parts []string
	for _, l := range labels {
		color := LabelColor(l)
		style := lipgloss.NewStyle().Foreground(color)
		parts = append(parts, style.Render(l))
	}
	return strings.Join(parts, " ")
}

// FormatLabelsBadge renders a list of labels as [label] badges.
func FormatLabelsBadge(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	var parts []string
	for _, l := range labels {
		color := LabelColor(l)
		style := lipgloss.NewStyle().Foreground(color)
		parts = append(parts, style.Render("["+l+"]"))
	}
	return strings.Join(parts, " ")
}

// --- Table helpers ---

// WarningStyle is used for warning messages.
var WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107")).Bold(true)

// RenderCell renders a table cell padded to a given width.
func RenderCell(content string, width int, style lipgloss.Style) string {
	rendered := style.Render(content)
	visibleLen := lipgloss.Width(rendered)
	padding := width - visibleLen
	if padding > 0 {
		return rendered + strings.Repeat(" ", padding)
	}
	return rendered
}

// Truncate truncates a string to a given width.
func Truncate(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
}

// ExtractEmail extracts the email from a "Name <email>" formatted string.
func ExtractEmail(user string) string {
	start := strings.Index(user, "<")
	end := strings.Index(user, ">")
	if start >= 0 && end > start {
		return user[start+1 : end]
	}
	return ""
}

// ExtractName extracts the name from a "Name <email>" formatted string.
func ExtractName(user string) string {
	idx := strings.Index(user, "<")
	if idx > 0 {
		return strings.TrimSpace(user[:idx])
	}
	return user
}

// TerminalWidth returns the current terminal width with a default fallback.
func TerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		width = 80
	}
	return width
}

// ScreenWidth is an alias for TerminalWidth.
func ScreenWidth() int {
	return TerminalWidth()
}

// FormatID formats an issue ID for display.
func FormatID(id string, maxWidth int) string {
	return Truncate(id, maxWidth)
}

// Indent returns a string with the given indentation level.
func Indent(level int) string {
	return strings.Repeat("  ", level)
}

// RenderHeader renders a section header.
func RenderHeader(title string) string {
	return fmt.Sprintf("\n%s\n%s\n",
		BoldStyle.Render(title),
		MutedStyle.Render(strings.Repeat("─", lipgloss.Width(title))))
}

// --- Doctor UI Helpers ---

var (
	ErrorPrefix = "✗"
	OKPrefix    = "✓"
	NotePrefix  = "ℹ"
)

// Stylize applies basic styling to a string for terminal output.
func Stylize(s string) string {
	return s
}
