package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/exponential/internal/model"
)

// RationaleHit is the UI-facing data for a single search result.
type RationaleHit struct {
	IssueID      string
	Title        string
	Status       string
	Labels       []string
	Document     string
	Fragment     string
	Score        float64
	ArtifactPath string
}

// RationaleData is the input for rendering rationale search results.
type RationaleData struct {
	Results      []RationaleHit
	Query        string
	TotalMatches int
}

var (
	specColor        = lipgloss.Color("#61afef") // Blue
	walkthroughColor = lipgloss.Color("#e5c07b") // Gold
	fragmentBorder   = lipgloss.Color("#3e4451") // Dark gray
)

func docStyle(docType string) lipgloss.Style {
	if docType == "spec" {
		return lipgloss.NewStyle().Foreground(specColor)
	}
	return lipgloss.NewStyle().Foreground(walkthroughColor)
}

// RenderRationale renders search results with colors, labels, fragments, and file paths.
func RenderRationale(data RationaleData, termWidth int) string {
	var sb strings.Builder

	if len(data.Results) == 0 {
		sb.WriteString(MutedStyle.Render(fmt.Sprintf("No results for %q", data.Query)))
		sb.WriteString("\n")
		return sb.String()
	}

	// Header
	sb.WriteString(fmt.Sprintf("Showing top %s results for %s\n\n",
		BoldStyle.Render(fmt.Sprintf("%d", len(data.Results))),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107")).Render(fmt.Sprintf("'%s'", data.Query))))

	fragWidth := termWidth - 8
	if fragWidth < 40 {
		fragWidth = 40
	}
	if fragWidth > 90 {
		fragWidth = 90
	}

	fragBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(fragmentBorder).
		Padding(0, 1).
		Width(fragWidth)

	for i, r := range data.Results {
		renderHit(&sb, r, fragBox)
		if i < len(data.Results)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

var linkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#61afef"))

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func renderHit(sb *strings.Builder, r RationaleHit, fragBox lipgloss.Style) {
	ds := docStyle(r.Document)

	// Line 1: Score · Artifact Type
	sb.WriteString(fmt.Sprintf("%s %s %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107")).Render(fmt.Sprintf("%.2f", r.Score)),
		MutedStyle.Render("·"),
		ds.Render(titleCase(r.Document))))

	// Line 2: Issue ID · Title · Labels · Status
	status := model.IssueStatus(r.Status)
	stStyle := StatusStyle(status)
	stIcon := StatusIcon(status)

	metaParts := []string{
		AccentStyle.Render(r.IssueID),
		BoldStyle.Render(r.Title),
	}
	if len(r.Labels) > 0 {
		metaParts = append(metaParts, FormatLabelsBadge(r.Labels))
	}
	metaParts = append(metaParts, stStyle.Render(fmt.Sprintf("%s %s", stIcon, r.Status)))
	sb.WriteString(fmt.Sprintf("%s\n", strings.Join(metaParts, MutedStyle.Render(" · "))))

	// Line 3: artifact file path (blue link)
	sb.WriteString(fmt.Sprintf("%s\n", linkStyle.Render(r.ArtifactPath)))

	// Fragment in a bordered box
	if r.Fragment != "" {
		frag := wrapFragment(r.Fragment, fragBox.GetWidth()-4)
		sb.WriteString(fragBox.Render(MutedStyle.Render(frag)))
		sb.WriteString("\n")
	}
}

func wrapFragment(text string, width int) string {
	if width <= 0 {
		return text
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		for len(paragraph) > width {
			breakAt := strings.LastIndexByte(paragraph[:width], ' ')
			if breakAt <= 0 {
				breakAt = width
			}
			lines = append(lines, paragraph[:breakAt])
			paragraph = strings.TrimSpace(paragraph[breakAt:])
		}
		if paragraph != "" {
			lines = append(lines, paragraph)
		}
	}
	return strings.Join(lines, "\n")
}
