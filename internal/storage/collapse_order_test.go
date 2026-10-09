package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

const (
	webActor   = "Jane Doe <jane@example.com>"
	agentActor = "claude-code"
)

func webUpdate(id string, p model.UpdatePayload, at time.Time) model.Event {
	return model.Event{ID: id, Type: model.EventTypeUpdate, Payload: p, CreatedAt: at, CreatedBy: webActor, Source: "web"}
}

func agentUpdate(id string, p model.UpdatePayload, at time.Time) model.Event {
	return model.Event{ID: id, Type: model.EventTypeUpdate, Payload: p, CreatedAt: at, CreatedBy: agentActor, OnBehalfOf: webActor, Source: "mcp"}
}

func seedCommitted(t *testing.T, base time.Time, ids ...string) {
	t.Helper()
	var events []model.Event
	for i, id := range ids {
		events = append(events, model.Event{
			ID: id, Type: model.EventTypeCreate,
			Payload:   model.CreatePayload{Title: id, Status: "PLANNED"},
			CreatedAt: base.Add(time.Duration(i) * time.Second), CreatedBy: webActor, Source: "web",
		})
	}
	writeEvents(t, events)
	commitDB(t)
}

func mustAppendCollapsed(t *testing.T, evt model.Event) {
	t.Helper()
	if err := AppendEventCollapsed(evt); err != nil {
		t.Fatal(err)
	}
}

func updatesFor(t *testing.T, id string) []model.Event {
	t.Helper()
	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	var out []model.Event
	for _, evt := range events {
		if evt.ID == id && evt.Type == model.EventTypeUpdate {
			out = append(out, evt)
		}
	}
	return out
}

func updatePayload(evt model.Event) model.UpdatePayload {
	b, _ := json.Marshal(evt.Payload)
	var p model.UpdatePayload
	json.Unmarshal(b, &p)
	return p
}

func TestCollapseDoesNotMergeAcrossActors(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a")

	mustAppendCollapsed(t, agentUpdate("a", model.UpdatePayload{Assignee: strptr("claude-code")}, base.Add(time.Minute)))
	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{SortOrder: strptr("a0")}, base.Add(2*time.Minute)))

	updates := updatesFor(t, "a")
	if len(updates) != 2 {
		t.Fatalf("expected 2 UPDATE events, got %d", len(updates))
	}
	if updates[0].CreatedBy != agentActor || updates[0].Source != "mcp" {
		t.Errorf("first event attribution = %q/%q, want agent/mcp", updates[0].CreatedBy, updates[0].Source)
	}
	if p := updatePayload(updates[0]); p.SortOrder != nil {
		t.Errorf("agent event carries web sort_order: %+v", p)
	}
	if updates[1].CreatedBy != webActor || updates[1].Source != "web" {
		t.Errorf("second event attribution = %q/%q, want web", updates[1].CreatedBy, updates[1].Source)
	}
	if p := updatePayload(updates[1]); p.Assignee != nil {
		t.Errorf("web event carries agent assignee: %+v", p)
	}
}

func TestCollapseDoesNotMergeAcrossOnBehalfOf(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a")

	first := agentUpdate("a", model.UpdatePayload{Title: strptr("one")}, base.Add(time.Minute))
	second := agentUpdate("a", model.UpdatePayload{Title: strptr("two")}, base.Add(2*time.Minute))
	second.OnBehalfOf = "Someone Else <else@example.com>"
	mustAppendCollapsed(t, first)
	mustAppendCollapsed(t, second)

	if n := len(updatesFor(t, "a")); n != 2 {
		t.Fatalf("expected 2 UPDATE events, got %d", n)
	}
}

func TestCollapseDoesNotMergePastNewerEventForIssue(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a")

	// web: pending status change, then the agent moves the issue on,
	// then web edits sort order. The stale web status must not re-apply.
	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{Status: strptr("BACKLOG")}, base.Add(time.Minute)))
	mustAppendCollapsed(t, agentUpdate("a", model.UpdatePayload{Status: strptr("DOING")}, base.Add(2*time.Minute)))
	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{SortOrder: strptr("a0")}, base.Add(3*time.Minute)))

	if got := readBackStatus(t, "a"); got != model.StatusDoing {
		t.Errorf("status = %s, want DOING", got)
	}
	updates := updatesFor(t, "a")
	if len(updates) != 3 {
		t.Fatalf("expected 3 UPDATE events, got %d", len(updates))
	}
	if p := updatePayload(updates[2]); p.Status != nil {
		t.Errorf("last web event carries stale status: %+v", p)
	}
}

func TestCollapseMovesMergedEventToEnd(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a", "b")

	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{SortOrder: strptr("a1")}, base.Add(1*time.Minute)))
	mustAppendCollapsed(t, webUpdate("b", model.UpdatePayload{SortOrder: strptr("b1")}, base.Add(2*time.Minute)))
	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{SortOrder: strptr("a2")}, base.Add(3*time.Minute)))
	mustAppendCollapsed(t, webUpdate("b", model.UpdatePayload{SortOrder: strptr("b2")}, base.Add(4*time.Minute)))
	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{Title: strptr("A")}, base.Add(5*time.Minute)))

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("expected 2 CREATE + 2 collapsed UPDATE, got %d events", len(events))
	}
	for i := 1; i < len(events); i++ {
		if !events[i].CreatedAt.After(events[i-1].CreatedAt) {
			t.Errorf("line %d (%s) not after line %d (%s)", i+1, events[i].CreatedAt, i, events[i-1].CreatedAt)
		}
	}
	last := events[len(events)-1]
	if last.ID != "a" {
		t.Errorf("last event is %s, want a (merged event should move to the end)", last.ID)
	}
	p := updatePayload(last)
	if p.SortOrder == nil || *p.SortOrder != "a2" || p.Title == nil || *p.Title != "A" {
		t.Errorf("merged payload = %+v, want sort_order a2 and title A", p)
	}
}

func TestCollapsePruneUsesRunningState(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a") // committed: PLANNED

	mustAppendCollapsed(t, agentUpdate("a", model.UpdatePayload{Status: strptr("DOING")}, base.Add(time.Minute)))
	// web reverts to PLANNED: equal to committed state, but not a no-op.
	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{Status: strptr("PLANNED")}, base.Add(2*time.Minute)))

	if got := readBackStatus(t, "a"); got != model.StatusPlanned {
		t.Errorf("status = %s, want PLANNED", got)
	}
}

func TestCollapseSameActorRoundTripLeavesNoUpdate(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a")

	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{Title: strptr("bar")}, base.Add(time.Minute)))
	mustAppendCollapsed(t, webUpdate("a", model.UpdatePayload{Title: strptr("a")}, base.Add(2*time.Minute)))

	if n := len(updatesFor(t, "a")); n != 0 {
		t.Errorf("expected no pending UPDATE after round-trip, got %d", n)
	}
}

func TestCollapseCommentEditKeepsOriginalTimeAndPosition(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a")

	comment := func(cid, text string, at time.Time) model.Event {
		return model.Event{ID: "a", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: cid, Text: text}, CreatedAt: at, CreatedBy: webActor, Source: "web"}
	}
	mustAppendCollapsed(t, comment("c1", "first", base.Add(1*time.Minute)))
	mustAppendCollapsed(t, comment("c2", "second", base.Add(2*time.Minute)))
	mustAppendCollapsed(t, comment("c1", "first, edited", base.Add(3*time.Minute)))

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected CREATE + 2 COMMENT, got %d events", len(events))
	}
	b, _ := json.Marshal(events[1].Payload)
	var p model.CommentPayload
	json.Unmarshal(b, &p)
	if p.ID != "c1" || p.Text != "first, edited" {
		t.Errorf("line 2 = %+v, want edited c1 in place", p)
	}
	if !events[1].CreatedAt.Equal(base.Add(1 * time.Minute)) {
		t.Errorf("edited comment created_at = %s, want original %s", events[1].CreatedAt, base.Add(time.Minute))
	}
}

func TestCollapseKeepsTimestampsStrictlyIncreasing(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedCommitted(t, base, "a", "b") // committed CREATEs at base and base+1s

	// Incoming events stamped earlier than (or equal to) what is already in
	// the file: a slow clock, or a caller that stamped before an earlier write.
	mustAppendCollapsed(t, agentUpdate("a", model.UpdatePayload{Title: strptr("A")}, base))
	mustAppendCollapsed(t, webUpdate("b", model.UpdatePayload{SortOrder: strptr("b1")}, base.Add(-time.Hour)))
	mustAppendCollapsed(t, webUpdate("b", model.UpdatePayload{Title: strptr("B")}, base.Add(time.Second)))

	if v, err := CheckEventOrder(); err != nil || len(v) != 0 {
		t.Fatalf("CheckEventOrder = %+v, %v; want no violations", v, err)
	}
	events, _ := ReadEvents()
	if got, want := events[len(events)-1].CreatedAt, base.Add(time.Second+2); !got.Equal(want) {
		t.Errorf("last event at %s, want %s (committed tail + 1ns steps)", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}
