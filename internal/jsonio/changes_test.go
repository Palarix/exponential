package jsonio

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }

func evt(typ model.EventType, payload interface{}) model.Event {
	return model.Event{ID: "xpo-a", Type: typ, Payload: payload, CreatedAt: time.Unix(0, 0).UTC(), CreatedBy: "A <a@x>"}
}

func created() model.Event {
	return evt(model.EventTypeCreate, model.CreatePayload{
		Title:    "Old title",
		Status:   "DOING",
		Assignee: "A <a@x>",
		Labels:   []string{"feature", "CLI"},
		Dependencies: []model.Dependency{
			{SourceID: "xpo-a", TargetID: "xpo-b", Kind: "blocks"},
		},
	})
}

func TestEventChanges_StatusFromTo(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeUpdate, model.UpdatePayload{Status: strp("DONE")}),
	})
	want := map[string]FieldChange{"status": {From: "DOING", To: "DONE"}}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

func TestEventChanges_LabelsAddedRemoved(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeUpdate, model.UpdatePayload{Labels: []string{"CLI", "bug"}}),
	})
	want := map[string]FieldChange{"labels": {Added: []string{"bug"}, Removed: []string{"feature"}}}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

func TestEventChanges_DependenciesAddedRemoved(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeUpdate, model.UpdatePayload{Dependencies: []model.Dependency{
			{SourceID: "xpo-a", TargetID: "xpo-c", Kind: "relates_to"},
		}}),
	})
	want := map[string]FieldChange{"dependencies": {
		Added:   []model.Dependency{{SourceID: "xpo-a", TargetID: "xpo-c", Kind: "relates_to"}},
		Removed: []model.Dependency{{SourceID: "xpo-a", TargetID: "xpo-b", Kind: "blocks"}},
	}}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

func TestEventChanges_UnchangedFieldsLeftOut(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeUpdate, model.UpdatePayload{
			Status: strp("DOING"),
			Labels: []string{"CLI", "feature"}, // same set, other order
			Title:  strp("New title"),
		}),
	})
	want := map[string]FieldChange{"title": {From: "Old title", To: "New title"}}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

func TestEventChanges_EmptyValuesLeftOut(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeUpdate, model.UpdatePayload{Assignee: strp(""), Estimate: intp(3)}),
	})
	want := map[string]FieldChange{
		"assignee":     {From: "A <a@x>"},
		"story_points": {To: 3},
	}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

func TestEventChanges_ReplaysEarlierUpdates(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeUpdate, model.UpdatePayload{Status: strp("BLOCKED")}),
		evt(model.EventTypeUpdate, model.UpdatePayload{Status: strp("DOING")}),
	})
	want := map[string]FieldChange{"status": {From: "BLOCKED", To: "DOING"}}
	if !reflect.DeepEqual(out[2].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[2].Changes, want)
	}
}

func TestEventChanges_AllScalarKeys(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeUpdate, model.UpdatePayload{
			Description: strp("text"),
			ParentID:    strp("xpo-p"),
			Priority:    intp(2),
			CycleID:     strp("2026-11-16"),
			MilestoneID: strp("ms-1"),
			SortOrder:   strp("a5"),
		}),
	})
	want := map[string]FieldChange{
		"description":  {To: "text"},
		"parent_id":    {To: "xpo-p"},
		"priority":     {To: 2},
		"cycle_id":     {To: "2026-11-16"},
		"milestone_id": {To: "ms-1"},
	}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

func TestEventChanges_MergeSetsDone(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeMerge, model.MergePayload{Branch: "b"}),
	})
	want := map[string]FieldChange{"status": {From: "DOING", To: "DONE"}}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

func TestEventChanges_NoChangesKeyOmitted(t *testing.T) {
	out := ToEventSummaries([]model.Event{
		created(),
		evt(model.EventTypeComment, model.CommentPayload{ID: "c1", Text: "hi"}),
		evt(model.EventTypeUpdate, model.UpdatePayload{SortOrder: strp("a1")}),
	})
	for i, e := range out {
		b, _ := json.Marshal(e)
		if strings.Contains(string(b), `"changes"`) {
			t.Errorf("event %d (%s) has changes: %s", i, e.Type, b)
		}
	}
}

// Events read from the log carry map payloads, not typed structs.
func TestEventChanges_MapPayloads(t *testing.T) {
	var create, update map[string]interface{}
	json.Unmarshal([]byte(`{"title":"T","status":"PLANNED"}`), &create)
	json.Unmarshal([]byte(`{"status":"DOING","labels":null,"dependencies":null}`), &update)
	out := ToEventSummaries([]model.Event{
		evt(model.EventTypeCreate, create),
		evt(model.EventTypeUpdate, update),
	})
	want := map[string]FieldChange{"status": {From: "PLANNED", To: "DOING"}}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}

// An agent assigned itself while working for a user: show reports the user.
func TestEventChanges_AgentAssigneeResolvedToPrincipal(t *testing.T) {
	upd := evt(model.EventTypeUpdate, model.UpdatePayload{Assignee: strp("bot <agent@mcp>")})
	upd.CreatedBy = "bot <agent@mcp>"
	upd.OnBehalfOf = "U <u@x>"
	out := ToEventSummaries([]model.Event{
		evt(model.EventTypeCreate, model.CreatePayload{Title: "T"}),
		upd,
	})
	want := map[string]FieldChange{"assignee": {To: "U <u@x>"}}
	if !reflect.DeepEqual(out[1].Changes, want) {
		t.Fatalf("changes = %#v, want %#v", out[1].Changes, want)
	}
}
