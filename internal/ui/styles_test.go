package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/exponential/internal/model"
)

func TestLabelColorWithConfig(t *testing.T) {
	SetLabelColors(map[string]string{
		"bug":     "#eb5757",
		"feature": "#b36cd9",
		"epic":    "#5e6ad2",
	})
	defer SetLabelColors(nil)

	tests := []struct {
		label    string
		expected lipgloss.Color
	}{
		{"bug", lipgloss.Color("#eb5757")},
		{"feature", lipgloss.Color("#b36cd9")},
		{"epic", lipgloss.Color("#5e6ad2")},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := LabelColor(tt.label)
			if got != tt.expected {
				t.Errorf("LabelColor(%q) = %v, want %v", tt.label, got, tt.expected)
			}
		})
	}
}

func TestLabelColorCaseInsensitiveFallback(t *testing.T) {
	SetLabelColors(map[string]string{
		"bug": "#eb5757",
	})
	defer SetLabelColors(nil)

	got := LabelColor("Bug")
	if got != lipgloss.Color("#eb5757") {
		t.Errorf("LabelColor(\"Bug\") = %v, want #eb5757 (case-insensitive fallback)", got)
	}
}

func TestLabelColorHashFallback(t *testing.T) {
	SetLabelColors(nil)

	a := LabelColor("custom-label")
	b := LabelColor("custom-label")
	if a != b {
		t.Errorf("LabelColor should be deterministic, got %v and %v", a, b)
	}

	c := LabelColor("other-label")
	// Just verify it returns a valid color from the palette
	found := false
	for _, lc := range labelColors {
		if c == lc {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("LabelColor(\"other-label\") = %v, expected a color from the palette", c)
	}
}

func TestLabelColorConfigPrecedence(t *testing.T) {
	SetLabelColors(map[string]string{
		"bug": "#00ff00",
	})
	defer SetLabelColors(nil)

	got := LabelColor("bug")
	if got != lipgloss.Color("#00ff00") {
		t.Errorf("Config color should take precedence, got %v, want #00ff00", got)
	}
}

func TestStatusIconForIssue(t *testing.T) {
	tests := []struct {
		name     string
		issue    *model.Issue
		wantIcon string
	}{
		{
			name:     "inferred doing shows distinct icon",
			issue:    &model.Issue{Status: model.StatusDoing, InferredStatus: true},
			wantIcon: "◉",
		},
		{
			name:     "explicit doing shows standard icon",
			issue:    &model.Issue{Status: model.StatusDoing, InferredStatus: false},
			wantIcon: "●",
		},
		{
			name:     "inferred flag on non-doing status is ignored",
			issue:    &model.Issue{Status: model.StatusPlanned, InferredStatus: true},
			wantIcon: StatusIcon(model.StatusPlanned),
		},
		{
			name:     "backlog uses standard icon",
			issue:    &model.Issue{Status: model.StatusBacklog},
			wantIcon: StatusIcon(model.StatusBacklog),
		},
		{
			name:     "done uses standard icon",
			issue:    &model.Issue{Status: model.StatusDone},
			wantIcon: StatusIcon(model.StatusDone),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusIconForIssue(tt.issue)
			if got != tt.wantIcon {
				t.Errorf("StatusIconForIssue() = %q, want %q", got, tt.wantIcon)
			}
		})
	}
}
