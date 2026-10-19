//go:build !windows

package chat

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRealClaude_BackgroundTask runs the real `claude` CLI, so it calls a
// model. It pins what xpo-fb4f9b relies on: the turn ends at `result` while
// a background shell runs, the task shows as a background task, the agent
// starts a turn of its own when the task ends, and a model switch happens in
// place. Run it with XPO_TEST_REAL_CLI=1 after a Claude Code upgrade.
func TestRealClaude_BackgroundTask(t *testing.T) {
	if os.Getenv("XPO_TEST_REAL_CLI") != "1" {
		t.Skip("set XPO_TEST_REAL_CLI=1 to run against the real claude CLI")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude is not installed")
	}
	dir := t.TempDir()
	wt := filepath.Join(dir, "wt")
	os.Mkdir(wt, 0o755)
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{Store: store, Drivers: map[string]Driver{ProviderClaude: &ClaudeDriver{}}, Dir: dir, MCP: MCPTarget{Executable: "/nonexistent/xpo"}})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()
	// Work Mode allows Bash without a card.
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt}); err != nil {
		t.Fatal(err)
	}
	ch := tap(r, m.ID)

	start := time.Now()
	r.Send(m.ID, SendRequest{Text: "Use the Bash tool with run_in_background=true to run: sleep 20. Do not wait for it. Reply with just OK.", Model: "haiku"})
	waitForEventWithin(t, ch, EventTurnDone, 90*time.Second)
	if d := time.Since(start); d > 60*time.Second {
		t.Errorf("the turn took %s; it must end at the reply, not when the task ends", d)
	}
	if tasks := r.Background(m.ID); len(tasks) != 1 || tasks[0].Kind != TaskMonitor {
		t.Fatalf("Background = %+v, want one monitor task", tasks)
	}

	// The model switches in place: the next turn runs on the same process.
	pid := mustMeta(t, store, m.ID).AgentPID
	r.Send(m.ID, SendRequest{Text: "Reply with just the number 4.", Model: "sonnet"})
	waitForEventWithin(t, ch, EventTurnDone, 90*time.Second)
	waitFor(t, "the turn to end", func() bool { return !r.Running(m.ID) }) // the meta is saved after turn_done
	if meta := mustMeta(t, store, m.ID); meta.AgentPID != pid || meta.ResolvedModel == "" || meta.ResolvedModel == "claude-haiku-5-5" {
		t.Errorf("after the switch: pid %d (was %d), model %q", meta.AgentPID, pid, meta.ResolvedModel)
	}

	// The task ends, and the agent replies on its own.
	waitForEventWithin(t, ch, EventAgentTurn, 90*time.Second)
	waitForEventWithin(t, ch, EventTurnDone, 90*time.Second)
	if tasks := r.Background(m.ID); len(tasks) != 0 {
		t.Errorf("Background after the task ended = %+v", tasks)
	}
}

// TestRealClaude_EditRewinds runs the real `claude` CLI, so it calls a
// model. It pins what xpo-52ec1f relies on: an assistant message UUID from
// stream-json is a valid --resume-session-at point, and the edited turn is
// gone from the session. Run it with XPO_TEST_REAL_CLI=1 after a Claude Code
// upgrade.
func TestRealClaude_EditRewinds(t *testing.T) {
	testRealEditRewinds(t, ProviderClaude, &ClaudeDriver{}, "haiku")
}

// TestRealCodex_EditRewinds is TestRealClaude_EditRewinds for Codex: a turn
// ID is a valid thread/revert point.
func TestRealCodex_EditRewinds(t *testing.T) {
	testRealEditRewinds(t, ProviderCodex, &CodexDriver{}, "")
}

func testRealEditRewinds(t *testing.T, provider string, driver Driver, model string) {
	if os.Getenv("XPO_TEST_REAL_CLI") != "1" {
		t.Skip("set XPO_TEST_REAL_CLI=1 to run against the real " + provider + " CLI")
	}
	if _, err := exec.LookPath(provider); err != nil {
		t.Skip(provider + " is not installed")
	}
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{Store: store, Drivers: map[string]Driver{provider: driver}, DefaultProvider: provider, Dir: dir, MCP: MCPTarget{Executable: "/nonexistent/xpo"}})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()

	for _, text := range []string{
		"Remember the word APPLE. Reply with just OK.",
		"Now forget that: remember the word PEAR instead. Reply with just OK.",
	} {
		if err := r.Send(m.ID, SendRequest{Text: text, Provider: provider, Model: model}); err != nil {
			t.Fatal(err)
		}
		r.Wait()
	}
	if err := r.Send(m.ID, SendRequest{Text: "Which word did I ask you to remember? Reply with just the word.", Provider: provider, Model: model, Edit: true}); err != nil {
		t.Fatal(err)
	}
	r.Wait()

	_, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventTurnDone {
		t.Fatalf("last event = %+v", last)
	}
	texts := eventsOfType(activeEvents(events), EventAssistantText)
	reply := texts[len(texts)-1].Text
	if !strings.Contains(reply, "APPLE") || strings.Contains(reply, "PEAR") {
		t.Errorf("reply after the edit = %q, want APPLE: the PEAR turn must be gone", reply)
	}
}

// TestRealClaude_Steer and TestRealCodex_Steer run the real CLIs, so they
// call a model. They pin that a steer reaches the running turn: the agent
// follows it in the same turn.
func TestRealClaude_Steer(t *testing.T) { testRealSteer(t, ProviderClaude, &ClaudeDriver{}, "haiku") }
func TestRealCodex_Steer(t *testing.T)  { testRealSteer(t, ProviderCodex, &CodexDriver{}, "") }

func testRealSteer(t *testing.T, provider string, driver Driver, model string) {
	if os.Getenv("XPO_TEST_REAL_CLI") != "1" {
		t.Skip("set XPO_TEST_REAL_CLI=1 to run against the real " + provider + " CLI")
	}
	if _, err := exec.LookPath(provider); err != nil {
		t.Skip(provider + " is not installed")
	}
	dir := t.TempDir()
	wt := filepath.Join(dir, "wt")
	os.Mkdir(wt, 0o755)
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{Store: store, Drivers: map[string]Driver{provider: driver}, DefaultProvider: provider, Dir: dir, MCP: MCPTarget{Executable: "/nonexistent/xpo"}})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()
	// Work Mode runs shell commands without a card.
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt}); err != nil {
		t.Fatal(err)
	}
	ch := tap(r, m.ID)
	r.Send(m.ID, SendRequest{Text: "Run the shell command `sleep 8 && echo done` in the foreground and wait for it. Then tell me its output.", Provider: provider, Model: model})
	waitForEventWithin(t, ch, EventToolStart, 90*time.Second)
	if queued, err := r.SendOrQueue(m.ID, SendRequest{Text: "Also: end your reply with the word BANANA.", Provider: provider, Model: model}); err != nil || !queued {
		t.Fatalf("SendOrQueue = %v, %v; want it queued", queued, err)
	}
	if err := r.SteerQueued(m.ID, r.Queued(m.ID)[0].ID); err != nil {
		t.Fatalf("SteerQueued = %v", err)
	}
	waitForEventWithin(t, ch, EventTurnDone, 120*time.Second)
	r.Wait()

	_, events, _ := store.Get(m.ID)
	if n := len(eventsOfType(events, EventUserMessage)); n != 1 {
		t.Errorf("user messages = %d, want 1", n)
	}
	texts := eventsOfType(events, EventAssistantText)
	if len(texts) == 0 || !strings.Contains(texts[len(texts)-1].Text, "BANANA") {
		t.Errorf("the turn did not follow the steer: %+v", texts)
	}
}

func mustMeta(t *testing.T, store *Store, id string) Meta {
	t.Helper()
	m, err := store.Meta(id)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// waitForEventWithin returns the first event of type typ on ch.
func waitForEventWithin(t *testing.T, ch <-chan Event, typ string, d time.Duration) Event {
	t.Helper()
	timeout := time.After(d)
	for {
		select {
		case e := <-ch:
			if e.Type == typ {
				return e
			}
		case <-timeout:
			t.Fatalf("no %s event within %s", typ, d)
		}
	}
}

// TestRealCodex_BackgroundCommand runs the real `codex` CLI, so it calls a
// model. It pins that a command still running at turn/completed shows as a
// background task until it ends (codex-cli 0.162.1). Run it with
// XPO_TEST_REAL_CLI=1 after a Codex upgrade.
func TestRealCodex_BackgroundCommand(t *testing.T) {
	if os.Getenv("XPO_TEST_REAL_CLI") != "1" {
		t.Skip("set XPO_TEST_REAL_CLI=1 to run against the real codex CLI")
	}
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex is not installed")
	}
	dir := t.TempDir()
	wt := filepath.Join(dir, "wt")
	os.Mkdir(wt, 0o755)
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{Store: store, Drivers: map[string]Driver{ProviderCodex: &CodexDriver{}}, DefaultProvider: ProviderCodex, Dir: dir, MCP: MCPTarget{Executable: "/nonexistent/xpo"}})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt}); err != nil {
		t.Fatal(err)
	}
	ch := tap(r, m.ID)

	r.Send(m.ID, SendRequest{Text: "Start the shell command `sleep 25 && echo finished` as a long-running command without nohup or &. Do not wait for it: leave it running and reply with just OK right away.", Provider: ProviderCodex})
	waitForEventWithin(t, ch, EventTurnDone, 120*time.Second)
	waitFor(t, "the turn to end", func() bool { return !r.Running(m.ID) })
	if tasks := r.Background(m.ID); len(tasks) != 1 || tasks[0].Kind != TaskMonitor {
		t.Fatalf("Background = %+v, want the sleep", tasks)
	}
	bg := waitForEventWithin(t, ch, EventBackgroundTasks, 60*time.Second)
	if len(bg.Tasks) != 0 {
		t.Errorf("background_tasks after the command ended = %+v", bg.Tasks)
	}
}
