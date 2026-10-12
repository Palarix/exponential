package chat

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

const turnFixture = `{"type":"system","subtype":"init","session_id":"sess-42"}
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
{"type":"result","subtype":"success","result":"You have 1 open issue.","session_id":"sess-42","duration_ms":800,"total_cost_usd":0.01,"usage":{"input_tokens":3,"output_tokens":7}}
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
		Store:  store,
		Binary: fakeClaude(t, dir, body),
		Dir:    dir,
		MCP:    MCPTarget{Executable: "/bin/xpo"},
	})
	t.Cleanup(r.Shutdown)
	return r, store, dir
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

	if meta.ClaudeSessionID != "sess-42" || meta.State != StateIdle || meta.Title != "how many?" {
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
	if meta.State != StateIdle || meta.ClaudeSessionID != "sess-42" {
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
