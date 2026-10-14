//go:build !windows

package chat

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Claude Code lines for a background shell, as Claude Code 2.1.299 prints
// them (trimmed).
const (
	claudeTaskStarted   = `{"type":"system","subtype":"background_tasks_changed","tasks":[{"task_id":"bg-1","task_type":"local_bash","description":"Serve the board"}]}`
	claudeTaskEnded     = `{"type":"system","subtype":"task_notification","task_id":"bg-1","status":"completed","summary":"Serve the board finished"}`
	claudeTasksNone     = `{"type":"system","subtype":"background_tasks_changed","tasks":[]}`
	claudeAgentTurnInit = `{"type":"system","subtype":"init","session_id":"sess-42","model":"claude-haiku-5-5"}`
	claudeAgentText     = `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"The server stopped."}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`
)

// printLines is a shell command that prints each line.
func printLines(lines ...string) string {
	var b strings.Builder
	for _, l := range lines {
		for _, one := range strings.Split(l, "\n") {
			b.WriteString(`printf '%s\n' '` + one + `'` + "\n")
		}
	}
	return b.String()
}

func alive(pid int) bool { return pid > 0 && syscall.Kill(pid, 0) == nil }

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func hasSession(r *Runner, id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[id] != nil
}

// The regression of xpo-fb4f9b: a child of the agent that outlives the
// result, and keeps stdout open, must not keep the turn running.
func TestRunner_BackgroundChildDoesNotHoldTheTurn(t *testing.T) {
	r, store, dir := newTestRunner(t, `sleep 30 &
cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "start the server"})
	waitDone(t, r)

	_, events, _ := store.Get(m.ID)
	if last := events[len(events)-1]; last.Type != EventTurnDone {
		t.Fatalf("last event = %+v, want turn_done", last)
	}
	if r.Running(m.ID) {
		t.Fatal("Running = true after the reply")
	}
	// The composer can send: the next message is a turn of its own.
	if err := r.Send(m.ID, SendRequest{Text: "and now?"}); err != nil {
		t.Fatalf("second Send: %v", err)
	}
	waitDone(t, r)
	if n := starts(t, dir); n != 1 {
		t.Errorf("agent started %d times, want once", n)
	}
}

func TestRunner_BackgroundTasksAndAgentTurn(t *testing.T) {
	r, store, _ := newTestRunner(t, printLines(claudeAgentTurnInit, claudeTaskStarted, claudeResultLine)+
		"( sleep 0.5\n"+printLines(claudeTaskEnded, claudeTasksNone, claudeAgentTurnInit, claudeAgentText, claudeResultLine)+") &")
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "serve it"})

	bg := waitForEvent(t, ch, EventBackgroundTasks)
	if len(bg.Tasks) != 1 || bg.Tasks[0] != (BackgroundTask{ID: "bg-1", Kind: TaskMonitor, Description: "Serve the board"}) {
		t.Fatalf("background_tasks = %+v", bg.Tasks)
	}
	waitForEvent(t, ch, EventTurnDone)
	if tasks := r.Background(m.ID); len(tasks) != 1 {
		t.Errorf("Background after the turn = %+v, want the server", tasks)
	}

	// The task ends; the agent replies on its own.
	if bg := waitForEvent(t, ch, EventBackgroundTasks); len(bg.Tasks) != 0 {
		t.Errorf("background_tasks after the task ended = %+v, want none", bg.Tasks)
	}
	agent := waitForEvent(t, ch, EventAgentTurn)
	if agent.Text != "Serve the board finished" {
		t.Errorf("agent_turn text = %q", agent.Text)
	}
	waitForEvent(t, ch, EventTurnDone)
	waitDone(t, r)

	meta, events, _ := store.Get(m.ID)
	var types []string
	for _, e := range events {
		types = append(types, e.Type)
	}
	want := []string{EventUserMessage, EventTurnDone, EventAgentTurn, EventAssistantText, EventTurnDone}
	if !slices.Equal(types, want) {
		t.Fatalf("events = %v, want %v", types, want)
	}
	if events[3].Text != "The server stopped." {
		t.Errorf("agent turn text = %q", events[3].Text)
	}
	if meta.State != StateIdle {
		t.Errorf("state = %s", meta.State)
	}
	if tasks := r.Background(m.ID); len(tasks) != 0 {
		t.Errorf("Background = %+v, want none", tasks)
	}
}

func TestRunner_ModelAndEffortSwitchInPlace(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one", Model: "haiku", Effort: "low"})
	waitDone(t, r)
	r.Send(m.ID, SendRequest{Text: "two", Model: "sonnet", Effort: "high"})
	waitDone(t, r)
	r.Send(m.ID, SendRequest{Text: "three"})
	waitDone(t, r)

	if n := starts(t, dir); n != 1 {
		t.Fatalf("agent started %d times, want once", n)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "control.txt"))
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var msg struct {
			Request map[string]any `json:"request"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatal(err)
		}
		req := msg.Request
		switch req["subtype"] {
		case "set_model":
			got = append(got, "model="+anyString(req["model"]))
		case "apply_flag_settings":
			got = append(got, "effort="+anyString(req["settings"].(map[string]any)["effortLevel"]))
		}
	}
	want := []string{"model=sonnet", "effort=high", "model=<nil>", "effort=<nil>"}
	if !slices.Equal(got, want) {
		t.Errorf("control requests = %v, want %v", got, want)
	}
	_, events, _ := store.Get(m.ID)
	if n := len(eventsOfType(events, EventTurnDone)); n != 3 {
		t.Errorf("turn_done events = %d, want 3", n)
	}
}

func anyString(v any) string {
	if v == nil {
		return "<nil>"
	}
	return v.(string)
}

// Stop interrupts the turn; the agent, and its background tasks, stay.
func TestRunner_CancelKeepsTheAgent(t *testing.T) {
	r, store, dir := newTestRunner(t, `head -n 3 "$FIXTURE"
IFS= read -r interrupt; printf '%s\n' "$interrupt" >> "$(dirname "$0")/interrupts.txt"
printf '%s\n' '{"type":"result","subtype":"error_during_execution","is_error":true,"session_id":"sess-42"}'`)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "x"})
	waitForEvent(t, ch, EventTextDelta)
	r.Cancel(m.ID)
	waitDone(t, r)

	_, events, _ := store.Get(m.ID)
	if last := events[len(events)-1]; last.Type != EventTurnStopped {
		t.Fatalf("last event = %+v", last)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "interrupts.txt"))
	if !strings.Contains(string(data), `"subtype":"interrupt"`) {
		t.Errorf("the agent got %q, want an interrupt request", data)
	}
	if !hasSession(r, m.ID) {
		t.Fatal("the agent ended with the stopped turn")
	}
}

func TestRunner_StopBackgroundEndsTheAgent(t *testing.T) {
	r, store, _ := newTestRunner(t, printLines(claudeAgentTurnInit, claudeTaskStarted, claudeResultLine))
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "serve it"})
	waitDone(t, r)
	meta, _ := store.Meta(m.ID)
	pid := meta.AgentPID
	if !alive(pid) || meta.AgentStarted == nil {
		t.Fatalf("meta agent = %d %v, want the live agent", pid, meta.AgentStarted)
	}
	waitForEvent(t, ch, EventBackgroundTasks)

	if !r.StopBackground(m.ID) {
		t.Fatal("StopBackground = false")
	}
	if bg := waitForEvent(t, ch, EventBackgroundTasks); len(bg.Tasks) != 0 {
		t.Errorf("background_tasks = %+v, want none", bg.Tasks)
	}
	waitFor(t, "the agent to exit", func() bool { return !alive(pid) })
	if meta, _ := store.Meta(m.ID); meta.AgentPID != 0 || meta.AgentStarted != nil {
		t.Errorf("meta agent = %d %v, want cleared", meta.AgentPID, meta.AgentStarted)
	}
	if r.StopBackground(m.ID) {
		t.Error("StopBackground = true without an agent")
	}
}

func TestRunner_LeaveWorkStopsBackgroundTasksInOneLine(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-1")
	setTurn(t, dir, claudeAgentTurnInit+"\n"+claudeTaskStarted+"\n"+claudeResultLine+"\n")
	m, _ := store.Create()
	r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt})
	r.Send(m.ID, SendRequest{Text: "serve it"})
	waitDone(t, r)
	if len(r.Background(m.ID)) != 1 {
		t.Fatal("no background task")
	}

	if _, err := r.LeaveWork(m.ID); err != nil {
		t.Fatal(err)
	}
	_, events, _ := store.Get(m.ID)
	modes := modeEvents(events)
	last := modes[len(modes)-1]
	if last.Mode != ModeChat || last.Message != "Entered Chat Mode. Background tasks stopped." {
		t.Errorf("mode_changed = %+v", last)
	}
	waitFor(t, "the agent to close", func() bool { return !hasSession(r, m.ID) })

	// Without background tasks, the line stays as it was.
	r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt})
	setTurn(t, dir, turnFixture)
	r.Send(m.ID, SendRequest{Text: "x"})
	waitDone(t, r)
	r.LeaveWork(m.ID)
	_, events, _ = store.Get(m.ID)
	modes = modeEvents(events)
	if last := modes[len(modes)-1]; last.Message != "" {
		t.Errorf("mode_changed message = %q, want none", last.Message)
	}
}

func TestRunner_ModeChangeRestartsTheAgentInTheWorktree(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-1")
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	waitDone(t, r)
	r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt})
	r.Send(m.ID, SendRequest{Text: "two"})
	waitDone(t, r)

	if n := starts(t, dir); n != 2 {
		t.Fatalf("agent started %d times, want twice", n)
	}
	pwd, _ := os.ReadFile(filepath.Join(dir, "pwd.txt"))
	real, _ := filepath.EvalSymlinks(wt)
	if strings.TrimSpace(string(pwd)) != real {
		t.Errorf("agent runs in %s, want %s", pwd, real)
	}
}

// fastReaper closes agents idle for idle, and looks often.
func fastReaper(idle time.Duration) func(*RunnerConfig) {
	return func(c *RunnerConfig) {
		c.IdleTimeout = idle
		c.ReapEvery = idle / 5
	}
}

func TestRunner_IdleAgentIsClosed(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, fastReaper(100*time.Millisecond))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	waitDone(t, r)
	waitFor(t, "the idle agent to close", func() bool { return !hasSession(r, m.ID) })
}

func TestRunner_IdleAgentWithTasksStays(t *testing.T) {
	r, store, _ := newTestRunner(t, printLines(claudeAgentTurnInit, claudeTaskStarted, claudeResultLine), fastReaper(50*time.Millisecond))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "serve it"})
	waitDone(t, r)
	time.Sleep(200 * time.Millisecond)
	if !hasSession(r, m.ID) {
		t.Fatal("the reaper closed an agent with a background task")
	}
}

func TestRunner_ShutdownStopsAgents(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	waitDone(t, r)
	meta, _ := store.Meta(m.ID)
	if !alive(meta.AgentPID) {
		t.Fatal("no live agent")
	}
	r.Shutdown()
	waitFor(t, "the agent to exit", func() bool { return !alive(meta.AgentPID) })
}

func TestStore_StopOrphans(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "chats"))
	start := func() (*exec.Cmd, time.Time) {
		cmd := exec.Command("sleep", "30")
		setProcessGroup(cmd)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { killGroup(cmd.Process.Pid); _ = cmd.Wait() })
		return cmd, time.Now().UTC()
	}
	orphan, started := start()
	m, _ := store.Create()
	m.AgentPID, m.AgentStarted = orphan.Process.Pid, &started
	store.SaveMeta(m)

	// A process that reuses a recorded PID started at another time.
	other, _ := start()
	long := time.Now().Add(-time.Hour).UTC()
	m2, _ := store.Create()
	m2.AgentPID, m2.AgentStarted = other.Process.Pid, &long
	store.SaveMeta(m2)

	n, err := store.StopOrphans()
	if err != nil || n != 1 {
		t.Fatalf("StopOrphans = %d, %v; want 1", n, err)
	}
	done := make(chan error, 1)
	go func() { done <- orphan.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the orphan still runs")
	}
	if !alive(other.Process.Pid) {
		t.Error("stopped a process that only reuses the PID")
	}
	for _, id := range []string{m.ID, m2.ID} {
		if meta, _ := store.Meta(id); meta.AgentPID != 0 || meta.AgentStarted != nil {
			t.Errorf("meta %s agent = %d %v, want cleared", id, meta.AgentPID, meta.AgentStarted)
		}
	}
}

// A Codex command still in progress at turn/completed is a background task
// until its item completes.
func TestCodexDriver_BackgroundCommand(t *testing.T) {
	r, store, dir := newCodexRunner(t, `sleep 0.5; printf '%s\n' '{"method":"item/completed","params":{"item":{"type":"commandExecution","id":"exec-bg","command":"/bin/zsh -lc '"'"'npm run dev'"'"'","status":"completed","commandActions":[{"type":"unknown","command":"npm run dev"}],"aggregatedOutput":"","exitCode":0},"threadId":"thr-1","turnId":"turn-1"}}'`)
	os.WriteFile(filepath.Join(dir, "turn.jsonl"), []byte(`{"id":3,"result":{"turn":{"id":"turn-1","status":"inProgress","items":[]}}}
{"method":"turn/started","params":{"threadId":"thr-1","turn":{"id":"turn-1","status":"inProgress"}}}
{"method":"item/started","params":{"item":{"type":"commandExecution","id":"exec-bg","command":"/bin/zsh -lc 'npm run dev'","status":"inProgress","commandActions":[{"type":"unknown","command":"npm run dev"}],"aggregatedOutput":null,"exitCode":null},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"agentMessage","id":"m-1","text":"OK","phase":"final_answer"},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"turn/completed","params":{"threadId":"thr-1","turn":{"id":"turn-1","status":"completed","error":null,"durationMs":5}}}
`), 0o644)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "serve it", Provider: "codex"})

	bg := waitForEvent(t, ch, EventBackgroundTasks)
	if len(bg.Tasks) != 1 || bg.Tasks[0] != (BackgroundTask{ID: "exec-bg", Kind: TaskMonitor, Description: "npm run dev"}) {
		t.Fatalf("background_tasks = %+v", bg.Tasks)
	}
	waitDone(t, r)
	if bg := waitForEvent(t, ch, EventBackgroundTasks); len(bg.Tasks) != 0 {
		t.Errorf("background_tasks after the command ended = %+v", bg.Tasks)
	}
	_, events, _ := store.Get(m.ID)
	if n := len(eventsOfType(events, EventAgentTurn)); n != 0 {
		t.Errorf("agent turns = %d; Codex does not start turns on its own", n)
	}
}
