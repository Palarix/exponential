package beats

import (
	"strings"
	"testing"

	"github.com/palarix/beats/internal/model"
)

func TestResolveIssue_ExactMatch(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "exact"})

	got, err := tr.GetIssue(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != issue.ID {
		t.Errorf("got %s, want %s", got.ID, issue.ID)
	}
}

func TestResolveIssue_PrefixMatch(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "prefix"})

	shortID := strings.TrimPrefix(issue.ID, "test-")
	got, err := tr.GetIssue(shortID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != issue.ID {
		t.Errorf("got %s, want %s", got.ID, issue.ID)
	}
}

func TestResolveIssue_SubstringMatch(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "substring"})

	hash := strings.TrimPrefix(issue.ID, "test-")
	partial := hash[:3]
	got, err := tr.GetIssue(partial)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != issue.ID {
		t.Errorf("got %s, want %s", got.ID, issue.ID)
	}
}

func TestResolveIssue_NotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	_, err := tr.GetIssue("zzzzzzzzz")
	if err == nil {
		t.Error("expected not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say not found, got: %s", err)
	}
}

func TestResolveIssue_Ambiguous(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.AddIssue(model.CreatePayload{Title: "one"})
	tr.AddIssue(model.CreatePayload{Title: "two"})

	// "test-" is a common substring of all issues
	_, err := tr.GetIssue("test-")
	if err == nil {
		t.Error("expected ambiguous error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error should say ambiguous, got: %s", err)
	}
}
