package chat

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

const turnFixture = `{"type":"system","subtype":"init","session_id":"sess-42","model":"claude-haiku-5-5"}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Checking "}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"issues."}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","name":"mcp__xpo__list"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"match\":\"login\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":1}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"You have 1 open issue."}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"result","subtype":"success","result":"You have 1 open issue.","session_id":"sess-42","duration_ms":800,"total_cost_usd":0.01,"usage":{"input_tokens":3,"output_tokens":7},"modelUsage":{"claude-haiku-5-5":{"contextWindow":200000}}}
`

// fakeClaude writes a shell script that records its args and stdin in dir,
// then runs body.
func fakeClaude(t *testing.T, dir, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake claude is a shell script")
	}
	script := filepath.Join(dir, "claude")
	content := "#!/bin/sh\n" +
		`for a in "$@"; do printf '%s\n' "$a"; done > "` + dir + `/args.txt"` + "\n" +
		`cat > "` + dir + `/stdin.txt"` + "\n" +
		body + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func newTestRunner(t *testing.T, body string) (*Runner, *Store, string) {
	t.Helper()
	dir := t.TempDir()
	fixture := filepath.Join(dir, "fixture.jsonl")
	os.WriteFile(fixture, []byte(turnFixture), 0o644)
	body = strings.ReplaceAll(body, "$FIXTURE", fixture)

	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{
		Store:   store,
		Drivers: map[string]Driver{"claude": &ClaudeDriver{Binary: fakeClaude(t, dir, body)}},
		Dir:     dir,
		MCP:     MCPTarget{Executable: "/bin/xpo"},
	})
	t.Cleanup(r.Shutdown)
	return r, store, dir
}

// learned records what the runner teaches the catalog.
type learned struct{ calls []string }

func (l *learned) Learn(provider, picked, resolved string, window int) {
	l.calls = append(l.calls, fmt.Sprintf("%s|%s|%s|%d", provider, picked, resolved, window))
}

func TestRunner_RecordsResolvedModel(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	l := &learned{}
	r.cfg.Learner = l
	m, _ := store.Create()

	if err := r.Send(m.ID, SendRequest{Text: "hi", Model: "haiku"}); err != nil {
		t.Fatal(err)
	}
	r.Wait()

	meta, _ := store.Meta(m.ID)
	if meta.ResolvedModel != "claude-haiku-5-5" || meta.ContextWindow != 200000 {
		t.Errorf("meta = %+v, want the resolved model and its context window", meta)
	}
	if want := []string{"claude|haiku|claude-haiku-5-5|200000"}; !slices.Equal(l.calls, want) {
		t.Errorf("learned %v, want %v", l.calls, want)
	}
}

func TestRunner_FailedTurnLearnsNothing(t *testing.T) {
	r, store, _ := newTestRunner(t, `echo '{"type":"system","subtype":"init","session_id":"s","model":"claude-haiku-5-5"}'; echo '{"type":"result","is_error":true,"result":"Not logged in","session_id":"s"}'`)
	l := &learned{}
	r.cfg.Learner = l
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi"})
	r.Wait()

	if meta, _ := store.Meta(m.ID); meta.ResolvedModel != "" || len(l.calls) != 0 {
		t.Errorf("meta = %+v, learned %v; want nothing from a failed turn", meta, l.calls)
	}
}

func TestRunner_TurnPersistsSettledEvents(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()

	if err := r.Send(m.ID, SendRequest{Text: "how many?", Context: &Context{IssueID: "xpo-1"}, Model: "haiku", Effort: "low"}); err != nil {
		t.Fatal(err)
	}
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	types := make([]string, len(events))
	for i, e := range events {
		types[i] = e.Type
	}
	want := []string{EventUserMessage, EventAssistantText, EventToolCall, EventAssistantText, EventTurnDone}
	if !slices.Equal(types, want) {
		t.Fatalf("event types = %v, want %v", types, want)
	}
	if events[1].Text != "Checking issues." || events[3].Text != "You have 1 open issue." {
		t.Errorf("assistant text = %q / %q", events[1].Text, events[3].Text)
	}
	if events[2].Name != "mcp__xpo__list" || events[2].Arg != "login" || events[2].Status != "done" {
		t.Errorf("tool call = %+v", events[2])
	}
	if events[4].DurationMS != 800 || events[4].TokensOut != 7 || events[4].CostUSD != 0.01 {
		t.Errorf("turn done = %+v", events[4])
	}

	if meta.SessionID != "sess-42" || meta.State != StateIdle || meta.Title != "how many?" {
		t.Errorf("meta = %+v", meta)
	}
	if meta.Model != "haiku" || meta.Effort != "low" {
		t.Errorf("meta model/effort = %q/%q", meta.Model, meta.Effort)
	}

	stdin, _ := os.ReadFile(filepath.Join(dir, "stdin.txt"))
	if !strings.Contains(string(stdin), "open issue: xpo-1") || !strings.HasSuffix(string(stdin), "how many?") {
		t.Errorf("stdin = %q", stdin)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), "--model\nhaiku\n") || !strings.Contains(string(args), "--effort\nlow\n") {
		t.Errorf("args = %s", args)
	}
}

func TestRunner_SecondTurnResumes(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	r.Send(m.ID, SendRequest{Text: "two"})
	r.Wait()

	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), "--resume\nsess-42\n") {
		t.Fatalf("second turn args = %s", args)
	}
	meta, _, _ := store.Get(m.ID)
	if meta.Title != "one" {
		t.Errorf("title changed to %q", meta.Title)
	}
}

func TestRunner_MCPConfigFileRemovedAfterTurn(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()

	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	lines := strings.Split(string(args), "\n")
	path := lines[slices.Index(lines, "--mcp-config")+1]
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("mcp config %s still exists (err=%v)", path, err)
	}
}

func TestRunner_PublishesLiveAndSettledEvents(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()

	var types []string
	for {
		select {
		case e := <-ch:
			types = append(types, e.Type)
			continue
		default:
		}
		break
	}
	for _, want := range []string{EventUserMessage, EventTextDelta, EventToolStart, EventToolCall, EventTurnDone} {
		if !slices.Contains(types, want) {
			t.Errorf("published events %v missing %s", types, want)
		}
	}
}

func TestRunner_BusyWhileRunning(t *testing.T) {
	r, store, _ := newTestRunner(t, `sleep 5`)
	m, _ := store.Create()
	if err := r.Send(m.ID, SendRequest{Text: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Send(m.ID, SendRequest{Text: "y"}); err != ErrBusy {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	r.Cancel(m.ID)
	r.Wait()
}

func TestRunner_Cancel(t *testing.T) {
	r, store, _ := newTestRunner(t, `head -n 3 "$FIXTURE"; sleep 30`)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "x"})

	// Cancel once the partial text has streamed.
	timeout := time.After(5 * time.Second)
	for streamed := false; !streamed; {
		select {
		case e := <-ch:
			streamed = e.Type == EventTextDelta
		case <-timeout:
			t.Fatal("no text streamed")
		}
	}
	if !r.Running(m.ID) {
		t.Fatal("Running = false during the turn")
	}
	if !r.Cancel(m.ID) {
		t.Fatal("Cancel returned false for a running turn")
	}
	done := make(chan struct{})
	go func() { r.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not stop after Cancel")
	}

	meta, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventTurnStopped {
		t.Fatalf("last event = %+v", last)
	}
	if events[len(events)-2].Type != EventAssistantText || events[len(events)-2].Text != "Checking " {
		t.Errorf("partial text not kept: %+v", events)
	}
	if meta.State != StateIdle || meta.SessionID != "sess-42" {
		t.Errorf("meta = %+v", meta)
	}
}

func TestRunner_ErrorResult(t *testing.T) {
	r, store, _ := newTestRunner(t, `echo '{"type":"result","is_error":true,"result":"Invalid API key"}'`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	_, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventError || last.Message != "Invalid API key" {
		t.Fatalf("last event = %+v", last)
	}
}

func TestRunner_ProcessFailsWithoutResult(t *testing.T) {
	r, store, _ := newTestRunner(t, `echo "boom: not logged in" >&2; exit 1`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	meta, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventError || !strings.Contains(last.Message, "not logged in") {
		t.Fatalf("last event = %+v", last)
	}
	if meta.State != StateIdle {
		t.Errorf("state = %s", meta.State)
	}
}

func TestRunner_DefaultModelFromConfig(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	r.cfg.DefaultModel = "sonnet"
	r.cfg.DefaultProvider = "claude"
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), "--model\nsonnet\n") {
		t.Fatalf("args = %s", args)
	}
}

func TestRunner_UnknownChat(t *testing.T) {
	r, _, _ := newTestRunner(t, `true`)
	if err := r.Send("missing", SendRequest{Text: "x"}); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestRunner_CancelAndWait(t *testing.T) {
	r, store, _ := newTestRunner(t, `sleep 30`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})

	done := make(chan struct{})
	go func() { r.CancelAndWait(m.ID); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("CancelAndWait did not return")
	}
	if r.Running(m.ID) {
		t.Fatal("turn still running after CancelAndWait")
	}
	r.CancelAndWait(m.ID) // no turn: returns at once
}

// toolResultsFixture has two parallel Read calls whose results arrive in
// reverse order, the second one denied, then a Bash call that never gets a
// result before the turn ends.
const toolResultsFixture = `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t-a","name":"Read"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"file_path\":\"a.go\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"t-b","name":"Read"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"file_path\":\"/etc/hosts\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":1}}
{"type":"system","subtype":"permission_denied","tool_use_id":"t-b"}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"not granted","is_error":true,"tool_use_id":"t-b"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"package a","tool_use_id":"t-a"}]}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t-c","name":"Bash"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"result","subtype":"success","result":"ok","session_id":"s","duration_ms":1,"usage":{}}
`

func TestRunner_ToolCallsSettleWithResults(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$(dirname "$0")/tools.jsonl"`)
	os.WriteFile(filepath.Join(dir, "tools.jsonl"), []byte(toolResultsFixture), 0o644)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()

	_, events, _ := store.Get(m.ID)
	var calls []Event
	for _, e := range events {
		if e.Type == EventToolCall {
			calls = append(calls, e)
		}
	}
	want := []struct{ id, arg, status, msg string }{
		{"t-b", "/etc/hosts", "denied", "not granted"},
		{"t-a", "a.go", "done", ""},
		{"t-c", "", "done", ""}, // no result before the turn ended
	}
	if len(calls) != len(want) {
		t.Fatalf("tool calls = %+v", calls)
	}
	for i, w := range want {
		c := calls[i]
		if c.ToolID != w.id || c.Arg != w.arg || c.Status != w.status || c.Message != w.msg {
			t.Errorf("call %d = %+v, want %+v", i, c, w)
		}
	}

	for {
		select {
		case e := <-ch:
			if e.Type == EventToolStart && e.ToolID == "" {
				t.Errorf("tool_start without tool_id: %+v", e)
			}
			continue
		default:
		}
		break
	}
}

func TestRunner_DefaultModelOnlyForDefaultProvider(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	r.cfg.DefaultModel = "gpt-x"
	r.cfg.DefaultProvider = "codex"
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x", Provider: "claude"})
	r.Wait()
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if strings.Contains(string(args), "--model") {
		t.Fatalf("codex's chat.model passed to claude: %s", args)
	}
}

func TestRunner_ProviderDefaultsToConfigured(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	if m.Agent != "" {
		t.Fatalf("new chat agent = %q, want it unset until the first message", m.Agent)
	}
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	meta, _, _ := store.Get(m.ID)
	if meta.Agent != "claude" {
		t.Fatalf("agent = %q", meta.Agent)
	}
}

func TestRunner_ProviderLockedAfterSession(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	r.cfg.Drivers["codex"] = r.cfg.Drivers["claude"]
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one", Provider: "claude"})
	r.Wait()
	if err := r.Send(m.ID, SendRequest{Text: "two", Provider: "codex"}); err != ErrProviderLocked {
		t.Fatalf("err = %v, want ErrProviderLocked", err)
	}
	if r.Running(m.ID) {
		t.Fatal("a turn started for a locked chat")
	}
}

func TestRunner_ProviderSwitchBeforeSession(t *testing.T) {
	r, store, _ := newTestRunner(t, `echo "boom" >&2; exit 1`)
	r.cfg.Drivers["codex"] = r.cfg.Drivers["claude"]
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one", Provider: "claude"})
	r.Wait()
	if err := r.Send(m.ID, SendRequest{Text: "two", Provider: "codex"}); err != nil {
		t.Fatalf("err = %v; a chat without a session can change provider", err)
	}
	r.Wait()
	meta, _, _ := store.Get(m.ID)
	if meta.Agent != "codex" {
		t.Fatalf("agent = %q", meta.Agent)
	}
}

func TestRunner_UnknownProvider(t *testing.T) {
	r, store, _ := newTestRunner(t, `true`)
	m, _ := store.Create()
	if err := r.Send(m.ID, SendRequest{Text: "x", Provider: "gemini"}); err == nil || !strings.Contains(err.Error(), "gemini") {
		t.Fatalf("err = %v", err)
	}
}
