package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type PulseData struct {
	ProjectName   string
	VelocityPts   int
	VelocityDelta int
	CycleTimeHrs  float64
	CycleCount    int
	LeadTimeHrs   float64
	LeadCount     int
	WIPTotal      int
	WIPStale      int
	BlockersTotal int
	BlockersOldest int
	ThroughputWk  int
	ThroughputDelta int
}

func formatDuration(hrs float64) string {
	if hrs < 1 {
		return fmt.Sprintf("%dm", int(hrs*60))
	}
	if hrs < 24 {
		return fmt.Sprintf("%.1fh", hrs)
	}
	return fmt.Sprintf("%.1fd", hrs/24)
}

func RenderPulse(d PulseData, width int) string {
	green := lipgloss.NewStyle().Foreground(DoingColor)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107"))
	red := lipgloss.NewStyle().Foreground(BlockedColor)
	muted := lipgloss.NewStyle().Foreground(MutedColor)
	bold := lipgloss.NewStyle().Bold(true).Foreground(WhiteColor)

	type row struct {
		label string
		value string
	}

	// Velocity
	velStyle := green
	deltaStr := ""
	if d.VelocityDelta > 0 {
		deltaStr = muted.Render(fmt.Sprintf(" (+%d)", d.VelocityDelta))
	} else if d.VelocityDelta < 0 {
		velStyle = yellow
		deltaStr = yellow.Render(fmt.Sprintf(" (%d)", d.VelocityDelta))
	}
	velValue := velStyle.Render(fmt.Sprintf("%d pts/wk", d.VelocityPts)) + deltaStr

	// Cycle time
	ctValue := muted.Render("—")
	if d.CycleCount > 0 {
		ctValue = green.Render(formatDuration(d.CycleTimeHrs)) +
			muted.Render(fmt.Sprintf(" median (n=%d)", d.CycleCount))
	}

	// Lead time
	ltValue := muted.Render("—")
	if d.LeadCount > 0 {
		ltValue = green.Render(formatDuration(d.LeadTimeHrs)) +
			muted.Render(fmt.Sprintf(" median (n=%d)", d.LeadCount))
	}

	// WIP
	wipStyle := green
	wipParts := []string{fmt.Sprintf("%d active", d.WIPTotal)}
	if d.WIPStale > 0 {
		wipStyle = yellow
		wipParts = append(wipParts, yellow.Render(fmt.Sprintf("%d stale", d.WIPStale)))
	}
	wipValue := wipStyle.Render(wipParts[0])
	if len(wipParts) > 1 {
		wipValue += muted.Render(", ") + wipParts[1]
	}

	// Blockers
	blkValue := green.Render("0")
	if d.BlockersTotal > 0 {
		blkValue = red.Render(fmt.Sprintf("%d", d.BlockersTotal))
		if d.BlockersOldest > 0 {
			blkValue += muted.Render(fmt.Sprintf(" (oldest %dd)", d.BlockersOldest))
		}
	}

	// Throughput
	tpValue := green.Render(fmt.Sprintf("%d issues/wk", d.ThroughputWk))
	if d.ThroughputDelta != 0 {
		sign := "+"
		if d.ThroughputDelta < 0 {
			sign = ""
		}
		tpValue += muted.Render(fmt.Sprintf(" (%s%d)", sign, d.ThroughputDelta))
	}

	rows := []row{
		{"Velocity", velValue},
		{"Cycle Time", ctValue},
		{"Lead Time", ltValue},
		{"WIP", wipValue},
		{"Blockers", blkValue},
		{"Throughput", tpValue},
	}

	maxLabel := 0
	for _, r := range rows {
		if len(r.label) > maxLabel {
			maxLabel = len(r.label)
		}
	}

	var lines []string
	for _, r := range rows {
		padding := strings.Repeat(" ", maxLabel-len(r.label))
		lines = append(lines, fmt.Sprintf("  %s%s  %s",
			bold.Render(r.label+":"), padding, r.value))
	}

	content := strings.Join(lines, "\n")

	maxWidth := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxWidth {
			maxWidth = w
		}
	}
	boxWidth := maxWidth + 4
	if boxWidth > width {
		boxWidth = width
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Padding(0, 1).
		Width(boxWidth)

	title := d.ProjectName
	if title == "" {
		title = "Project"
	}
	header := bold.Render("  " + title + " Pulse")

	return "\n" + header + "\n" + box.Render(content) + "\n"
}
