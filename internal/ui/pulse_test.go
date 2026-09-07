package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderPulse_AllZeros(t *testing.T) {
	out := RenderPulse(PulseData{}, 80)
	if !strings.Contains(out, "Pulse") {
		t.Error("expected header")
	}
	if !strings.Contains(out, "Velocity:") {
		t.Error("expected Velocity row")
	}
	if !strings.Contains(out, "0 pts/wk") {
		t.Error("expected zero velocity")
	}
	if !strings.Contains(out, "Blockers:") {
		t.Error("expected Blockers row")
	}
}

func TestRenderPulse_WithData(t *testing.T) {
	d := PulseData{
		VelocityPts:     42,
		VelocityDelta:   5,
		CycleTimeHrs:    0.5,
		CycleCount:      10,
		LeadTimeHrs:     4.0,
		LeadCount:       8,
		WIPTotal:        3,
		WIPStale:        1,
		BlockersTotal:   2,
		BlockersOldest:  3,
		ThroughputWk:    7,
		ThroughputDelta: -2,
	}

	out := RenderPulse(d, 120)
	if !strings.Contains(out, "42 pts/wk") {
		t.Errorf("expected velocity 42, got:\n%s", out)
	}
	if !strings.Contains(out, "(+5)") {
		t.Error("expected positive delta")
	}
	if !strings.Contains(out, "30m") {
		t.Errorf("expected 30m cycle time for 0.5hrs, got:\n%s", out)
	}
	if !strings.Contains(out, "4.0h") {
		t.Errorf("expected 4.0h lead time, got:\n%s", out)
	}
	if !strings.Contains(out, "3 active") {
		t.Error("expected 3 active WIP")
	}
	if !strings.Contains(out, "1 stale") {
		t.Error("expected 1 stale WIP")
	}
	if !strings.Contains(out, "oldest 3d") {
		t.Error("expected blocker age")
	}
	if !strings.Contains(out, "7 issues/wk") {
		t.Error("expected throughput")
	}
	if !strings.Contains(out, "(-2)") {
		t.Error("expected negative throughput delta")
	}
}

func TestRenderPulse_NegativeVelocityDelta(t *testing.T) {
	d := PulseData{
		VelocityPts:   10,
		VelocityDelta: -3,
	}
	out := RenderPulse(d, 80)
	if !strings.Contains(out, "(-3)") {
		t.Errorf("expected negative delta, got:\n%s", out)
	}
}

func TestRenderPulse_LinesWithinWidth(t *testing.T) {
	d := PulseData{
		VelocityPts:    218,
		VelocityDelta:  12,
		CycleTimeHrs:   0.2,
		CycleCount:     50,
		LeadTimeHrs:    48.0,
		LeadCount:      40,
		WIPTotal:       5,
		WIPStale:       2,
		BlockersTotal:  1,
		BlockersOldest: 7,
		ThroughputWk:   15,
		ThroughputDelta: 3,
	}
	width := 80
	out := RenderPulse(d, width)
	for _, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("line exceeds width %d (got %d): %q", width, w, line)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		hrs  float64
		want string
	}{
		{0.2, "12m"},
		{0.5, "30m"},
		{1.5, "1.5h"},
		{23.9, "23.9h"},
		{24.0, "1.0d"},
		{48.0, "2.0d"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.hrs)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.hrs, got, tt.want)
		}
	}
}
