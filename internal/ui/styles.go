package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/beats/internal/model"
)

// Base Styles
var (
	baseStyle = lipgloss.NewStyle().Padding(0, 1)

	// Status Colors (Adaptive)
	StatusBacklogColor = lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#626262"}
	StatusPlannedColor = lipgloss.AdaptiveColor{Light: "#8B4513", Dark: "#A0522D"} // Brownish
	StatusDoingColor   = lipgloss.AdaptiveColor{Light: "#1524ffff", Dark: "#337effff"}
	StatusBlockedColor = lipgloss.AdaptiveColor{Light: "#b30000ff", Dark: "#ff0000ff"}
	StatusDoneColor    = lipgloss.AdaptiveColor{Light: "#009938ff", Dark: "#00cb55ff"}

	// Type Colors
	BugColor     = lipgloss.Color("196")
	EpicColor    = lipgloss.Color("99")
	FeatureColor = lipgloss.Color("39")
	DefaultColor = lipgloss.Color("255")
)

// StatusIcons maps issue status to display icon
var statusIcons = map[model.IssueStatus]string{
	model.StatusBacklog: "•",
	model.StatusPlanned: "●",
	model.StatusDoing:   "⋯",
	model.StatusBlocked: "x",
	model.StatusDone:    "✓",
}

// StatusIcon returns the icon for a given status
func StatusIcon(status model.IssueStatus) string {
	if icon, ok := statusIcons[status]; ok {
		return icon
	}
	return "•"
}

// StatusStyle returns the style for a given status
func StatusStyle(status model.IssueStatus) lipgloss.Style {
	var c lipgloss.TerminalColor
	switch status {
	case model.StatusBacklog:
		c = StatusBacklogColor
	case model.StatusPlanned:
		// Planned was Color("67") in one place and gray in another.
		// Let's pick a distinct color. In old code:
		// StatusIconsStr: PLANNED -> ●
		// StatusStylesStr: PLANNED -> Color("67") (blueish)
		// StatusStyles: PLANNED -> Gray
		// Let's go with Blueish/Cyan distinct from Backlog. 67 is SteelBlue.
		c = lipgloss.Color("67")
	case model.StatusDoing:
		c = StatusDoingColor
	case model.StatusBlocked:
		c = StatusBlockedColor
	case model.StatusDone:
		c = StatusDoneColor
	default:
		c = StatusBacklogColor
	}
	return lipgloss.NewStyle().Foreground(c)
}

// Type styles
var (
	RedStyle    = lipgloss.NewStyle().Foreground(BugColor).Bold(true)
	BlueStyle   = lipgloss.NewStyle().Foreground(FeatureColor).Bold(true)
	PurpleStyle = lipgloss.NewStyle().Foreground(EpicColor).Bold(true)
	WhiteStyle  = lipgloss.NewStyle().Foreground(DefaultColor)
	GreenStyle  = lipgloss.NewStyle().Foreground(StatusDoneColor).Bold(true)
	YellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	CodeStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#ea1188ff", Dark: "#ea1188ff"})
	StrikeStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("240")) // Darker gray for done items

	// Shared prefixes
	OKPrefix    = GreenStyle.Render("[✔]")
	ErrorPrefix = RedStyle.Render("[x]")
	NotePrefix  = YellowStyle.Render("[!]")
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

// Stylize replaces strings in backticks with CodeStyle
// e.g. "Run `beats doctor`" -> "Run " + CodeStyle("beats doctor")
func Stylize(s string) string {
	re := regexp.MustCompile("`([^`]+)`")
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// match includes the backticks, e.g. `beats doctor`
		// Remove them and apply style
		inner := match[1 : len(match)-1]
		return CodeStyle.Render(inner)
	})
}
