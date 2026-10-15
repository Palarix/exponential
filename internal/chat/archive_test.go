package chat

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestStore_CreateHasZeroUsage(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	if m.Usage == nil || *m.Usage != (Usage{}) {
		t.Fatalf("usage = %+v, want zero totals", m.Usage)
	}
}

// A chat saved before usage totals existed gets them from its turn_done
// events the first time it is read.
func TestStore_MetaBackfillsUsage(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	s.Append(m.ID,
		Event{Type: EventUserMessage, Text: "hi"},
		Event{Type: EventTurnDone, TokensIn: 10, TokensOut: 4, CostUSD: 0.02},
		Event{Type: EventTurnDone, TokensIn: 5, TokensOut: 1},
	)
	old := `{"id":"` + m.ID + `","title":"old","agent":"claude","state":"idle"}`
	if err := os.WriteFile(s.metaPath(m.ID), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := s.Meta(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := Usage{Turns: 2, TokensIn: 15, TokensOut: 5, CostUSD: 0.02, CostKnown: true}
	if got.Usage == nil || *got.Usage != want {
		t.Fatalf("usage = %+v, want %+v", got.Usage, want)
	}

	// The backfill is saved, so the log is read once.
	s.Append(m.ID, Event{Type: EventTurnDone, TokensIn: 100})
	again, _ := s.Meta(m.ID)
	if *again.Usage != want {
		t.Fatalf("usage after save = %+v, want %+v", again.Usage, want)
	}
}

func TestStore_MetaBackfillsUnknownCost(t *testing.T) {
	s := newTestStore(t)
	m, _ := s.Create()
	s.Append(m.ID, Event{Type: EventTurnDone, TokensIn: 3, TokensOut: 2})
	os.WriteFile(s.metaPath(m.ID), []byte(`{"id":"`+m.ID+`","state":"idle"}`), 0o600)

	got, _ := s.Meta(m.ID)
	if got.Usage.CostKnown || got.Usage.Turns != 1 {
		t.Fatalf("usage = %+v, want one turn with unknown cost", got.Usage)
	}
}

func TestRunner_TurnsAddUsage(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	for _, text := range []string{"one", "two"} {
		if err := r.Send(m.ID, SendRequest{Text: text}); err != nil {
			t.Fatal(err)
		}
		r.Wait()
	}

	meta, _ := store.Meta(m.ID)
	want := Usage{Turns: 2, TokensIn: 6, TokensOut: 14, CostUSD: 0.02, CostKnown: true}
	if meta.Usage == nil || meta.Usage.Turns != want.Turns || meta.Usage.TokensIn != want.TokensIn ||
		meta.Usage.TokensOut != want.TokensOut || !meta.Usage.CostKnown || meta.Usage.CostUSD < 0.0199 || meta.Usage.CostUSD > 0.0201 {
		t.Fatalf("usage = %+v, want %+v", meta.Usage, want)
	}
}

func TestRunner_ArchiveAndUnarchive(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()

	got, err := r.Archive(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ArchivedAt == nil {
		t.Fatalf("meta = %+v, want archived_at", got)
	}
	if err := r.Send(m.ID, SendRequest{Text: "hi"}); !errors.Is(err, ErrArchived) {
		t.Fatalf("send to archived chat: err = %v, want ErrArchived", err)
	}

	got, err = r.Unarchive(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ArchivedAt != nil {
		t.Fatalf("meta = %+v, want no archived_at", got)
	}
	if err := r.Send(m.ID, SendRequest{Text: "hi"}); err != nil {
		t.Fatalf("send after unarchive: %v", err)
	}
	r.Wait()
}

func TestRunner_ArchiveRefusesRunningChat(t *testing.T) {
	r, store, _ := newTestRunner(t, `sleep 1; cat "$FIXTURE"`)
	m, _ := store.Create()
	if err := r.Send(m.ID, SendRequest{Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Archive(m.ID); !errors.Is(err, ErrBusy) {
		t.Fatalf("archive running chat: err = %v, want ErrBusy", err)
	}
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.ArchivedAt != nil {
		t.Fatal("running chat was archived")
	}
}

func TestRunner_ArchiveRefusesWorkMode(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	m.Mode = ModeWork
	m.Work = &WorkBinding{IssueID: "xpo-1", Worktree: t.TempDir()}
	store.SaveMeta(m)

	if _, err := r.Archive(m.ID); !errors.Is(err, ErrArchiveInWork) {
		t.Fatalf("archive chat in work mode: err = %v, want ErrArchiveInWork", err)
	}
}

func TestRunner_BindWorkRefusesArchivedChat(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, func(c *RunnerConfig) { c.CanWork = true })
	m, _ := store.Create()
	r.Archive(m.ID)

	_, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: t.TempDir()})
	if !errors.Is(err, ErrArchived) {
		t.Fatalf("bind archived chat: err = %v, want ErrArchived", err)
	}
}

func TestRunner_ArchiveUnknownChat(t *testing.T) {
	r, _, _ := newTestRunner(t, `cat "$FIXTURE"`)
	if _, err := r.Archive("00000000-0000-4000-8000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if _, err := r.Unarchive("00000000-0000-4000-8000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestStore_AutoArchive(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	mk := func(age time.Duration, state string, work bool) Meta {
		m, _ := s.Create()
		m.UpdatedAt = now.Add(-age)
		m.State = state
		if work {
			m.Mode = ModeWork
			m.Work = &WorkBinding{IssueID: "xpo-1"}
		}
		s.SaveMeta(m)
		return m
	}
	fresh := mk(time.Hour, StateIdle, false)
	old := mk(40*24*time.Hour, StateIdle, false)
	oldInterrupted := mk(40*24*time.Hour, StateInterrupted, false)
	oldRunning := mk(40*24*time.Hour, StateRunning, false)
	oldWork := mk(40*24*time.Hour, StateIdle, true)

	n, err := s.AutoArchive(30*24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("archived %d, want 2", n)
	}
	for _, m := range []Meta{old, oldInterrupted} {
		got, err := s.Meta(m.ID)
		if err != nil || got.ArchivedAt == nil {
			t.Errorf("%s not archived (err %v)", m.ID, err)
		}
		if !got.UpdatedAt.Equal(m.UpdatedAt) {
			t.Errorf("%s: archiving changed updated_at", m.ID)
		}
	}
	for _, m := range []Meta{fresh, oldRunning, oldWork} {
		got, err := s.Meta(m.ID)
		if err != nil || got.ArchivedAt != nil {
			t.Errorf("%s archived or gone (err %v)", m.ID, err)
		}
	}

	// A second run archives nothing more and deletes nothing.
	if n, _ := s.AutoArchive(30*24*time.Hour, now); n != 0 {
		t.Fatalf("second run archived %d, want 0", n)
	}
	if list, _ := s.List(); len(list) != 5 {
		t.Fatalf("%d chats left, want 5", len(list))
	}
}
