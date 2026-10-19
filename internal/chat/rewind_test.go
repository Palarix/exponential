package chat

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// seqLog numbers events from seq 1, like the store.
func seqLog(events ...Event) []Event {
	for i := range events {
		events[i].Seq = i + 1
	}
	return events
}

func evUser(text string) Event     { return Event{Type: EventUserMessage, Text: text} }
func evDone(anchor string) Event   { return Event{Type: EventTurnDone, Anchor: anchor} }
func evFailed(anchor string) Event { return Event{Type: EventError, Anchor: anchor} }

func TestEditPoint(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		events   []Event
		want     editPlan
		err      bool
	}{
		{name: "no message", provider: ProviderClaude, events: seqLog(Event{Type: EventModeChanged}), err: true},
		{name: "claude first turn starts fresh", provider: ProviderClaude,
			events: seqLog(evUser("a"), Event{Type: EventAssistantText}, evDone("a1")),
			want:   editPlan{FromSeq: 1, Fresh: true}},
		{name: "claude resumes at the turn before", provider: ProviderClaude,
			events: seqLog(evUser("a"), evDone("a1"), evUser("b"), evDone("a2")),
			want:   editPlan{FromSeq: 3, Anchor: "a1"}},
		{name: "claude turn before has no anchor", provider: ProviderClaude,
			events: seqLog(evUser("a"), evFailed(""), evUser("b"), evDone("a2")),
			err:    true},
		{name: "claude stopped turn before keeps its anchor", provider: ProviderClaude,
			events: seqLog(evUser("a"), Event{Type: EventTurnStopped, Anchor: "a1"}, evUser("b"), evDone("a2")),
			want:   editPlan{FromSeq: 3, Anchor: "a1"}},
		{name: "claude drops agent turns after the message", provider: ProviderClaude,
			events: seqLog(evUser("a"), evDone("a1"), evUser("b"), evDone("a2"), Event{Type: EventAgentTurn}, evDone("a3")),
			want:   editPlan{FromSeq: 3, Anchor: "a1"}},
		{name: "claude turn before is an agent turn", provider: ProviderClaude,
			events: seqLog(evUser("a"), evDone("a1"), Event{Type: EventAgentTurn}, evDone("a2"), evUser("b"), evDone("a3")),
			want:   editPlan{FromSeq: 5, Anchor: "a2"}},
		{name: "codex reverts the edited turn", provider: ProviderCodex,
			events: seqLog(evUser("a"), evDone("turn-1"), evUser("b"), evDone("turn-2")),
			want:   editPlan{FromSeq: 3, Anchor: "turn-2"}},
		{name: "codex first turn", provider: ProviderCodex,
			events: seqLog(evUser("a"), evDone("turn-1")),
			want:   editPlan{FromSeq: 1, Anchor: "turn-1"}},
		{name: "codex edited turn has no anchor", provider: ProviderCodex,
			events: seqLog(evUser("a"), evDone("turn-1"), evUser("b"), evFailed("")),
			err:    true},
		{name: "an earlier rewind drops its turns", provider: ProviderClaude,
			events: seqLog(evUser("a"), evDone("a1"), evUser("b"), evDone("a2"), Event{Type: EventRewound, FromSeq: 3}, evUser("c"), evDone("a3")),
			want:   editPlan{FromSeq: 6, Anchor: "a1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := editPoint(c.events, c.provider)
			if c.err {
				if !errors.Is(err, ErrCannotEdit) {
					t.Fatalf("err = %v, want ErrCannotEdit", err)
				}
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("editPoint = %+v, %v; want %+v", got, err, c.want)
			}
		})
	}
}

func TestRunner_TurnEndRecordsAnchor(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	_, events, _ := store.Get(m.ID)
	if d := eventsOfType(events, EventTurnDone); len(d) != 1 || d[0].Anchor != "asst-1" {
		t.Fatalf("turn_done = %+v, want anchor asst-1", d)
	}
}

func TestCodexDriver_TurnEndRecordsAnchor(t *testing.T) {
	r, store, _ := newCodexRunner(t, "")
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x", Provider: "codex"})
	r.Wait()
	_, events, _ := store.Get(m.ID)
	if d := eventsOfType(events, EventTurnDone); len(d) != 1 || d[0].Anchor != "turn-1" {
		t.Fatalf("turn_done = %+v, want anchor turn-1", d)
	}
}

// lastUser returns the seq and text of the last user message.
func lastUser(events []Event) (int, string) {
	users := eventsOfType(events, EventUserMessage)
	if len(users) == 0 {
		return 0, ""
	}
	u := users[len(users)-1]
	return u.Seq, u.Text
}

func TestRunner_EditRewindsClaude(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	r.Send(m.ID, SendRequest{Text: "two"})
	r.Wait()
	_, before, _ := store.Get(m.ID)
	seq, _ := lastUser(before)

	if err := r.Send(m.ID, SendRequest{Text: "two again", Edit: true}); err != nil {
		t.Fatalf("edit = %v", err)
	}
	r.Wait()

	if n := starts(t, dir); n != 2 {
		t.Fatalf("agent started %d times, want a restart for the rewind", n)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), "--resume\nsess-42\n") || !strings.Contains(string(args), "--resume-session-at\nasst-1\n") {
		t.Fatalf("restart args = %s", args)
	}
	_, events, _ := store.Get(m.ID)
	rw := eventsOfType(events, EventRewound)
	if len(rw) != 1 || rw[0].FromSeq != seq {
		t.Fatalf("rewound = %+v, want from seq %d", rw, seq)
	}
	if _, text := lastUser(events); text != "two again" {
		t.Errorf("last message = %q", text)
	}
	if last := events[len(events)-1]; last.Type != EventTurnDone {
		t.Errorf("last event = %+v", last)
	}
}

func TestRunner_EditFirstTurnStartsFresh(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	if err := r.Send(m.ID, SendRequest{Text: "one again", Edit: true}); err != nil {
		t.Fatalf("edit = %v", err)
	}
	r.Wait()

	if n := starts(t, dir); n != 2 {
		t.Fatalf("agent started %d times, want twice", n)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if strings.Contains(string(args), "--resume") {
		t.Fatalf("a fresh session resumed: %s", args)
	}
}

func TestRunner_EditNeedsAnAnchor(t *testing.T) {
	r, store, _ := newTestRunner(t, `grep -v '"assistant"' "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	r.Send(m.ID, SendRequest{Text: "two"})
	r.Wait()
	_, before, _ := store.Get(m.ID)

	if err := r.Send(m.ID, SendRequest{Text: "two again", Edit: true}); !errors.Is(err, ErrCannotEdit) {
		t.Fatalf("edit = %v, want ErrCannotEdit", err)
	}
	if r.Running(m.ID) {
		t.Error("a refused edit left a turn running")
	}
	if _, after, _ := store.Get(m.ID); len(after) != len(before) {
		t.Errorf("a refused edit logged events: %+v", after[len(before):])
	}
}

func TestCodexDriver_EditRevertsTheTurn(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one", Provider: "codex"})
	r.Wait()
	r.Send(m.ID, SendRequest{Text: "two", Provider: "codex"})
	r.Wait()
	os.Remove(filepath.Join(dir, "stdin.txt"))

	if err := r.Send(m.ID, SendRequest{Text: "two again", Provider: "codex", Edit: true}); err != nil {
		t.Fatalf("edit = %v", err)
	}
	r.Wait()

	data, _ := os.ReadFile(filepath.Join(dir, "stdin.txt"))
	var methods []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		for _, m := range []string{"thread/resume", "thread/revert", "turn/start"} {
			if strings.Contains(line, `"`+m+`"`) {
				methods = append(methods, m)
			}
		}
	}
	if want := []string{"thread/resume", "thread/revert", "turn/start"}; !slices.Equal(methods, want) {
		t.Fatalf("requests = %v, want %v", methods, want)
	}
	p := params(sentRequests(t, dir)["thread/revert"][0])
	if p["threadId"] != "thr-1" || p["beforeTurnId"] != "turn-1" {
		t.Errorf("thread/revert params = %v", p)
	}
	_, events, _ := store.Get(m.ID)
	if last := events[len(events)-1]; last.Type != EventTurnDone {
		t.Errorf("last event = %+v", last)
	}
}

func TestCodexDriver_EditRevertFails(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one", Provider: "codex"})
	r.Wait()
	os.WriteFile(filepath.Join(dir, "revert-error"), nil, 0o644)

	r.Send(m.ID, SendRequest{Text: "one again", Provider: "codex", Edit: true})
	r.Wait()

	_, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventError || !strings.Contains(last.Message, "unknown turn") {
		t.Fatalf("last event = %+v", last)
	}
	if n := len(sentRequests(t, dir)["turn/start"]); n != 1 {
		t.Errorf("turn/start sent %d times, want only the first turn's", n)
	}
}
