package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func strp(s string) *string { return &s }
func intptr(i int) *int     { return &i }

// ---------------------------------------------------------------------------
// mergeUpdatePayloads — per-field coverage
// ---------------------------------------------------------------------------

func makeMergeEvent(payload model.UpdatePayload) model.Event {
	return model.Event{
		ID: "x", Type: model.EventTypeUpdate,
		Payload: payload, CreatedAt: time.Now().UTC(), CreatedBy: "test",
	}
}

func decodeMergedPayload(t *testing.T, evt model.Event) model.UpdatePayload {
	t.Helper()
	b, _ := json.Marshal(evt.Payload)
	var p model.UpdatePayload
	json.Unmarshal(b, &p)
	return p
}

func TestMergeUpdate_AllFieldsIndividually(t *testing.T) {
	cases := []struct {
		name    string
		old     model.UpdatePayload
		new_    model.UpdatePayload
		check   func(t *testing.T, p model.UpdatePayload)
	}{
		{"Title", model.UpdatePayload{Title: strp("old")}, model.UpdatePayload{Title: strp("new")},
			func(t *testing.T, p model.UpdatePayload) {
				if p.Title == nil || *p.Title != "new" {
					t.Errorf("Title = %v", p.Title)
				}
			}},
		{"Description", model.UpdatePayload{}, model.UpdatePayload{Description: strp("desc")},
			func(t *testing.T, p model.UpdatePayload) {
				if p.Description == nil || *p.Description != "desc" {
					t.Errorf("Description = %v", p.Description)
				}
			}},
		{"Status", model.UpdatePayload{}, model.UpdatePayload{Status: strp("DOING")},
			func(t *testing.T, p model.UpdatePayload) {
				if p.Status == nil || *p.Status != "DOING" {
					t.Errorf("Status = %v", p.Status)
				}
			}},
		{"ParentID", model.UpdatePayload{}, model.UpdatePayload{ParentID: strp("p1")},
			func(t *testing.T, p model.UpdatePayload) {
				if p.ParentID == nil || *p.ParentID != "p1" {
					t.Errorf("ParentID = %v", p.ParentID)
				}
			}},
		{"Estimate", model.UpdatePayload{}, model.UpdatePayload{Estimate: intptr(5)},
			func(t *testing.T, p model.UpdatePayload) {
				if p.Estimate == nil || *p.Estimate != 5 {
					t.Errorf("Estimate = %v", p.Estimate)
				}
			}},
		{"Priority", model.UpdatePayload{}, model.UpdatePayload{Priority: intptr(2)},
			func(t *testing.T, p model.UpdatePayload) {
				if p.Priority == nil || *p.Priority != 2 {
					t.Errorf("Priority = %v", p.Priority)
				}
			}},
		{"SortOrder", model.UpdatePayload{}, model.UpdatePayload{SortOrder: strp("z1")},
			func(t *testing.T, p model.UpdatePayload) {
				if p.SortOrder == nil || *p.SortOrder != "z1" {
					t.Errorf("SortOrder = %v", p.SortOrder)
				}
			}},
		{"Assignee", model.UpdatePayload{}, model.UpdatePayload{Assignee: strp("Alice")},
			func(t *testing.T, p model.UpdatePayload) {
				if p.Assignee == nil || *p.Assignee != "Alice" {
					t.Errorf("Assignee = %v", p.Assignee)
				}
			}},
		{"CycleID", model.UpdatePayload{}, model.UpdatePayload{CycleID: strp("2026-06-01")},
			func(t *testing.T, p model.UpdatePayload) {
				if p.CycleID == nil || *p.CycleID != "2026-06-01" {
					t.Errorf("CycleID = %v", p.CycleID)
				}
			}},
		{"Labels", model.UpdatePayload{}, model.UpdatePayload{Labels: []string{"bug", "feature"}},
			func(t *testing.T, p model.UpdatePayload) {
				if len(p.Labels) != 2 || p.Labels[0] != "bug" {
					t.Errorf("Labels = %v", p.Labels)
				}
			}},
		{"Dependencies", model.UpdatePayload{}, model.UpdatePayload{Dependencies: []model.Dependency{{SourceID: "x", TargetID: "y", Kind: "blocks"}}},
			func(t *testing.T, p model.UpdatePayload) {
				if len(p.Dependencies) != 1 || p.Dependencies[0].TargetID != "y" {
					t.Errorf("Dependencies = %v", p.Dependencies)
				}
			}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			existing := makeMergeEvent(tc.old)
			merged := mergeUpdatePayloads(&existing, makeMergeEvent(tc.new_))
			if !merged {
				t.Fatal("merge should return true")
			}
			tc.check(t, decodeMergedPayload(t, existing))
		})
	}
}

func TestMergeUpdate_NilDoesNotOverwrite(t *testing.T) {
	existing := makeMergeEvent(model.UpdatePayload{Title: strp("keep"), Status: strp("DOING")})
	mergeUpdatePayloads(&existing, makeMergeEvent(model.UpdatePayload{Assignee: strp("Alice")}))

	p := decodeMergedPayload(t, existing)
	if p.Title == nil || *p.Title != "keep" {
		t.Error("nil incoming Title should not clear existing")
	}
	if p.Status == nil || *p.Status != "DOING" {
		t.Error("nil incoming Status should not clear existing")
	}
	if p.Assignee == nil || *p.Assignee != "Alice" {
		t.Error("Assignee should be set from incoming")
	}
}

func TestMergeUpdate_TimestampUpdated(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	existing := model.Event{ID: "x", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Title: strp("a")}, CreatedAt: t1}
	mergeUpdatePayloads(&existing, model.Event{ID: "x", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Title: strp("b")}, CreatedAt: t2})

	if !existing.CreatedAt.Equal(t2) {
		t.Errorf("CreatedAt should be updated to incoming, got %v", existing.CreatedAt)
	}
}

// ---------------------------------------------------------------------------
// mergeCommentPayloads
// ---------------------------------------------------------------------------

func TestMergeComment_SameID(t *testing.T) {
	existing := model.Event{ID: "x", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: "c1", Text: "old"}, CreatedAt: time.Now().UTC()}
	incoming := model.Event{ID: "x", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: "c1", Text: "updated"}, CreatedAt: time.Now().UTC()}

	merged := mergeCommentPayloads(&existing, incoming)
	if !merged {
		t.Error("same comment ID should merge")
	}
	b, _ := json.Marshal(existing.Payload)
	var p model.CommentPayload
	json.Unmarshal(b, &p)
	if p.Text != "updated" {
		t.Errorf("text = %q, want updated", p.Text)
	}
}

func TestMergeComment_DifferentID(t *testing.T) {
	existing := model.Event{ID: "x", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: "c1", Text: "first"}, CreatedAt: time.Now().UTC()}
	incoming := model.Event{ID: "x", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: "c2", Text: "second"}, CreatedAt: time.Now().UTC()}

	merged := mergeCommentPayloads(&existing, incoming)
	if merged {
		t.Error("different comment IDs should not merge")
	}
	b, _ := json.Marshal(existing.Payload)
	var p model.CommentPayload
	json.Unmarshal(b, &p)
	if p.Text != "first" {
		t.Error("existing should be unchanged when merge returns false")
	}
}

// ---------------------------------------------------------------------------
// pruneNoopUpdates — per-field
// ---------------------------------------------------------------------------

func TestPruneNoop_SingleFieldNoop(t *testing.T) {
	committed := map[string]*model.Issue{
		"x": {ID: "x", Title: "T", Status: model.StatusPlanned, Estimate: 3, Assignee: "A"},
	}
	cases := []struct {
		name    string
		payload model.UpdatePayload
	}{
		{"Title", model.UpdatePayload{Title: strp("T")}},
		{"Status", model.UpdatePayload{Status: strp("PLANNED")}},
		{"Estimate", model.UpdatePayload{Estimate: intptr(3)}},
		{"Assignee", model.UpdatePayload{Assignee: strp("A")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := []model.Event{{ID: "x", Type: model.EventTypeUpdate, Payload: tc.payload, CreatedAt: time.Now().UTC()}}
			result := pruneNoopUpdates(events, committed)
			if len(result) != 0 {
				t.Errorf("single-field noop should be pruned, got %d events", len(result))
			}
		})
	}
}

func TestPruneNoop_NonExistentIssue_Passthrough(t *testing.T) {
	committed := map[string]*model.Issue{}
	events := []model.Event{{ID: "new", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: strp("DOING")}, CreatedAt: time.Now().UTC()}}
	result := pruneNoopUpdates(events, committed)
	if len(result) != 1 {
		t.Error("update on non-existent issue should pass through")
	}
}

func TestPruneNoop_NonUpdateEvent_Passthrough(t *testing.T) {
	committed := map[string]*model.Issue{}
	events := []model.Event{{ID: "x", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "X"}, CreatedAt: time.Now().UTC()}}
	result := pruneNoopUpdates(events, committed)
	if len(result) != 1 {
		t.Error("CREATE events should pass through unchanged")
	}
}

func TestPruneNoop_LabelsEqual(t *testing.T) {
	committed := map[string]*model.Issue{
		"x": {ID: "x", Labels: []string{"bug", "feature"}},
	}
	events := []model.Event{{ID: "x", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Labels: []string{"bug", "feature"}}, CreatedAt: time.Now().UTC()}}
	result := pruneNoopUpdates(events, committed)
	if len(result) != 0 {
		t.Error("identical labels should be pruned")
	}
}

func TestPruneNoop_LabelsDifferent(t *testing.T) {
	committed := map[string]*model.Issue{
		"x": {ID: "x", Labels: []string{"bug"}},
	}
	events := []model.Event{{ID: "x", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Labels: []string{"bug", "feature"}}, CreatedAt: time.Now().UTC()}}
	result := pruneNoopUpdates(events, committed)
	if len(result) != 1 {
		t.Error("different labels should not be pruned")
	}
}

func TestPruneNoop_DepsEqual(t *testing.T) {
	dep := model.Dependency{SourceID: "x", TargetID: "y", Kind: "blocks"}
	committed := map[string]*model.Issue{
		"x": {ID: "x", Dependencies: []model.Dependency{dep}},
	}
	events := []model.Event{{ID: "x", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Dependencies: []model.Dependency{dep}}, CreatedAt: time.Now().UTC()}}
	result := pruneNoopUpdates(events, committed)
	if len(result) != 0 {
		t.Error("identical deps should be pruned")
	}
}

// ---------------------------------------------------------------------------
// strSlicesEqual / depsEqual
// ---------------------------------------------------------------------------

func TestStrSlicesEqual(t *testing.T) {
	cases := []struct{ a, b []string; want bool }{
		{nil, nil, true},
		{[]string{}, []string{}, true},
		{[]string{"a"}, []string{"a"}, true},
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"a"}, []string{"b"}, false},
		{[]string{"a"}, []string{"a", "b"}, false},
		{[]string{"a", "b"}, []string{"b", "a"}, false},
	}
	for _, tc := range cases {
		if got := strSlicesEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("strSlicesEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestDepsEqual(t *testing.T) {
	d1 := model.Dependency{SourceID: "a", TargetID: "b", Kind: "blocks"}
	d2 := model.Dependency{SourceID: "a", TargetID: "c", Kind: "blocks"}

	if !depsEqual([]model.Dependency{d1}, []model.Dependency{d1}) {
		t.Error("identical deps should be equal")
	}
	if depsEqual([]model.Dependency{d1}, []model.Dependency{d2}) {
		t.Error("different deps should not be equal")
	}
	if depsEqual([]model.Dependency{d1}, []model.Dependency{}) {
		t.Error("different length deps should not be equal")
	}
}

// ---------------------------------------------------------------------------
// countLines
// ---------------------------------------------------------------------------

func TestCountLines(t *testing.T) {
	cases := []struct{ data string; want int }{
		{"", 0},
		{"\n", 0},
		{"a\n", 1},
		{"a\nb\n", 2},
		{"a\nb", 2},
		{"a\nb\n\n", 2},
	}
	for _, tc := range cases {
		if got := countLines([]byte(tc.data)); got != tc.want {
			t.Errorf("countLines(%q) = %d, want %d", tc.data, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// rewriteFile
// ---------------------------------------------------------------------------

func TestRewriteFile_AtomicWrite(t *testing.T) {
	setupBeatsDir(t)
	path := filepath.Join(".beats", "issues.db")

	committed := []byte(`{"id":"a"}\n`)
	uncommitted := []model.Event{
		{ID: "b", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "B"}, CreatedAt: time.Now().UTC(), CreatedBy: "t"},
	}

	err := rewriteFile(path, committed, uncommitted)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file should be cleaned up after rename")
	}
}

func TestRewriteFile_EmptyUncommitted(t *testing.T) {
	setupBeatsDir(t)
	path := filepath.Join(".beats", "issues.db")

	committed := []byte(`{"id":"a"}` + "\n")
	err := rewriteFile(path, committed, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != `{"id":"a"}`+"\n" {
		t.Errorf("file should contain only committed bytes, got %q", string(data))
	}
}
