package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLabelColor(t *testing.T) {
	tests := []struct {
		label    string
		expected lipgloss.Color
	}{
		{"bug", BlockedColor},
		{"Bug", BlockedColor},
		{"critical-bug", BlockedColor},
		{"feature", PlannedColor},
		{"Feature", PlannedColor},
		{"new-feature", PlannedColor},
		{"epic", AccentColor},
		{"Epic", AccentColor},
		{"project-epic", AccentColor},
		{"other", WhiteColor},
		{"documentation", WhiteColor},
		{"wontfix", WhiteColor},
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
