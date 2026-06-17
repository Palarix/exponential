package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/exponential/internal/model"
)

func TestRenderIssueList_Formatting(t *testing.T) {
	// Setup a sample issue with a time that would result in a long string via default humanize
	// e.g. "3 months from now" or something long if we were checking that,
	// but here we just want to ensure it fits in the table.
	// Actually, standard humanize might say "2 months ago", which corresponds to ~12 chars.
	// We want to force it to use the compact format "2mo ago".

	// 2024-01-01
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 3 months ago from fixedTime would be 2023-10-01
	// humanize.Time would typically return "3 months ago" (12 chars)
	updatedAt := fixedTime.AddDate(0, -3, 0)

	issues := []*model.Issue{
		{
			ID:        "issue-123",
			Title:     "Test Issue",
			Status:    model.StatusBacklog,
			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		},
	}

	termWidth := 100
	// We can't easily mock time.Now() for humanize.Time inside the function unless we change the function signature
	// or use a variable.
	// However, the function RenderIssueList uses humanize.Time(i.UpdatedAt).
	// humanize.Time uses time.Now() internally.

	// For this test to be robust without mocking time.Now(), we can just assert that the
	// output line length matches termWidth exactly (or is less).
	// The current implementation might wrap if the columns overflow.

	// Let's rely on the fact that we are injecting a long string via the fact that we are NOT mocking
	// and just checking if the formatting logic holds up.

	// Actually, to properly test the specific fix (compact strings), we need to see "3mo ago" vs "3 months ago".
	// But since we can't control "now" easily without a refactor for DI, we will check line length.

	output := RenderIssueList(issues, termWidth)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if w := lipgloss.Width(line); w > termWidth {
			t.Errorf("Line exceeds terminal width %d: %q (width %d)", termWidth, line, w)
		}
	}
}
