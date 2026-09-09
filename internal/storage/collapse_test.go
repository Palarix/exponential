package storage

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	exec.Command("git", "init", "-b", "main").Run()
	exec.Command("git", "config", "user.email", "test@test.com").Run()
	exec.Command("git", "config", "user.name", "Test").Run()
	return dir
}

func writeEvents(t *testing.T, events []model.Event) {
	t.Helper()
	path := filepath.Join(".xpo", "issues.db")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, evt := range events {
		b, _ := json.Marshal(evt)
		f.Write(b)
		f.WriteString("\n")
	}
	f.Close()
}

func commitDB(t *testing.T) {
	t.Helper()
	exec.Command("git", "add", ".xpo/issues.db").Run()
	exec.Command("git", "commit", "-m", "snap").Run()
}

func readBackStatus(t *testing.T, id string) model.IssueStatus {
	t.Helper()
	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	issues := make(map[string]*model.Issue)
	for _, evt := range events {
		switch evt.Type {
		case model.EventTypeCreate:
			b, _ := json.Marshal(evt.Payload)
			var p model.CreatePayload
			json.Unmarshal(b, &p)
			st := model.IssueStatus(p.Status)
			if st == "" {
				st = model.StatusBacklog
			}
			issues[evt.ID] = &model.Issue{ID: evt.ID, Status: st}
		case model.EventTypeUpdate:
			issue, ok := issues[evt.ID]
			if !ok {
				continue
			}
			b, _ := json.Marshal(evt.Payload)
			var p model.UpdatePayload
			json.Unmarshal(b, &p)
			if p.Status != nil {
				issue.Status = model.IssueStatus(*p.Status)
			}
		}
	}
	issue, ok := issues[id]
	if !ok {
		t.Fatalf("issue %s not found", id)
	}
	return issue.Status
}

func strptr(s string) *string { return &s }

func readBackDescription(t *testing.T, id string) string {
	t.Helper()
	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	desc := ""
	for _, evt := range events {
		if evt.ID != id {
			continue
		}
		switch evt.Type {
		case model.EventTypeCreate:
			b, _ := json.Marshal(evt.Payload)
			var p model.CreatePayload
			json.Unmarshal(b, &p)
			desc = p.Description
		case model.EventTypeUpdate:
			b, _ := json.Marshal(evt.Payload)
			var p model.UpdatePayload
			json.Unmarshal(b, &p)
			if p.Description != nil {
				desc = *p.Description
			}
		}
	}
	return desc
}

func TestCollapseMergesIntoLastUpdate(t *testing.T) {
	setupTestRepo(t)

	create := model.Event{
		ID:        "m",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "Multi", Description: "original"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
	writeEvents(t, []model.Event{create})
	commitDB(t)

	// First uncommitted update: changes status
	AppendEventCollapsed(model.Event{
		ID: "m", Type: model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Status: strptr("DOING")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})

	// Second uncommitted update: changes description
	AppendEventCollapsed(model.Event{
		ID: "m", Type: model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Description: strptr("updated desc")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})

	// Now change description again — must merge into the LAST update (the
	// one that already carries a description), not the first one.
	AppendEventCollapsed(model.Event{
		ID: "m", Type: model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Description: strptr("final desc")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})

	if got := readBackDescription(t, "m"); got != "final desc" {
		t.Errorf("description = %q, want %q", got, "final desc")
	}
}

func TestCollapsePreservesCreateStatus(t *testing.T) {
	setupTestRepo(t)

	create := model.Event{
		ID:        "x",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "X", Status: "PLANNED"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
	writeEvents(t, []model.Event{create})
	commitDB(t)

	// Uncommitted: change status to BACKLOG
	err := AppendEventCollapsed(model.Event{
		ID:        "x",
		Type:      model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Status: strptr("BACKLOG")},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := readBackStatus(t, "x"); got != model.StatusBacklog {
		t.Errorf("status = %s, want BACKLOG (collapse pruned the status change)", got)
	}
}

func TestCollapsePreservesStatusAfterSortOrderMerge(t *testing.T) {
	setupTestRepo(t)

	create := model.Event{
		ID:        "x",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "X", Status: "PLANNED"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
	writeEvents(t, []model.Event{create})
	commitDB(t)

	// Uncommitted sort_order change
	err := AppendEventCollapsed(model.Event{
		ID:        "x",
		Type:      model.EventTypeUpdate,
		Payload:   model.UpdatePayload{SortOrder: strptr("a0")},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Now change status — merges into the existing uncommitted UPDATE
	err = AppendEventCollapsed(model.Event{
		ID:        "x",
		Type:      model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Status: strptr("DOING")},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := readBackStatus(t, "x"); got != model.StatusDoing {
		t.Errorf("status = %s, want DOING", got)
	}
}

func TestCollapsePrunesRoundTrip(t *testing.T) {
	setupTestRepo(t)

	create := model.Event{
		ID:        "x",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "X", Status: "PLANNED"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
	writeEvents(t, []model.Event{create})
	commitDB(t)

	// Move to DOING
	AppendEventCollapsed(model.Event{
		ID: "x", Type: model.EventTypeUpdate,
		Payload: model.UpdatePayload{Status: strptr("DOING")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})
	if got := readBackStatus(t, "x"); got != model.StatusDoing {
		t.Fatalf("after DOING: status = %s", got)
	}

	// Move back to PLANNED — round-trip should collapse to nothing
	AppendEventCollapsed(model.Event{
		ID: "x", Type: model.EventTypeUpdate,
		Payload: model.UpdatePayload{Status: strptr("PLANNED")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})
	if got := readBackStatus(t, "x"); got != model.StatusPlanned {
		t.Errorf("after round-trip: status = %s, want PLANNED", got)
	}
}

func TestCollapseDefaultBacklogStatus(t *testing.T) {
	setupTestRepo(t)

	// Create with no explicit status → defaults to BACKLOG
	create := model.Event{
		ID:        "y",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "Y"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
	writeEvents(t, []model.Event{create})
	commitDB(t)

	// Move to PLANNED
	err := AppendEventCollapsed(model.Event{
		ID: "y", Type: model.EventTypeUpdate,
		Payload: model.UpdatePayload{Status: strptr("PLANNED")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := readBackStatus(t, "y"); got != model.StatusPlanned {
		t.Errorf("status = %s, want PLANNED", got)
	}

	// Move back to BACKLOG — should correctly prune as round-trip
	AppendEventCollapsed(model.Event{
		ID: "y", Type: model.EventTypeUpdate,
		Payload: model.UpdatePayload{Status: strptr("BACKLOG")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})
	if got := readBackStatus(t, "y"); got != model.StatusBacklog {
		t.Errorf("after round-trip: status = %s, want BACKLOG", got)
	}
}

func TestCollapseMergePreservesEarlierFields(t *testing.T) {
	setupTestRepo(t)

	create := model.Event{
		ID:        "z",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "Z", Status: "BACKLOG"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
	writeEvents(t, []model.Event{create})
	commitDB(t)

	// Change title (uncommitted)
	AppendEventCollapsed(model.Event{
		ID: "z", Type: model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Title: strptr("Z renamed")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})

	// Change status — should merge with title change, both survive prune
	AppendEventCollapsed(model.Event{
		ID: "z", Type: model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Status: strptr("DOING")},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})

	events, _ := ReadEvents()
	// Should have CREATE + one merged UPDATE
	updates := 0
	for _, evt := range events {
		if evt.ID == "z" && evt.Type == model.EventTypeUpdate {
			updates++
			b, _ := json.Marshal(evt.Payload)
			var p model.UpdatePayload
			json.Unmarshal(b, &p)
			if p.Title == nil || *p.Title != "Z renamed" {
				t.Errorf("merged UPDATE lost title: %+v", p)
			}
			if p.Status == nil || *p.Status != "DOING" {
				t.Errorf("merged UPDATE lost status: %+v", p)
			}
		}
	}
	if updates != 1 {
		t.Errorf("expected 1 merged UPDATE, got %d", updates)
	}
}
