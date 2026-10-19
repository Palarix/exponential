//go:build !windows

package chat

import (
	"errors"
	"slices"
	"testing"
)

// slowBody runs a slow turn for a message that says "slow", and the fixture
// turn for any other.
const slowBody = `case "$line" in
*slow*) head -n 3 "$FIXTURE"; sleep 1; tail -n +4 "$FIXTURE" ;;
*) cat "$FIXTURE" ;;
esac`

func userTexts(events []Event) []string {
	var out []string
	for _, e := range eventsOfType(events, EventUserMessage) {
		out = append(out, e.Text)
	}
	return out
}

// queueEvents returns the queue sizes the live queue events published.
func queueSizes(ch <-chan Event) []int {
	var out []int
	for {
		select {
		case e := <-ch:
			if e.Type == EventQueue {
				out = append(out, len(e.Queue))
			}
		default:
			return out
		}
	}
}

func TestRunner_QueueDrainsAsOneMessage(t *testing.T) {
	r, store, _ := newTestRunner(t, slowBody)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	if queued, err := r.SendOrQueue(m.ID, SendRequest{Text: "slow one"}); err != nil || queued {
		t.Fatalf("first message: queued %v, err %v", queued, err)
	}
	waitForEvent(t, ch, EventTextDelta)

	if queued, err := r.SendOrQueue(m.ID, SendRequest{Text: "two", Context: &Context{IssueID: "xpo-1"}}); err != nil || !queued {
		t.Fatalf("second message: queued %v, err %v", queued, err)
	}
	notes := []AttachmentData{{Name: "notes.txt", MIME: "text/plain", Data: []byte("x")}}
	if queued, err := r.SendOrQueue(m.ID, SendRequest{Text: "three", Model: "haiku", Attachments: notes}); err != nil || !queued {
		t.Fatalf("third message: queued %v, err %v", queued, err)
	}
	q := r.Queued(m.ID)
	if len(q) != 2 || q[0].Text != "two" || q[1].Text != "three" || len(q[1].Attachments) != 1 {
		t.Fatalf("queue = %+v", q)
	}
	waitDone(t, r)

	meta, events, _ := store.Get(m.ID)
	if got := userTexts(events); !slices.Equal(got, []string{"slow one", "two\n\nthree"}) {
		t.Fatalf("user messages = %q", got)
	}
	drained := eventsOfType(events, EventUserMessage)[1]
	if len(drained.Attachments) != 1 || drained.Context == nil || drained.Context.IssueID != "xpo-1" {
		t.Errorf("drained message = %+v, want the attachment and the latest context", drained)
	}
	if n := len(eventsOfType(events, EventTurnDone)); n != 2 {
		t.Errorf("turn_done events = %d, want 2", n)
	}
	if meta.Model != "haiku" || meta.State != StateIdle {
		t.Errorf("meta = %+v, want the latest model and idle", meta)
	}
	if len(r.Queued(m.ID)) != 0 {
		t.Errorf("queue after the drain = %+v", r.Queued(m.ID))
	}
	if got := queueSizes(ch); !slices.Equal(got, []int{1, 2, 0}) {
		t.Errorf("published queue sizes = %v, want [1 2 0]", got)
	}
}

func TestRunner_StopKeepsTheQueue(t *testing.T) {
	r, store, _ := newTestRunner(t, `case "$line" in *slow*) head -n 3 "$FIXTURE"; sleep 30 ;; *) cat "$FIXTURE" ;; esac`)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	r.SendOrQueue(m.ID, SendRequest{Text: "slow one"})
	waitForEvent(t, ch, EventTextDelta)
	r.SendOrQueue(m.ID, SendRequest{Text: "two"})
	r.Cancel(m.ID)
	waitDone(t, r)

	if q := r.Queued(m.ID); len(q) != 1 {
		t.Fatalf("queue after Stop = %+v, want the message kept", q)
	}
	if _, events, _ := store.Get(m.ID); len(userTexts(events)) != 1 {
		t.Fatalf("a stopped turn sent the queue: %q", userTexts(events))
	}

	// The next finished turn drains it.
	r.SendOrQueue(m.ID, SendRequest{Text: "three"})
	waitDone(t, r)
	if _, events, _ := store.Get(m.ID); !slices.Equal(userTexts(events), []string{"slow one", "three", "two"}) {
		t.Fatalf("user messages = %q", userTexts(events))
	}
}

func TestRunner_RemoveQueued(t *testing.T) {
	r, store, _ := newTestRunner(t, slowBody)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	r.SendOrQueue(m.ID, SendRequest{Text: "slow one"})
	waitForEvent(t, ch, EventTextDelta)
	r.SendOrQueue(m.ID, SendRequest{Text: "two"})
	id := r.Queued(m.ID)[0].ID

	got, err := r.RemoveQueued(m.ID, id)
	if err != nil || got.Text != "two" {
		t.Fatalf("RemoveQueued = %+v, %v", got, err)
	}
	if _, err := r.RemoveQueued(m.ID, id); !errors.Is(err, ErrQueuedNotFound) {
		t.Fatalf("second RemoveQueued = %v, want ErrQueuedNotFound", err)
	}
	waitDone(t, r)
	if _, events, _ := store.Get(m.ID); len(userTexts(events)) != 1 {
		t.Errorf("a removed message was sent: %q", userTexts(events))
	}
}

func TestRunner_EditIsNeverQueued(t *testing.T) {
	r, store, _ := newTestRunner(t, slowBody)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	r.SendOrQueue(m.ID, SendRequest{Text: "slow one"})
	waitForEvent(t, ch, EventTextDelta)
	if _, err := r.SendOrQueue(m.ID, SendRequest{Text: "x", Edit: true}); !errors.Is(err, ErrBusy) {
		t.Fatalf("edit while running = %v, want ErrBusy", err)
	}
	waitDone(t, r)
}

func TestRunner_ArchiveClearsTheQueue(t *testing.T) {
	r, store, _ := newTestRunner(t, `case "$line" in *slow*) head -n 3 "$FIXTURE"; sleep 30 ;; *) cat "$FIXTURE" ;; esac`)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	r.SendOrQueue(m.ID, SendRequest{Text: "slow one"})
	waitForEvent(t, ch, EventTextDelta)
	r.SendOrQueue(m.ID, SendRequest{Text: "two"})
	r.Cancel(m.ID)
	waitDone(t, r)

	if _, err := r.Archive(m.ID); err != nil {
		t.Fatal(err)
	}
	if q := r.Queued(m.ID); len(q) != 0 {
		t.Fatalf("queue after archive = %+v", q)
	}
}
