package exponential

import (
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestDeleteIssue_Simple(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "delete me"})

	err := tr.DeleteIssue(issue.ID, "testing", false)
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if _, exists := issues[issue.ID]; exists {
		t.Error("deleted issue should be filtered out of projected issues")
	}
}

func TestDeleteIssue_CascadeTrue(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	child1, _ := tr.AddIssue(model.CreatePayload{Title: "child1", ParentID: parent.ID})
	child2, _ := tr.AddIssue(model.CreatePayload{Title: "child2", ParentID: parent.ID})

	err := tr.DeleteIssue(parent.ID, "cascade", true)
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if _, exists := issues[parent.ID]; exists {
		t.Error("parent should be deleted")
	}
	if _, exists := issues[child1.ID]; exists {
		t.Error("child1 should be cascade-deleted")
	}
	if _, exists := issues[child2.ID]; exists {
		t.Error("child2 should be cascade-deleted")
	}
}

func TestDeleteIssue_CascadeFalse_UnparentsChildren(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID})

	err := tr.DeleteIssue(parent.ID, "no cascade", false)
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if _, exists := issues[parent.ID]; exists {
		t.Error("parent should be deleted")
	}
	if _, exists := issues[child.ID]; !exists {
		t.Fatal("child should NOT be deleted")
	}
	if issues[child.ID].ParentID != "" {
		t.Errorf("child ParentID should be cleared, got %q", issues[child.ID].ParentID)
	}
}

func TestDeleteIssue_NotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	err := tr.DeleteIssue("nonexistent", "testing", false)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestDeleteIssue_NoChildren(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "standalone"})
	other, _ := tr.AddIssue(model.CreatePayload{Title: "unrelated"})

	tr.DeleteIssue(issue.ID, "clean", true)

	issues := readAllIssues(t)
	if _, exists := issues[other.ID]; !exists {
		t.Error("unrelated issue should not be affected")
	}
}
