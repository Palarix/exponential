package chat

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// toolTurn is a Claude turn that calls one xpo tool with an "id" argument
// and gets a result, or an error result when failed.
func toolTurn(tool, issue string, failed bool) string {
	result := `{"type":"tool_result","content":"ok","tool_use_id":"t-1"}`
	if failed {
		result = `{"type":"tool_result","content":"no","is_error":true,"tool_use_id":"t-1"}`
	}
	return `{"type":"system","subtype":"init","session_id":"sess-7"}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t-1","name":"mcp__xpo__` + tool + `"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"id\":\"` + issue + `\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"user","message":{"role":"user","content":[` + result + `]}}
{"type":"result","subtype":"success","result":"ok","session_id":"sess-7","duration_ms":1,"usage":{}}
`
}

// newModeRunner returns a runner whose fake claude prints turn.jsonl from
// dir, with Work Mode enabled. Issues resolve to worktrees under dir.
func newModeRunner(t *testing.T) (*Runner, *Store, string) {
	t.Helper()
	r, store, dir := newTestRunner(t, `cat "$(dirname "$0")/turn.jsonl"`)
	r.cfg.CanWork = true
	r.cfg.ResolveWork = func(id string) (WorkBinding, error) {
		full := id
		if !strings.HasPrefix(id, "xpo-") {
			full = "xpo-" + id
		}
		wt := filepath.Join(dir, "worktrees", full)
		if _, err := os.Stat(wt); err != nil {
			return WorkBinding{}, fmt.Errorf("no worktree for %s", full)
		}
		return WorkBinding{IssueID: full, Worktree: wt}, nil
	}
	setTurn(t, dir, turnFixture)
	return r, store, dir
}

func setTurn(t *testing.T, dir, fixture string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "turn.jsonl"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeWorktree(t *testing.T, dir, issue string) string {
	t.Helper()
	wt := filepath.Join(dir, "worktrees", issue)
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	return wt
}

func modeEvents(events []Event) []Event {
	var out []Event
	for _, e := range events {
		if e.Type == EventModeChanged {
			out = append(out, e)
		}
	}
	return out
}

func lastArgs(t *testing.T, dir string) []string {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestRunner_NewChatRunsInChatMode(t *testing.T) {
	r, store, dir := newModeRunner(t)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi"})
	r.Wait()

	args := lastArgs(t, dir)
	if flagValue(args, "--tools") != chatTools || !slices.Contains(flagValues(args, "--disallowedTools"), "mcp__xpo__merge") {
		t.Errorf("args = %v, want the chat profile", args)
	}
	meta, _ := store.Meta(m.ID)
	if meta.Mode != "" && meta.Mode != ModeChat {
		t.Errorf("mode = %q", meta.Mode)
	}
}

func TestRunner_StartEntersWorkModeAfterTheTurn(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-abc")
	setTurn(t, dir, toolTurn("start", "abc", false))
	m, _ := store.Create()

	r.Send(m.ID, SendRequest{Text: "let's start working on xpo-abc"})
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	if meta.Mode != ModeWork || meta.Work == nil || meta.Work.IssueID != "xpo-abc" || meta.Work.Worktree != wt {
		t.Fatalf("meta = %+v, want Work Mode on xpo-abc", meta)
	}
	if meta.State != StateIdle {
		t.Errorf("state = %q, want idle after the continued turn", meta.State)
	}
	if !slices.Equal(meta.WorkedIssues, []string{"xpo-abc"}) {
		t.Errorf("worked issues = %v", meta.WorkedIssues)
	}
	// The mode changes after the turn, and the board continues in Work Mode
	// with no user message: turn_done, mode_changed, turn_continued, then
	// the continued turn.
	var types []string
	for _, e := range events {
		types = append(types, e.Type)
	}
	i := slices.Index(types, EventModeChanged)
	if i < 1 || types[i-1] != EventTurnDone || i+1 >= len(types) || types[i+1] != EventTurnContinued {
		t.Fatalf("events = %v, want turn_done, mode_changed, turn_continued", types)
	}
	if e := events[i]; e.Mode != ModeWork || e.IssueID != "xpo-abc" {
		t.Errorf("mode_changed = %+v", e)
	}
	if e := events[i+1]; e.IssueID != "xpo-abc" {
		t.Errorf("turn_continued = %+v, want issue xpo-abc", e)
	}
	if types[len(types)-1] != EventTurnDone {
		t.Errorf("events = %v, want the continued turn to finish", types)
	}
	if n := countType(events, EventUserMessage); n != 1 {
		t.Errorf("user messages = %d, want 1", n)
	}

	// The continued turn ran in the worktree with the work profile, and
	// resumed the same session.
	args := lastArgs(t, dir)
	if flagValue(args, "--tools") != workTools || flagValue(args, "--resume") != "sess-7" {
		t.Errorf("args = %v, want the work profile resuming sess-7", args)
	}
	realWT, _ := filepath.EvalSymlinks(wt)
	if pwd, _ := os.ReadFile(filepath.Join(dir, "pwd.txt")); strings.TrimSpace(string(pwd)) != realWT {
		t.Errorf("turn ran in %q, want the worktree %q", strings.TrimSpace(string(pwd)), realWT)
	}
	stdin, _ := os.ReadFile(filepath.Join(dir, "stdin.txt"))
	if !strings.Contains(string(stdin), "mode: work on xpo-abc") {
		t.Errorf("prompt = %s, want the work mode line", stdin)
	}
}

func countType(events []Event, typ string) int {
	n := 0
	for _, e := range events {
		if e.Type == typ {
			n++
		}
	}
	return n
}

func TestRunner_ContinuedTurnHoldsTheChat(t *testing.T) {
	r, store, dir := newModeRunner(t)
	makeWorktree(t, dir, "xpo-abc")
	// The Chat Mode turn starts the issue; the continued Work Mode turn
	// (the one with Edit) runs until it is stopped.
	r.cfg.Drivers["claude"] = &ClaudeDriver{Binary: fakeClaude(t, dir, `cat "$(dirname "$0")/turn.jsonl"; if grep -q Edit "$(dirname "$0")/args.txt"; then sleep 5; fi`)}
	setTurn(t, dir, toolTurn("start", "abc", false))
	m, _ := store.Create()
	events, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

	r.Send(m.ID, SendRequest{Text: "start xpo-abc"})
	for e := range events {
		if e.Type == EventTurnContinued {
			break
		}
	}
	if err := r.Send(m.ID, SendRequest{Text: "hello"}); !errors.Is(err, ErrBusy) {
		t.Errorf("send during the continued turn = %v, want ErrBusy", err)
	}
	if !r.Running(m.ID) {
		t.Error("the chat is not running during the continued turn")
	}
	r.Cancel(m.ID)
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.State != StateIdle {
		t.Errorf("state = %q, want idle", meta.State)
	}
}

func TestRunner_StoppedStartTurnDoesNotContinue(t *testing.T) {
	r, store, dir := newModeRunner(t)
	makeWorktree(t, dir, "xpo-abc")
	// The start call succeeds, then the turn runs on until it is stopped.
	r.cfg.Drivers["claude"] = &ClaudeDriver{Binary: fakeClaude(t, dir, `head -n 5 "$(dirname "$0")/turn.jsonl"; sleep 5`)}
	setTurn(t, dir, toolTurn("start", "abc", false))
	m, _ := store.Create()
	events, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

	r.Send(m.ID, SendRequest{Text: "start xpo-abc"})
	for e := range events {
		if e.Type == EventToolCall {
			break
		}
	}
	r.Cancel(m.ID)
	r.Wait()

	_, stored, _ := store.Get(m.ID)
	if n := countType(stored, EventTurnContinued); n != 0 {
		t.Errorf("events = %+v, want no continued turn after a stop", stored)
	}
}

func TestRunner_FailedStartKeepsChatMode(t *testing.T) {
	r, store, dir := newModeRunner(t)
	makeWorktree(t, dir, "xpo-abc")
	setTurn(t, dir, toolTurn("start", "xpo-abc", true))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "start"})
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	if meta.Mode == ModeWork || len(modeEvents(events)) != 0 || countType(events, EventTurnContinued) != 0 {
		t.Fatalf("meta = %+v, events = %+v; want Chat Mode and no continued turn", meta, events)
	}
}

func TestRunner_StartInWorkModeDoesNotRebind(t *testing.T) {
	r, store, dir := newModeRunner(t)
	makeWorktree(t, dir, "xpo-abc")
	makeWorktree(t, dir, "xpo-def")
	m, _ := store.Create()
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: filepath.Join(dir, "worktrees", "xpo-abc")}); err != nil {
		t.Fatal(err)
	}
	setTurn(t, dir, toolTurn("start", "xpo-def", false))
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.Work == nil || meta.Work.IssueID != "xpo-abc" {
		t.Fatalf("work = %+v, want still xpo-abc", meta.Work)
	}
}

func TestRunner_MergeLeavesWorkMode(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-abc")
	m, _ := store.Create()
	r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt})
	setTurn(t, dir, toolTurn("merge", "xpo-abc", false))

	r.Send(m.ID, SendRequest{Text: "merge it"})
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	if meta.Mode != ModeChat || meta.Work != nil {
		t.Fatalf("meta = %+v, want Chat Mode", meta)
	}
	if !slices.Equal(meta.WorkedIssues, []string{"xpo-abc"}) {
		t.Errorf("worked issues = %v, want the record kept", meta.WorkedIssues)
	}
	modes := modeEvents(events)
	if len(modes) != 2 || modes[1].Mode != ModeChat || modes[1].IssueID != "xpo-abc" {
		t.Errorf("mode events = %+v", modes)
	}
	if countType(events, EventTurnContinued) != 0 {
		t.Errorf("events = %+v, want no continued turn after a merge", events)
	}
}

func TestRunner_FailedMergeStaysInWorkMode(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-abc")
	m, _ := store.Create()
	r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt})
	setTurn(t, dir, toolTurn("merge", "xpo-abc", true))
	r.Send(m.ID, SendRequest{Text: "merge it"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.Mode != ModeWork {
		t.Fatalf("mode = %q, want work", meta.Mode)
	}
}

func TestRunner_BindWork(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-abc")
	a, _ := store.Create()
	b, _ := store.Create()

	meta, err := r.BindWork(a.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt})
	if err != nil || meta.Mode != ModeWork {
		t.Fatalf("bind = %+v, %v", meta, err)
	}
	if _, events, _ := store.Get(a.ID); len(modeEvents(events)) != 1 {
		t.Errorf("events = %+v, want one mode_changed", events)
	}
	// Binding again to the same issue is a no-op, not a conflict.
	if _, err := r.BindWork(a.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt}); err != nil {
		t.Errorf("rebind = %v", err)
	}
	if _, err := r.BindWork(b.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt}); !errors.Is(err, ErrIssueHeld) {
		t.Errorf("second chat bind = %v, want ErrIssueHeld", err)
	}
	if got, ok := r.WorkChat("xpo-abc"); !ok || got.ID != a.ID {
		t.Errorf("WorkChat = %+v, %v", got, ok)
	}
	if _, ok := r.WorkChat("xpo-zzz"); ok {
		t.Error("WorkChat found a chat for an unbound issue")
	}
}

// A chat in Work Mode on one issue cannot be bound to another: that would
// take the first issue's writer away.
func TestRunner_BindWorkRefusesChatBoundElsewhere(t *testing.T) {
	r, store, dir := newModeRunner(t)
	m, _ := store.Create()
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: makeWorktree(t, dir, "xpo-abc")}); err != nil {
		t.Fatal(err)
	}
	_, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-def", Worktree: makeWorktree(t, dir, "xpo-def")})
	if !errors.Is(err, ErrWorkingElsewhere) {
		t.Fatalf("bind = %v, want ErrWorkingElsewhere", err)
	}
	if meta, _ := store.Meta(m.ID); meta.Work == nil || meta.Work.IssueID != "xpo-abc" {
		t.Errorf("work = %+v, want still on xpo-abc", meta.Work)
	}
}

func TestRunner_BindWorkRefusedWithoutWork(t *testing.T) {
	r, store, dir := newModeRunner(t)
	r.cfg.CanWork = false
	m, _ := store.Create()
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: dir}); !errors.Is(err, ErrWorkUnavailable) {
		t.Fatalf("err = %v, want ErrWorkUnavailable", err)
	}
	r.Send(m.ID, SendRequest{Text: "hi"})
	r.Wait()
	if got := flagValues(lastArgs(t, dir), "--disallowedTools"); !slices.Contains(got, "mcp__xpo__start") {
		t.Errorf("--disallowedTools = %v, want start denied", got)
	}
}

func TestRunner_BindWorkRefusedWhileRunning(t *testing.T) {
	r, store, dir := newModeRunner(t)
	setTurn(t, dir, "")
	r.cfg.Drivers["claude"] = &ClaudeDriver{Binary: fakeClaude(t, dir, "sleep 5")}
	wt := makeWorktree(t, dir, "xpo-abc")
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt}); !errors.Is(err, ErrBusy) {
		t.Errorf("bind = %v, want ErrBusy", err)
	}
	if _, err := r.LeaveWork(m.ID); !errors.Is(err, ErrBusy) {
		t.Errorf("leave = %v, want ErrBusy", err)
	}
	r.CancelAndWait(m.ID)
}

func TestRunner_LeaveWork(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-abc")
	m, _ := store.Create()
	if _, err := r.LeaveWork(m.ID); !errors.Is(err, ErrNotInWork) {
		t.Errorf("leave in chat mode = %v, want ErrNotInWork", err)
	}
	r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt})
	meta, err := r.LeaveWork(m.ID)
	if err != nil || meta.Mode != ModeChat || meta.Work != nil {
		t.Fatalf("leave = %+v, %v", meta, err)
	}
	// The worktree stays: leaving is not a merge.
	if _, err := os.Stat(wt); err != nil {
		t.Errorf("worktree gone: %v", err)
	}
	_, events, _ := store.Get(m.ID)
	if modes := modeEvents(events); len(modes) != 2 || modes[1].Mode != ModeChat {
		t.Errorf("mode events = %+v", modes)
	}
	// Another chat may now take the issue.
	other, _ := store.Create()
	if _, err := r.BindWork(other.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt}); err != nil {
		t.Errorf("bind after leave = %v", err)
	}
}

func TestRunner_MissingWorktreeFallsBackToChatMode(t *testing.T) {
	r, store, dir := newModeRunner(t)
	wt := makeWorktree(t, dir, "xpo-abc")
	m, _ := store.Create()
	r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt})
	os.RemoveAll(wt)

	r.Send(m.ID, SendRequest{Text: "go on"})
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	if meta.Mode != ModeChat || meta.Work != nil {
		t.Fatalf("meta = %+v, want Chat Mode", meta)
	}
	if flagValue(lastArgs(t, dir), "--tools") != chatTools {
		t.Errorf("turn ran with %v, want the chat profile", lastArgs(t, dir))
	}
	// The divider comes before the user message of the turn it affects.
	i := slices.IndexFunc(events, func(e Event) bool { return e.Type == EventModeChanged && e.Mode == ModeChat })
	if i < 0 || events[i+1].Type != EventUserMessage || events[i].Message == "" {
		t.Errorf("events = %+v, want a mode_changed with a reason before the message", events)
	}
}

func TestRunner_OldChatWithoutModeIsChatMode(t *testing.T) {
	r, store, dir := newModeRunner(t)
	m, _ := store.Create()
	m.Mode = ""
	store.SaveMeta(m)
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	if flagValue(lastArgs(t, dir), "--tools") != chatTools {
		t.Errorf("args = %v", lastArgs(t, dir))
	}
}
