package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func makeEvent(id string, evtType model.EventType) model.Event {
	return model.Event{ID: id, Type: evtType, Payload: model.CreatePayload{Title: id}, CreatedAt: time.Now().UTC(), CreatedBy: "test"}
}

func makeDone(id string) model.Event {
	return model.Event{ID: id, Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: id, Status: "DONE"}, CreatedAt: time.Now().UTC(), CreatedBy: "test"}
}

func makeStatusUpdate(id, status string) model.Event {
	return model.Event{ID: id, Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: &status}, CreatedAt: time.Now().UTC(), CreatedBy: "test"}
}

func ids(list ...string) map[string]bool {
	m := make(map[string]bool)
	for _, id := range list {
		m[id] = true
	}
	return m
}

func eventIDs(events []model.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.ID)
	}
	return out
}

func TestArchiveEvents_MovesToArchive(t *testing.T) {
	setupXpoDir(t)

	AppendEvent(makeEvent("a", model.EventTypeCreate))
	AppendEvent(makeDone("old"))

	res, err := ArchiveEvents(ids("old"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Archived, []string{"old"}) || len(res.Skipped) != 0 {
		t.Errorf("result = %+v, want archived [old], no skips", res)
	}

	remaining, _ := ReadEvents()
	if len(remaining) != 1 || remaining[0].ID != "a" {
		t.Errorf("issues.db should have only active event, got %d events", len(remaining))
	}

	archivedEvts, _ := ReadArchivedEvents()
	if len(archivedEvts) != 1 || archivedEvts[0].ID != "old" {
		t.Errorf("archive.db should have archived event, got %d events", len(archivedEvts))
	}
}

func TestArchiveEvents_CreatesBackup(t *testing.T) {
	setupXpoDir(t)

	AppendEvent(makeEvent("a", model.EventTypeCreate))

	ArchiveEvents(ids())

	entries, _ := os.ReadDir(".xpo")
	backupFound := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".bak") {
			backupFound = true
		}
	}
	if !backupFound {
		t.Error("backup file not created")
	}
}

func TestArchiveEvents_AppendsToExistingArchive(t *testing.T) {
	setupXpoDir(t)

	AppendEvent(makeDone("a"))
	AppendEvent(makeDone("b"))

	ArchiveEvents(ids("a"))

	AppendEvent(makeEvent("c", model.EventTypeCreate))

	ArchiveEvents(ids("b"))

	archivedEvts, _ := ReadArchivedEvents()
	if len(archivedEvts) != 2 {
		t.Errorf("archive should have 2 events from two rounds, got %d", len(archivedEvts))
	}
}

func TestArchiveEvents_EmptyArchiveList(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeEvent("a", model.EventTypeCreate))

	_, err := ArchiveEvents(ids())
	if err != nil {
		t.Fatal(err)
	}

	archPath := filepath.Join(".xpo", "archive.db")
	info, err := os.Stat(archPath)
	if err != nil {
		t.Fatal("archive.db should be created even if empty archive list")
	}
	if info.Size() != 0 {
		t.Error("archive.db should be empty when no events archived")
	}
}

func TestArchiveEvents_EmptyActiveList(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeDone("a"))

	_, err := ArchiveEvents(ids("a"))
	if err != nil {
		t.Fatal(err)
	}

	remaining, _ := ReadEvents()
	if len(remaining) != 0 {
		t.Errorf("issues.db should be empty, got %d events", len(remaining))
	}
}

func TestArchiveEvents_KeepsEventsAppendedAfterPreview(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeEvent("a", model.EventTypeCreate))
	AppendEvent(makeDone("old"))

	preview := ids("old")

	// Another agent writes between the preview and the archive.
	AppendEvent(makeEvent("late", model.EventTypeCreate))
	AppendEvent(makeStatusUpdate("a", "DOING"))

	if _, err := ArchiveEvents(preview); err != nil {
		t.Fatal(err)
	}

	remaining, _ := ReadEvents()
	if got, want := eventIDs(remaining), []string{"a", "late", "a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("issues.db = %v, want %v", got, want)
	}
}

func TestArchiveEvents_MovesLateEventsOfArchivedIssue(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeDone("old"))

	preview := ids("old")

	comment := model.Event{ID: "old", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: "c1", Text: "late"}, CreatedAt: time.Now().UTC(), CreatedBy: "test"}
	AppendEvent(comment)

	if _, err := ArchiveEvents(preview); err != nil {
		t.Fatal(err)
	}

	remaining, _ := ReadEvents()
	if len(remaining) != 0 {
		t.Errorf("issues.db should be empty, got %v", eventIDs(remaining))
	}
	archivedEvts, _ := ReadArchivedEvents()
	if len(archivedEvts) != 2 {
		t.Errorf("archive.db should hold both events of old, got %d", len(archivedEvts))
	}
}

func TestArchiveEvents_SkipsIssueReopenedAfterPreview(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeDone("reopened"))
	AppendEvent(makeDone("done"))

	preview := ids("reopened", "done")

	AppendEvent(makeStatusUpdate("reopened", "DOING"))

	res, err := ArchiveEvents(preview)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Archived, []string{"done"}) {
		t.Errorf("archived = %v, want [done]", res.Archived)
	}
	if !reflect.DeepEqual(res.Skipped, []string{"reopened"}) {
		t.Errorf("skipped = %v, want [reopened]", res.Skipped)
	}

	remaining, _ := ReadEvents()
	if got, want := eventIDs(remaining), []string{"reopened", "reopened"}; !reflect.DeepEqual(got, want) {
		t.Errorf("issues.db = %v, want %v", got, want)
	}
}

func TestArchiveEvents_ArchivesDeletedIssue(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeEvent("gone", model.EventTypeCreate))
	AppendEvent(model.Event{ID: "gone", Type: model.EventTypeDelete, CreatedAt: time.Now().UTC(), CreatedBy: "test"})

	res, err := ArchiveEvents(ids("gone"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Archived, []string{"gone"}) {
		t.Errorf("archived = %v, want [gone]", res.Archived)
	}
}

func TestArchiveEvents_ReplacesFileAtomically(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeEvent("a", model.EventTypeCreate))
	AppendEvent(makeDone("old"))

	path := filepath.Join(".xpo", "issues.db")
	before, _ := os.Stat(path)

	if _, err := ArchiveEvents(ids("old")); err != nil {
		t.Fatal(err)
	}

	after, _ := os.Stat(path)
	if os.SameFile(before, after) {
		t.Error("issues.db was rewritten in place; expected temp file + rename")
	}
}
