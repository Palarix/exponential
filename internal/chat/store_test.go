package chat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "chats"))
}

func TestStore_CreateAndGet(t *testing.T) {
	s := newTestStore(t)
	m, err := s.Create()
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == "" || m.State != StateIdle || m.CreatedAt.IsZero() {
		t.Fatalf("unexpected meta: %+v", m)
	}

	got, events, err := s.Get(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != m.ID || len(events) != 0 {
		t.Fatalf("got %+v, %d events", got, len(events))
	}
}

func TestStore_CreateCallsOnCreateDirOnce(t *testing.T) {
	s := newTestStore(t)
	calls := 0
	s.OnCreateDir = func() { calls++ }
	for range 2 {
		if _, err := s.Create(); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("OnCreateDir called %d times, want 1", calls)
	}
}

func TestStore_MetaFileIsPrivate(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	info, err := os.Stat(filepath.Join(s.Dir, m.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %v, want 0600", info.Mode().Perm())
	}
}

func TestStore_AppendAssignsSequence(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()

	a, err := s.Append(m.ID, Event{Type: EventUserMessage, Text: "hi"}, Event{Type: EventAssistantText, Text: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.Append(m.ID, Event{Type: EventTurnDone})
	if a[0].Seq != 1 || a[1].Seq != 2 || b[0].Seq != 3 {
		t.Fatalf("seqs = %d %d %d", a[0].Seq, a[1].Seq, b[0].Seq)
	}
	if a[0].Time.IsZero() {
		t.Fatal("Append must stamp Time")
	}

	_, events, _ := s.Get(m.ID)
	if len(events) != 3 || events[1].Text != "hello" {
		t.Fatalf("events = %+v", events)
	}
}

func TestStore_AppendSequenceSurvivesReopen(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	s.Append(m.ID, Event{Type: EventUserMessage}, Event{Type: EventTurnDone})

	reopened := NewStore(s.Dir)
	got, _ := reopened.Append(m.ID, Event{Type: EventUserMessage})
	if got[0].Seq != 3 {
		t.Fatalf("seq after reopen = %d, want 3", got[0].Seq)
	}
}

func TestStore_GetUnknown(t *testing.T) {
	s := newTestStore(t)
	if _, _, err := s.Get("nope"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if _, _, err := s.Get("../etc/passwd"); err != ErrNotFound {
		t.Fatalf("path traversal err = %v, want ErrNotFound", err)
	}
}

func TestStore_ListNewestFirst(t *testing.T) {
	s := newTestStore(t)
	old, _ := s.Create()
	newer, _ := s.Create()
	old.UpdatedAt = time.Now().Add(-time.Hour)
	if err := s.SaveMeta(old); err != nil {
		t.Fatal(err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != newer.ID {
		t.Fatalf("list = %+v", list)
	}
}

func TestStore_ListEmptyDir(t *testing.T) {
	s := newTestStore(t)
	list, err := s.List()
	if err != nil || len(list) != 0 {
		t.Fatalf("list = %v, err = %v", list, err)
	}
}

func TestStore_MarkInterrupted(t *testing.T) {
	s := newTestStore(t)
	running, _ := s.Create()
	running.State = StateRunning
	s.SaveMeta(running)
	idle, _ := s.Create()

	if err := s.MarkInterrupted(); err != nil {
		t.Fatal(err)
	}
	r, _, _ := s.Get(running.ID)
	i, _, _ := s.Get(idle.ID)
	if r.State != StateInterrupted || i.State != StateIdle {
		t.Fatalf("states = %s, %s", r.State, i.State)
	}
}

func TestTitleFrom(t *testing.T) {
	cases := map[string]string{
		"how many issues?":        "how many issues?",
		"  first line\nsecond":    "first line",
		"":                        "",
		"äöü " + repeat("x", 100): "äöü " + repeat("x", 100),
		"äöü " + repeat("x", 300): "äöü " + repeat("x", 196) + "…",
	}
	for in, want := range cases {
		if got := TitleFrom(in); got != want {
			t.Errorf("TitleFrom(%q) = %q, want %q", in, got, want)
		}
	}
}

func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}

func TestStore_Delete(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	s.Append(m.ID, Event{Type: EventUserMessage})

	if err := s.Delete(m.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Get(m.ID); err != ErrNotFound {
		t.Fatalf("Get after Delete: err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, m.ID+".jsonl")); !os.IsNotExist(err) {
		t.Fatalf("event log still exists: %v", err)
	}
	if err := s.Delete(m.ID); err != ErrNotFound {
		t.Fatalf("second Delete: err = %v, want ErrNotFound", err)
	}
	if err := s.Delete("../x"); err != ErrNotFound {
		t.Fatalf("invalid id: err = %v", err)
	}
}

func TestStore_DeleteResetsSequenceCache(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	s.Append(m.ID, Event{Type: EventUserMessage})
	s.Delete(m.ID)
	if _, ok := s.lastSeq[m.ID]; ok {
		t.Fatal("Delete must forget the cached seq")
	}
}

func TestStore_Rename(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()

	got, err := s.Rename(m.ID, "  Login bug triage  ")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Login bug triage" {
		t.Fatalf("title = %q", got.Title)
	}
	stored, _ := s.Meta(m.ID)
	if stored.Title != "Login bug triage" {
		t.Fatalf("stored title = %q", stored.Title)
	}
	if _, err := s.Rename(m.ID, "   "); err != ErrEmptyTitle {
		t.Fatalf("empty title: err = %v", err)
	}
	if _, err := s.Rename("00000000-0000-4000-8000-000000000000", "x"); err != ErrNotFound {
		t.Fatalf("unknown chat: err = %v", err)
	}
}

func TestStore_RenameMakesTheTitleFinal(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	if got, err := s.SetAutoTitle(m.ID, "Fix login"); err != nil || got.Title != "Fix login" || !got.AutoTitle {
		t.Fatalf("SetAutoTitle = %+v, %v", got, err)
	}
	if got, _ := s.Rename(m.ID, "Mine"); got.AutoTitle {
		t.Fatal("a rename left the title automatic")
	}
	if stored, _ := s.Meta(m.ID); stored.AutoTitle || stored.Title != "Mine" {
		t.Fatalf("stored = %+v", stored)
	}
}

func TestStore_ReplaceAutoTitle(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	s.SetAutoTitle(m.ID, "placeholder")

	if ok, err := s.ReplaceAutoTitle(m.ID, "other", "Generated"); ok || err != nil {
		t.Fatalf("replace with a stale placeholder = %v, %v; want no change", ok, err)
	}
	if ok, err := s.ReplaceAutoTitle(m.ID, "placeholder", "Generated"); !ok || err != nil {
		t.Fatalf("replace = %v, %v", ok, err)
	}
	if stored, _ := s.Meta(m.ID); stored.Title != "Generated" {
		t.Fatalf("title = %q", stored.Title)
	}
	if _, err := s.ReplaceAutoTitle("00000000-0000-4000-8000-000000000000", "a", "b"); err != ErrNotFound {
		t.Fatalf("unknown chat: err = %v", err)
	}
}

func TestStore_MetaMigratesClaudeSessionID(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	old := `{"id":"` + m.ID + `","title":"old","agent":"claude","claude_session_id":"sess-old","state":"idle"}`
	if err := os.WriteFile(s.metaPath(m.ID), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := s.Meta(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "sess-old" || got.Agent != "claude" {
		t.Fatalf("meta = %+v", got)
	}

	if err := s.SaveMeta(got); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(s.metaPath(m.ID))
	if strings.Contains(string(data), "claude_session_id") || !strings.Contains(string(data), `"session_id": "sess-old"`) {
		t.Fatalf("saved meta = %s", data)
	}
}

// recordChanges collects the IDs OnChange reports.
func recordChanges(s *Store) *[]string {
	var ids []string
	s.OnChange = func(id string) { ids = append(ids, id) }
	return &ids
}

func TestStore_OnChangeOnMetaWritesAndDelete(t *testing.T) {
	s := newTestStore(t)
	ids := recordChanges(s)
	m, _ := s.Create()
	if _, err := s.Rename(m.ID, "renamed"); err != nil {
		t.Fatal(err)
	}
	m, _ = s.Meta(m.ID)
	m.State = StateRunning
	if err := s.SaveMeta(m); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(m.ID); err != nil {
		t.Fatal(err)
	}
	want := []string{m.ID, m.ID, m.ID, m.ID}
	if strings.Join(*ids, ",") != strings.Join(want, ",") {
		t.Fatalf("OnChange = %v, want %v", *ids, want)
	}
}

func TestStore_OnChangeOnApprovalEventsOnly(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	ids := recordChanges(s)
	if _, err := s.Append(m.ID, Event{Type: EventUserMessage}, Event{Type: EventAssistantText}, Event{Type: EventToolCall}); err != nil {
		t.Fatal(err)
	}
	if len(*ids) != 0 {
		t.Fatalf("OnChange on plain events: %v", *ids)
	}
	s.Append(m.ID, Event{Type: EventApprovalRequested})
	s.Append(m.ID, Event{Type: EventApprovalResolved})
	if len(*ids) != 2 || (*ids)[0] != m.ID || (*ids)[1] != m.ID {
		t.Fatalf("OnChange = %v, want two calls for %s", *ids, m.ID)
	}
}

func TestStore_NoOnChangeOnFailedDelete(t *testing.T) {
	s := newTestStore(t)
	ids := recordChanges(s)
	if err := s.Delete("00000000-0000-0000-0000-000000000000"); err != ErrNotFound {
		t.Fatalf("err = %v", err)
	}
	if len(*ids) != 0 {
		t.Fatalf("OnChange = %v", *ids)
	}
}
