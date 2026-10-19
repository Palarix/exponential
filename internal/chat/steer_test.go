//go:build !windows

package chat

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// steerBody runs the fixture turn slowly for the first user message, and
// only records the others: a steer must not start a turn of its own.
const steerBody = `printf '%s\n' "$line" >> "$(dirname "$FIXTURE")/users.txt"
case "$line" in
*steer-me*) ;;
*) head -n 3 "$FIXTURE"; sleep 1; tail -n +4 "$FIXTURE" ;;
esac`

func TestRunner_SteerGoesIntoTheRunningTurn(t *testing.T) {
	r, store, dir := newTestRunner(t, steerBody)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	if err := r.Send(m.ID, SendRequest{Text: "x"}); err != nil {
		t.Fatal(err)
	}
	waitForEvent(t, ch, EventTextDelta)

	r.SendOrQueue(m.ID, SendRequest{Text: "steer-me: also check y"})
	if err := r.SteerQueued(m.ID, r.Queued(m.ID)[0].ID); err != nil {
		t.Fatalf("SteerQueued = %v", err)
	}
	if q := r.Queued(m.ID); len(q) != 0 {
		t.Errorf("queue after the steer = %+v", q)
	}
	waitDone(t, r)

	_, events, _ := store.Get(m.ID)
	steers := eventsOfType(events, EventSteer)
	if len(steers) != 1 || steers[0].Text != "steer-me: also check y" {
		t.Fatalf("steer events = %+v", steers)
	}
	if n := len(eventsOfType(events, EventUserMessage)); n != 1 {
		t.Errorf("user_message events = %d, want 1", n)
	}
	if n := len(eventsOfType(events, EventTurnDone)); n != 1 {
		t.Errorf("turn_done events = %d, want 1", n)
	}
	if starts(t, dir) != 1 {
		t.Errorf("agent started %d times, want once", starts(t, dir))
	}
	// The fake reads the steer once its turn is over.
	var lines []string
	waitFor(t, "the fake to read the steer", func() bool {
		data, _ := os.ReadFile(filepath.Join(dir, "users.txt"))
		lines = strings.Split(strings.TrimSpace(string(data)), "\n")
		return len(lines) == 2
	})
	var msg struct {
		Type    string `json:"type"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &msg); err != nil || msg.Type != "user" || msg.Message.Content != "steer-me: also check y" {
		t.Errorf("steer line = %q (%v)", lines[1], err)
	}
}

func TestRunner_SteerWithoutATurn(t *testing.T) {
	r, store, _ := newTestRunner(t, `case "$line" in *slow*) head -n 3 "$FIXTURE"; sleep 30 ;; *) cat "$FIXTURE" ;; esac`)
	m, _ := store.Create()
	if err := r.SteerQueued(m.ID, "nope"); !errors.Is(err, ErrQueuedNotFound) {
		t.Fatalf("SteerQueued of an unknown message = %v, want ErrQueuedNotFound", err)
	}
	ch := tap(r, m.ID)
	r.Send(m.ID, SendRequest{Text: "slow"})
	waitForEvent(t, ch, EventTextDelta)
	r.SendOrQueue(m.ID, SendRequest{Text: "y"})
	r.Cancel(m.ID)
	waitDone(t, r)

	id := r.Queued(m.ID)[0].ID
	if err := r.SteerQueued(m.ID, id); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("SteerQueued after the turn = %v, want ErrNotRunning", err)
	}
	if q := r.Queued(m.ID); len(q) != 1 {
		t.Errorf("a refused steer took the message: %+v", q)
	}
	if _, events, _ := store.Get(m.ID); len(eventsOfType(events, EventSteer)) != 0 {
		t.Errorf("a refused steer was logged: %+v", events)
	}
}

// codexSteerRunner runs a Codex turn that stays open until a steer arrives;
// the fake then completes it.
func codexSteerRunner(t *testing.T) (*Runner, *Store, string) {
	r, store, dir := newCodexRunner(t, "")
	turn := strings.SplitAfter(codexTurnFixture, "\n")
	os.WriteFile(filepath.Join(dir, "turn.jsonl"), []byte(strings.Join(turn[:7], "")), 0o644)
	os.WriteFile(filepath.Join(dir, "steer.jsonl"), []byte(turn[len(turn)-2]), 0o644)
	return r, store, dir
}

func TestCodexDriver_Steer(t *testing.T) {
	r, store, dir := codexSteerRunner(t)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	r.Send(m.ID, SendRequest{Text: "x", Provider: "codex"})
	waitForEvent(t, ch, EventTextDelta)

	r.SendOrQueue(m.ID, SendRequest{Text: "also check y", Provider: "codex"})
	if err := r.SteerQueued(m.ID, r.Queued(m.ID)[0].ID); err != nil {
		t.Fatalf("SteerQueued = %v", err)
	}
	waitDone(t, r)

	sent := sentRequests(t, dir)
	if len(sent["turn/steer"]) != 1 {
		t.Fatalf("turn/steer sent %d times", len(sent["turn/steer"]))
	}
	p := params(sent["turn/steer"][0])
	input, _ := json.Marshal(p["input"])
	if p["threadId"] != "thr-1" || p["expectedTurnId"] != "turn-1" || !strings.Contains(string(input), "also check y") {
		t.Errorf("turn/steer params = %v", p)
	}
	_, events, _ := store.Get(m.ID)
	if len(eventsOfType(events, EventSteer)) != 1 || len(eventsOfType(events, EventTurnDone)) != 1 {
		t.Errorf("events = %+v", events)
	}
}

func TestCodexDriver_SteerRefused(t *testing.T) {
	r, store, dir := codexSteerRunner(t)
	os.WriteFile(filepath.Join(dir, "steer-error"), nil, 0o644)
	m, _ := store.Create()
	ch := tap(r, m.ID)
	r.Send(m.ID, SendRequest{Text: "x", Provider: "codex"})
	waitForEvent(t, ch, EventTextDelta)

	r.SendOrQueue(m.ID, SendRequest{Text: "y", Provider: "codex"})
	if err := r.SteerQueued(m.ID, r.Queued(m.ID)[0].ID); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("SteerQueued = %v, want ErrNotRunning", err)
	}
	if len(r.Queued(m.ID)) != 1 {
		t.Errorf("a refused steer took the message")
	}
	r.Cancel(m.ID)
	waitDone(t, r)
	if _, events, _ := store.Get(m.ID); len(eventsOfType(events, EventSteer)) != 0 {
		t.Errorf("a refused steer was logged")
	}
}
