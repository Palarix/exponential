package exponential

import (
	"testing"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// Events built in one call must carry strictly increasing times in the
// order they will be appended, so the log stays ordered by time even
// before storage's own guard (xpo-e669d4).
func assertStrictlyIncreasing(t *testing.T, events []model.Event) {
	t.Helper()
	if len(events) < 2 {
		t.Fatalf("want at least 2 events to compare, got %d", len(events))
	}
	for i := 1; i < len(events); i++ {
		if !events[i].CreatedAt.After(events[i-1].CreatedAt) {
			t.Errorf("event %d (%s %s) at %s is not after event %d (%s %s) at %s",
				i, events[i].Type, events[i].ID, events[i].CreatedAt,
				i-1, events[i-1].Type, events[i-1].ID, events[i-1].CreatedAt)
		}
	}
}

func projected(t *testing.T) map[string]*model.Issue {
	t.Helper()
	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	return ProjectIssues(events)
}

func TestBuildUpdate_CascadeTimesStrictlyIncrease(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.LastCompleted = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "story", ParentID: parent.ID, Status: "DOING"})

	events, _, err := tr.buildUpdate(child.ID, model.UpdatePayload{Status: sp("DONE")}, projected(t))
	if err != nil {
		t.Fatal(err)
	}
	assertStrictlyIncreasing(t, events)
}

func TestBuildUpdate_LinkOnlyTimesStrictlyIncrease(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "a"})
	for _, title := range []string{"b", "c"} {
		other, _ := tr.AddIssue(model.CreatePayload{Title: title})
		if _, err := tr.UpdateIssue(other.ID, model.UpdatePayload{Dependencies: []model.Dependency{
			{SourceID: other.ID, TargetID: a.ID, Kind: model.DependencyBlocks},
		}}, ""); err != nil {
			t.Fatal(err)
		}
	}

	// Clearing a's links removes the two links stored on b and c.
	events, _, err := tr.buildUpdate(a.ID, model.UpdatePayload{Dependencies: []model.Dependency{}}, projected(t))
	if err != nil {
		t.Fatal(err)
	}
	assertStrictlyIncreasing(t, events)
}

func TestBuildMergeEvents_MergeBeforeDone(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "work", Status: "DOING"})
	c := &Client{local: tr, Transport: tr}

	events, _, err := c.buildMergeEvents(projected(t)[issue.ID], "work-branch", "abc123", MergeStrategySquash)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].Type != model.EventTypeMerge {
		t.Fatalf("first event is %s, want MERGE", events[0].Type)
	}
	assertStrictlyIncreasing(t, events)
}
