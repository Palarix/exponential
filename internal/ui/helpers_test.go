package ui

import (
	"testing"

	"github.com/palarix/beats/internal/model"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		s        string
		max      int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"ab", 3, "ab"},
		{"abcdef", 3, "abc"},
		{"abcdef", 2, "ab"},
		{"", 5, ""},
	}
	for _, tc := range cases {
		got := Truncate(tc.s, tc.max)
		if got != tc.expected {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.s, tc.max, got, tc.expected)
		}
	}
}

func TestExtractEmail(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Alice <alice@test.com>", "alice@test.com"},
		{"Bob <bob@example.com>", "bob@example.com"},
		{"no-email", ""},
		{"", ""},
		{"<just@email.com>", "just@email.com"},
	}
	for _, tc := range cases {
		got := ExtractEmail(tc.input)
		if got != tc.want {
			t.Errorf("ExtractEmail(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestExtractName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Alice <alice@test.com>", "Alice"},
		{"Bob Jones <bob@example.com>", "Bob Jones"},
		{"no-email", "no-email"},
		{"", ""},
	}
	for _, tc := range cases {
		got := ExtractName(tc.input)
		if got != tc.want {
			t.Errorf("ExtractName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatBranchStats_Nil(t *testing.T) {
	got := FormatBranchStats(nil)
	if got != "" {
		t.Errorf("nil stats should return empty, got %q", got)
	}
}

func TestFormatBranchStats_ZeroCommits(t *testing.T) {
	got := FormatBranchStats(&model.BranchStats{Commits: 0})
	if got != "" {
		t.Errorf("zero commits should return empty, got %q", got)
	}
}

func TestFormatBranchStats_WithCommits(t *testing.T) {
	got := FormatBranchStats(&model.BranchStats{Commits: 5})
	if got == "" {
		t.Error("should return non-empty for positive commits")
	}
}

func TestFormatLabels_Empty(t *testing.T) {
	got := FormatLabels(nil)
	if got != "" {
		t.Errorf("empty labels should return empty string, got %q", got)
	}
}

func TestFormatLabels_NonEmpty(t *testing.T) {
	got := FormatLabels([]string{"bug", "feature"})
	if got == "" {
		t.Error("should render non-empty for labels")
	}
}

func TestFormatLabelsBadge(t *testing.T) {
	got := FormatLabelsBadge([]string{"bug"})
	if got == "" {
		t.Error("should render non-empty badge")
	}
}

func TestRenderCell(t *testing.T) {
	got := RenderCell("hi", 10, WarningStyle)
	if got == "" {
		t.Error("should render non-empty cell")
	}
}
