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
		"äöü " + repeat("x", 100): "äöü " + repeat("x", 56) + "…",
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

func TestStore_Prune(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	mk := func(age time.Duration, state string) Meta {
		m, _ := s.Create()
		m.UpdatedAt = now.Add(-age)
		m.State = state
		s.SaveMeta(m)
		return m
	}
	fresh := mk(time.Hour, StateIdle)
	old := mk(40*24*time.Hour, StateIdle)
	oldInterrupted := mk(40*24*time.Hour, StateInterrupted)
	oldRunning := mk(40*24*time.Hour, StateRunning)

	n, err := s.Prune(30*24*time.Hour, 200, now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("pruned %d, want 2", n)
	}
	for _, m := range []Meta{old, oldInterrupted} {
		if _, err := s.Meta(m.ID); err != ErrNotFound {
			t.Errorf("%s not pruned", m.ID)
		}
	}
	for _, m := range []Meta{fresh, oldRunning} {
		if _, err := s.Meta(m.ID); err != nil {
			t.Errorf("%s pruned: %v", m.ID, err)
		}
	}
}

func TestStore_PruneKeepsMaxCount(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	var ids []string
	for i := range 5 {
		m, _ := s.Create()
		m.UpdatedAt = now.Add(-time.Duration(i) * time.Minute) // ids[0] newest
		s.SaveMeta(m)
		ids = append(ids, m.ID)
	}
	n, _ := s.Prune(30*24*time.Hour, 3, now)
	if n != 2 {
		t.Fatalf("pruned %d, want 2", n)
	}
	list, _ := s.List()
	if len(list) != 3 || list[0].ID != ids[0] || list[2].ID != ids[2] {
		t.Fatalf("kept %+v", list)
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
