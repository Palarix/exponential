//go:build !windows

package chat

import (
	"os"
	"os/exec"
	"path/filepath"
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
	r := NewRunner(RunnerConfig{Store: store, Drivers: map[string]Driver{ProviderClaude: &ClaudeDriver{}}, Dir: dir, MCP: MCPTarget{Executable: "/nonexistent/xpo"}, CanWork: true})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()
	// Work Mode allows Bash without a card.
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt}); err != nil {
		t.Fatal(err)
	}
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

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
	r := NewRunner(RunnerConfig{Store: store, Drivers: map[string]Driver{ProviderCodex: &CodexDriver{}}, DefaultProvider: ProviderCodex, Dir: dir, MCP: MCPTarget{Executable: "/nonexistent/xpo"}, CanWork: true})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt}); err != nil {
		t.Fatal(err)
	}
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

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
