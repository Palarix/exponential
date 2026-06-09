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

var configLabelColors map[string]string

// SetLabelColors sets the label color map from config.
// Call this once after loading config to enable config-driven label colors in CLI output.
func SetLabelColors(labels map[string]string) {
	configLabelColors = labels
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

// StatusIconForIssue returns the status icon, using a distinct icon for
// issues whose DOING status was inferred from a remote git branch.
func StatusIconForIssue(issue *model.Issue) string {
	if issue.InferredStatus && issue.Status == model.StatusDoing {
		return "◉"
	}
	return StatusIcon(issue.Status)
}

// FormatBranchStats renders a compact branch stats annotation like
// "[3 commits, 7 files, +142 -38]". Returns empty string if stats is nil
// or the branch has no commits.
func FormatBranchStats(stats *model.BranchStats) string {
	if stats == nil || stats.Commits == 0 {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(MutedColor)
	commitWord := "commits"
	if stats.Commits == 1 {
		commitWord = "commit"
	}
	fileWord := "files"
	if stats.FilesChanged == 1 {
		fileWord = "file"
	}
	return style.Render(fmt.Sprintf("[%d %s, %d %s, +%d -%d]",
		stats.Commits, commitWord,
		stats.FilesChanged, fileWord,
		stats.Insertions, stats.Deletions))
}

// LabelColor returns a color for a label, checking config colors first,
// then falling back to a deterministic hash-based color.
func LabelColor(label string) lipgloss.Color {
	if configLabelColors != nil {
		if c, ok := configLabelColors[label]; ok {
			return lipgloss.Color(c)
		}
		if c, ok := configLabelColors[strings.ToLower(label)]; ok {
			return lipgloss.Color(c)
		}
	}
	hash := uint32(0)
	for _, r := range label {
		hash = hash*31 + uint32(r)
	}
	return labelColors[hash%uint32(len(labelColors))]
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
