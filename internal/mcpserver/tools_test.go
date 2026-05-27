package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/inputs"
	"github.com/kuyio/beats/internal/model"
)

// setup creates a temp .beats workspace and returns a toolset bound to it.
// The cleanup func restores the original working directory.
func setup(t *testing.T) (*toolset, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	if err := os.MkdirAll(beatsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
	}
	return newToolset(cfg), func() { os.Chdir(origDir) }
}

func TestAddAndShow(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, addRes, err := ts.add(context.Background(), nil, inputs.AddInput{
		Title:       "Hello",
		Description: "## Body\n\nWith `code`.",
		Status:      "PLANNED",
		Labels:      []string{"feature"},
		StoryPoints: 3,
	})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if addRes.ID == "" {
		t.Fatal("expected non-empty issue ID")
	}
	if addRes.Status != "PLANNED" {
		t.Errorf("Status: got %q want PLANNED", addRes.Status)
	}

	_, showRes, err := ts.show(context.Background(), nil, showIn{ID: addRes.ID})
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if showRes.Description != "## Body\n\nWith `code`." {
		t.Errorf("description round-trip mismatch: %q", showRes.Description)
	}
	if showRes.StoryPoints != 3 {
		t.Errorf("StoryPoints: got %d want 3", showRes.StoryPoints)
	}
	if len(showRes.Labels) != 1 || showRes.Labels[0] != "feature" {
		t.Errorf("Labels mismatch: %v", showRes.Labels)
	}
}

func TestUpdateStatusTransition(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, addRes, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "Work"})

	doing := "DOING"
	_, updRes, err := ts.update(context.Background(), nil, updateIn{
		ID:          addRes.ID,
		UpdateInput: inputs.UpdateInput{Status: &doing},
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updRes.ID != addRes.ID {
		t.Errorf("ID mismatch: got %q want %q", updRes.ID, addRes.ID)
	}

	_, showRes, _ := ts.show(context.Background(), nil, showIn{ID: addRes.ID})
	if showRes.Status != "DOING" {
		t.Errorf("Status: got %q want DOING", showRes.Status)
	}
}

func TestUpdateEmptyRejected(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, addRes, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "x"})

	_, _, err := ts.update(context.Background(), nil, updateIn{ID: addRes.ID})
	if err == nil {
		t.Fatal("expected error for empty update")
	}
}

func TestCommentRoundTrip(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, addRes, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "x"})

	body := "## Comment\n\nWith `code` and \"quotes\" and $vars."
	if _, _, err := ts.comment(context.Background(), nil, commentIn{ID: addRes.ID, Body: body}); err != nil {
		t.Fatalf("comment failed: %v", err)
	}

	_, showRes, _ := ts.show(context.Background(), nil, showIn{ID: addRes.ID})
	if len(showRes.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(showRes.Comments))
	}
	if showRes.Comments[0].Text != body {
		t.Errorf("comment text mismatch:\n got: %q\nwant: %q", showRes.Comments[0].Text, body)
	}
}

func TestCommentRequiresBody(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, addRes, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "x"})
	if _, _, err := ts.comment(context.Background(), nil, commentIn{ID: addRes.ID, Body: ""}); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestList(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	ts.add(context.Background(), nil, inputs.AddInput{Title: "A", Labels: []string{"feature"}})
	ts.add(context.Background(), nil, inputs.AddInput{Title: "B", Labels: []string{"bug"}})
	ts.add(context.Background(), nil, inputs.AddInput{Title: "C", Labels: []string{"bug"}})

	_, all, err := ts.list(context.Background(), nil, listIn{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(all.Issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(all.Issues))
	}

	_, bugs, _ := ts.list(context.Background(), nil, listIn{Label: "bug"})
	if len(bugs.Issues) != 2 {
		t.Errorf("expected 2 bug issues, got %d", len(bugs.Issues))
	}
}

func TestLink(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, a, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "A"})
	_, b, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "B"})

	_, linkRes, err := ts.link(context.Background(), nil, linkIn{
		Source: a.ID,
		Target: b.ID,
		Type:   "depends_on",
	})
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}
	if linkRes.Kind != "depends_on" {
		t.Errorf("Kind: got %q want depends_on", linkRes.Kind)
	}

	_, showRes, _ := ts.show(context.Background(), nil, showIn{ID: a.ID})
	if len(showRes.Dependencies) != 1 || showRes.Dependencies[0].TargetID != b.ID {
		t.Errorf("dependency not persisted: %+v", showRes.Dependencies)
	}
}

func TestLinkInvalidType(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, a, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "A"})
	_, b, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "B"})

	_, _, err := ts.link(context.Background(), nil, linkIn{
		Source: a.ID,
		Target: b.ID,
		Type:   "garbage",
	})
	if err == nil {
		t.Fatal("expected error for invalid link type")
	}
}

func TestAddWithStatusAndLinks(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, target, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "Target"})

	_, addRes, err := ts.add(context.Background(), nil, inputs.AddInput{
		Title:  "Linked at creation",
		Status: "PLANNED",
		Links: []inputs.LinkInput{
			{Target: target.ID, Type: "blocks"},
		},
	})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	_, showRes, _ := ts.show(context.Background(), nil, showIn{ID: addRes.ID})
	if showRes.Status != "PLANNED" {
		t.Errorf("Status: got %q want PLANNED", showRes.Status)
	}
	if len(showRes.Dependencies) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(showRes.Dependencies))
	}
	if showRes.Dependencies[0].Kind != model.DependencyBlocks {
		t.Errorf("dep kind: got %q want blocks", showRes.Dependencies[0].Kind)
	}
	if showRes.Dependencies[0].SourceID != addRes.ID {
		t.Errorf("dep SourceID not auto-filled: %q", showRes.Dependencies[0].SourceID)
	}
}

func TestHistoryIncludesEvents(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	_, a, _ := ts.add(context.Background(), nil, inputs.AddInput{Title: "x"})
	doing := "DOING"
	ts.update(context.Background(), nil, updateIn{ID: a.ID, UpdateInput: inputs.UpdateInput{Status: &doing}})

	_, h, err := ts.history(context.Background(), nil, historyIn{ID: a.ID})
	if err != nil {
		t.Fatalf("history failed: %v", err)
	}
	if len(h.Events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(h.Events))
	}
}

func TestAgentIdentityFromEnv(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	t.Setenv(EnvAgentIdentity, "AgentTest <agent@test>")

	_, a, err := ts.add(context.Background(), nil, inputs.AddInput{Title: "Identity check"})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	_, showRes, _ := ts.show(context.Background(), nil, showIn{ID: a.ID})
	if showRes.CreatedBy != "AgentTest <agent@test>" {
		t.Errorf("CreatedBy: got %q want %q", showRes.CreatedBy, "AgentTest <agent@test>")
	}
}

func TestAgentIdentityFallsBackToConfig(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()

	// Ensure no env override is set
	os.Unsetenv(EnvAgentIdentity)

	_, a, err := ts.add(context.Background(), nil, inputs.AddInput{Title: "Config default"})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	_, showRes, _ := ts.show(context.Background(), nil, showIn{ID: a.ID})
	if showRes.CreatedBy != "Test User <test@test.com>" {
		t.Errorf("CreatedBy: got %q want config default", showRes.CreatedBy)
	}
}
