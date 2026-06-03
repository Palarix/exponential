package beats

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Fix login flow", "fix-login-flow"},
		{"special chars", "Fix: the bug! (v2)", "fix-the-bug-v2"},
		{"multiple spaces", "Fix   the   bug", "fix-the-bug"},
		{"leading trailing", "  Fix bug  ", "fix-bug"},
		{"numbers", "Issue 42 fix", "issue-42-fix"},
		{"empty", "", ""},
		{"all special", "!@#$%^&*()", ""},
		{"unicode", "Fix Umlaut Aeoeue", "fix-umlaut-aeoeue"},
		{"hyphens preserved", "fix-the-bug", "fix-the-bug"},
		{
			"truncation",
			"This is a very long title that should be truncated at a reasonable boundary",
			"this-is-a-very-long-title-that-should-be",
		},
		{
			"truncation on boundary",
			"Abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij extra",
			"abcdefghij-abcdefghij-abcdefghij-abcdefghij",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.in)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
