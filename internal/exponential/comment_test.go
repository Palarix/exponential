package exponential

import (
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestAddComment_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "commentable"})

	err := tr.AddComment(issue.ID, "a comment")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if len(issues[issue.ID].Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(issues[issue.ID].Comments))
	}
	if issues[issue.ID].Comments[0].Text != "a comment" {
		t.Errorf("comment text = %q", issues[issue.ID].Comments[0].Text)
	}
}

func TestAddComment_MultipleComments(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "multi"})

	tr.AddComment(issue.ID, "first")
	tr.AddComment(issue.ID, "second")

	issues := readAllIssues(t)
	if len(issues[issue.ID].Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(issues[issue.ID].Comments))
	}
}

func TestAddComment_IssueNotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	err := tr.AddComment("nonexistent", "text")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestAddComment_ShortID(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "short id"})

	shortID := issue.ID[len("test-"):]
	err := tr.AddComment(shortID, "via short id")
	if err != nil {
		t.Fatalf("comment via short ID failed: %v", err)
	}

	issues := readAllIssues(t)
	if len(issues[issue.ID].Comments) != 1 {
		t.Error("comment not added via short ID")
	}
}
