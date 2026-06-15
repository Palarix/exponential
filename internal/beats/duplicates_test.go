package beats

import (
	"testing"

	"github.com/palarix/beats/internal/model"
)

func TestCheckDuplicates_FindsSimilar(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.AddIssue(model.CreatePayload{Title: "Fix authentication bug in login"})
	tr.AddIssue(model.CreatePayload{Title: "Unrelated feature"})

	dups, err := tr.CheckDuplicates("Fix authentication bug")
	if err != nil {
		t.Fatal(err)
	}
	if len(dups) != 1 {
		t.Errorf("expected 1 duplicate, got %d", len(dups))
	}
}

func TestCheckDuplicates_NoMatch(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.AddIssue(model.CreatePayload{Title: "Fix authentication bug"})

	dups, err := tr.CheckDuplicates("Add pagination to dashboard")
	if err != nil {
		t.Fatal(err)
	}
	if len(dups) != 0 {
		t.Errorf("expected 0 duplicates, got %d", len(dups))
	}
}

func TestCheckDuplicates_ExactMatch(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.AddIssue(model.CreatePayload{Title: "Add search feature"})

	dups, err := tr.CheckDuplicates("Add search feature")
	if err != nil {
		t.Fatal(err)
	}
	if len(dups) != 1 {
		t.Errorf("expected 1 duplicate for exact match, got %d", len(dups))
	}
}

func TestCheckDuplicates_CaseInsensitive(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.AddIssue(model.CreatePayload{Title: "Fix Authentication Bug"})

	dups, err := tr.CheckDuplicates("fix authentication bug")
	if err != nil {
		t.Fatal(err)
	}
	if len(dups) != 1 {
		t.Errorf("match should be case-insensitive, got %d", len(dups))
	}
}

func TestCheckDuplicates_Empty(t *testing.T) {
	tr := setupLocalTransport(t)
	dups, err := tr.CheckDuplicates("anything")
	if err != nil {
		t.Fatal(err)
	}
	if len(dups) != 0 {
		t.Errorf("expected 0 on empty project, got %d", len(dups))
	}
}

func TestTokenize(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"Fix authentication bug", 3},
		{"hello, world!", 2},
		{"  spaced  out  ", 2},
		{"(parenthetical)", 1},
		{"", 0},
	}
	for _, tc := range cases {
		tokens := tokenize(tc.input)
		if len(tokens) != tc.want {
			t.Errorf("tokenize(%q) = %d tokens, want %d", tc.input, len(tokens), tc.want)
		}
	}
}
