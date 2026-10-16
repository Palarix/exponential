package chat

import (
	"encoding/json"
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

// fakeClaude writes a fake `claude` that lives across turns, like the real
// one. It records its args, its working directory and each start in dir.
// For each user message it records the message in stdin.txt and runs body.
// It records control requests in control.txt and answers them with success,
// and other lines, such as late permission answers, in other.txt.
func fakeClaude(t *testing.T, dir, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake claude is a shell script")
	}
	script := filepath.Join(dir, "claude")
	content := "#!/bin/sh\n" +
		`for a in "$@"; do printf '%s\n' "$a"; done > "` + dir + `/args.txt"` + "\n" +
		`pwd -P > "` + dir + `/pwd.txt"` + "\n" +
		`echo start >> "` + dir + `/starts.txt"` + "\n" +
		"turn() {\n" + body + "\n:\n}\n" +
		`while IFS= read -r line; do` + "\n" +
		`  case "$line" in` + "\n" +
		`  *'"control_request"'*)` + "\n" +
		`    printf '%s\n' "$line" >> "` + dir + `/control.txt"` + "\n" +
		`    id=$(printf '%s' "$line" | sed 's/.*"request_id":"\([^"]*\)".*/\1/')` + "\n" +
		`    printf '{"type":"control_response","response":{"subtype":"success","request_id":"%s"}}\n' "$id" ;;` + "\n" +
		`  *'"type":"user"'*) printf '%s\n' "$line" > "` + dir + `/stdin.txt"; turn ;;` + "\n" +
		`  *) printf '%s\n' "$line" >> "` + dir + `/other.txt" ;;` + "\n" +
		`  esac` + "\n" +
		`done` + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// newTestRunner returns a runner whose fake claude runs body for each turn.
// opts change the config before the runner starts.
func newTestRunner(t *testing.T, body string, opts ...func(*RunnerConfig)) (*Runner, *Store, string) {
	t.Helper()
	dir := t.TempDir()
	fixture := filepath.Join(dir, "fixture.jsonl")
	os.WriteFile(fixture, []byte(turnFixture), 0o644)
	body = strings.ReplaceAll(body, "$FIXTURE", fixture)

	store := NewStore(filepath.Join(dir, "chats"))
	cfg := RunnerConfig{
		Store:          store,
		Drivers:        map[string]Driver{"claude": &ClaudeDriver{Binary: fakeClaude(t, dir, body), StopGrace: time.Second}},
		Dir:            dir,
		MCP:            MCPTarget{Executable: "/bin/xpo"},
		InterruptGrace: 300 * time.Millisecond,
	}
	for _, o := range opts {
		o(&cfg)
	}
	r := NewRunner(cfg)
	t.Cleanup(r.Shutdown)
	return r, store, dir
}

// starts counts how often the fake agent in dir started.
func starts(t *testing.T, dir string) int {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join(dir, "starts.txt"))
	return strings.Count(string(data), "start")
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
	var msg struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(stdin, &msg); err != nil || msg.Type != "user" || msg.Message.Role != "user" {
		t.Fatalf("stdin is not a stream-json user message: %q (%v)", stdin, err)
	}
	if !strings.Contains(msg.Message.Content, "open issue: xpo-1") || !strings.HasSuffix(msg.Message.Content, "how many?") {
		t.Errorf("prompt = %q", msg.Message.Content)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), "--model\nhaiku\n") || !strings.Contains(string(args), "--effort\nlow\n") {
		t.Errorf("args = %s", args)
	}
}

// The agent lives across turns: the second message goes to the same
// process.
func TestRunner_SecondTurnUsesLiveSession(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	r.Send(m.ID, SendRequest{Text: "two"})
	r.Wait()

	if n := starts(t, dir); n != 1 {
		t.Fatalf("agent started %d times, want once", n)
	}
	if stdin, _ := os.ReadFile(filepath.Join(dir, "stdin.txt")); !strings.Contains(string(stdin), "two") {
		t.Errorf("second message = %s", stdin)
	}
	meta, events, _ := store.Get(m.ID)
	if meta.Title != "one" {
		t.Errorf("title changed to %q", meta.Title)
	}
	if n := len(eventsOfType(events, EventTurnDone)); n != 2 {
		t.Errorf("turn_done events = %d, want 2", n)
	}
}

// Once the agent ended, the next message starts it again and resumes the
// session from disk.
func TestRunner_ResumesAfterAgentEnds(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	if !r.StopBackground(m.ID) {
		t.Fatal("StopBackground = false with a live agent")
	}
	r.Send(m.ID, SendRequest{Text: "two"})
	r.Wait()

	if n := starts(t, dir); n != 2 {
		t.Fatalf("agent started %d times, want twice", n)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), "--resume\nsess-42\n") {
		t.Fatalf("second start args = %s", args)
	}
}

func TestRunner_MCPConfigFileRemovedWhenAgentEnds(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()

	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	lines := strings.Split(string(args), "\n")
	path := lines[slices.Index(lines, "--mcp-config")+1]
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("mcp config %s gone while the agent runs (err=%v)", path, err)
	}
	r.StopBackground(m.ID)
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

func TestRunner_ClosesASubscriberThatFallsBehind(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	keep, unsubscribeKeep := r.Subscribe(m.ID)
	defer unsubscribeKeep()

	// Overflow ch's buffer; keep drains as it goes.
	for i := range cap(ch) + 1 {
		r.live(m.ID, Event{Type: EventTextDelta, Text: fmt.Sprint(i)})
		<-keep
	}
	n := 0
drain:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break drain
			}
			n++
		case <-time.After(time.Second):
			t.Fatalf("subscriber still open after %d events", n)
		}
	}
	if n != cap(ch) {
		t.Fatalf("got %d events before close, want %d", n, cap(ch))
	}

	// Later events skip the closed subscriber, and unsubscribing is a no-op.
	r.live(m.ID, Event{Type: EventTextDelta, Text: "after"})
	if e := <-keep; e.Text != "after" {
		t.Fatalf("kept subscriber got %q", e.Text)
	}
	unsubscribe()
	unsubscribe()
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
	if meta.State != StateIdle || meta.SessionID != "sess-42" || meta.LastError != "" {
		t.Errorf("meta = %+v", meta)
	}
}

func TestRunner_ErrorResult(t *testing.T) {
	r, store, _ := newTestRunner(t, `echo '{"type":"result","is_error":true,"result":"Invalid API key"}'`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	r.Wait()
	meta, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventError || last.Message != "Invalid API key" {
		t.Fatalf("last event = %+v", last)
	}
	if meta.LastError != "Invalid API key" {
		t.Errorf("last error = %q, want the error result", meta.LastError)
	}
}

// failOnceBody fails the first turn with an error result and runs then for
// every later turn.
func failOnceBody(then string) string {
	return `if [ -f "$FIXTURE.failed" ]; then ` + then + `; else touch "$FIXTURE.failed"; echo '{"type":"result","is_error":true,"result":"Invalid API key","session_id":"s"}'; fi`
}

func TestRunner_FinishedTurnClearsLastError(t *testing.T) {
	r, store, _ := newTestRunner(t, failOnceBody(`cat "$FIXTURE"`))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.LastError == "" {
		t.Fatal("last error empty after a failed turn")
	}
	r.Send(m.ID, SendRequest{Text: "two"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.LastError != "" {
		t.Errorf("last error = %q after a finished turn, want empty", meta.LastError)
	}
}

func TestRunner_TurnStartClearsLastError(t *testing.T) {
	r, store, _ := newTestRunner(t, failOnceBody(`sleep 30`))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one"})
	r.Wait()
	r.Send(m.ID, SendRequest{Text: "two"})
	meta, _ := store.Meta(m.ID)
	r.Cancel(m.ID)
	r.Wait()
	if meta.State != StateRunning || meta.LastError != "" {
		t.Errorf("meta while running = state %q, last error %q; want running with no error", meta.State, meta.LastError)
	}
	if meta, _ := store.Meta(m.ID); meta.LastError != "" {
		t.Errorf("last error = %q after a stopped turn, want empty", meta.LastError)
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
	if !strings.Contains(meta.LastError, "not logged in") {
		t.Errorf("last error = %q, want the exit error", meta.LastError)
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

// cacheTurns answers the first turn with a 1h cache write and later turns
// with cache reads only.
const cacheTurns = `if [ -f seen ]; then
echo '{"type":"result","subtype":"success","result":"ok","session_id":"s","usage":{"cache_read_input_tokens":900}}'
else
touch seen
echo '{"type":"result","subtype":"success","result":"ok","session_id":"s","usage":{"cache_creation":{"ephemeral_1h_input_tokens":500}}}'
fi`

func TestRunner_RecordsCacheExpiry(t *testing.T) {
	r, store, _ := newTestRunner(t, cacheTurns)
	m, _ := store.Create()

	turn := func() (Meta, Event, time.Time, time.Time) {
		t.Helper()
		before := time.Now()
		if err := r.Send(m.ID, SendRequest{Text: "hi"}); err != nil {
			t.Fatal(err)
		}
		r.Wait()
		meta, events, _ := store.Get(m.ID)
		return meta, events[len(events)-1], before, time.Now()
	}
	inRange := func(got *time.Time, from, to time.Time, ttl time.Duration) bool {
		return got != nil && !got.Before(from.Add(ttl).Add(-time.Second)) && !got.After(to.Add(ttl))
	}

	meta, done, from, to := turn()
	if done.Type != EventTurnDone || done.CacheTTL != 3600 {
		t.Fatalf("turn_done = %+v, want cache_ttl 3600", done)
	}
	if meta.CacheTTL != 3600 || !inRange(meta.CacheExpiresAt, from, to, time.Hour) {
		t.Errorf("meta = ttl %d expires %v, want 1h from the turn end", meta.CacheTTL, meta.CacheExpiresAt)
	}

	// A turn that only reads the cache refreshes the TTL the chat last wrote.
	meta, done, from, to = turn()
	if done.CacheTTL != 3600 || !inRange(meta.CacheExpiresAt, from, to, time.Hour) {
		t.Errorf("second turn: turn_done %+v, expires %v; want 1h kept", done, meta.CacheExpiresAt)
	}
}

func TestRunner_CacheExpiryDefaultsToFiveMinutes(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	before := time.Now()
	r.Send(m.ID, SendRequest{Text: "hi"})
	r.Wait()
	meta, _ := store.Meta(m.ID)
	if meta.CacheTTL != 300 || meta.CacheExpiresAt == nil || meta.CacheExpiresAt.Before(before.Add(5*time.Minute).Add(-time.Second)) {
		t.Errorf("meta = ttl %d expires %v, want 5m", meta.CacheTTL, meta.CacheExpiresAt)
	}
}

func TestRunner_FailedTurnKeepsCacheExpiry(t *testing.T) {
	r, store, _ := newTestRunner(t, `echo '{"type":"result","is_error":true,"result":"Not logged in","session_id":"s"}'`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.CacheExpiresAt != nil {
		t.Errorf("expires = %v, want none after a failed turn", meta.CacheExpiresAt)
	}
}
