package chat

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// Codex app-server output, trimmed from a recorded session (codex-cli
// 0.162.1). Each phase is what the server prints after reading one more
// request: initialize, then thread/start or thread/resume, then turn/start.
const (
	codexInitFixture = `{"id":1,"result":{"userAgent":"xpo/0.162.1","codexHome":"/home/u/.codex","platformFamily":"unix","platformOs":"macos"}}
{"method":"account/updated","params":{"authMode":"chatgpt","planType":"plus"}}
`
	codexThreadFixture = `{"method":"remoteControl/status/changed","params":{"status":"disabled"}}
{"id":2,"result":{"thread":{"id":"thr-1","sessionId":"thr-1"},"model":"gpt-6.1-sol","reasoningEffort":"low"}}
{"method":"thread/started","params":{"thread":{"id":"thr-1"}}}
`
	// The turn includes a stale usage update for the previous turn, which
	// thread/resume replays, and a server request the driver does not forward
	// and must refuse.
	codexTurnFixture = `{"method":"thread/tokenUsage/updated","params":{"threadId":"thr-1","turnId":"turn-0","tokenUsage":{"total":{"inputTokens":900,"outputTokens":90},"last":{"inputTokens":900,"outputTokens":90}}}}
{"id":3,"result":{"turn":{"id":"turn-1","status":"inProgress","items":[]}}}
{"method":"turn/started","params":{"threadId":"thr-1","turn":{"id":"turn-1","status":"inProgress"}}}
{"method":"item/started","params":{"item":{"type":"userMessage","id":"u-1","content":[{"type":"text","text":"how many?"}]},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"userMessage","id":"u-1","content":[{"type":"text","text":"how many?"}]},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/started","params":{"item":{"type":"agentMessage","id":"m-1","text":"","phase":"commentary"},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/agentMessage/delta","params":{"delta":"Checking ","itemId":"m-1","threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/agentMessage/delta","params":{"delta":"issues.","itemId":"m-1","threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"agentMessage","id":"m-1","text":"Checking issues.","phase":"commentary"},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/started","params":{"item":{"type":"commandExecution","id":"exec-1","command":"/bin/zsh -lc 'head -1 go.mod'","cwd":"/repo","status":"inProgress","commandActions":[{"type":"unknown","command":"head -1 go.mod"}],"aggregatedOutput":null,"exitCode":null},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"commandExecution","id":"exec-1","command":"/bin/zsh -lc 'head -1 go.mod'","cwd":"/repo","status":"completed","commandActions":[{"type":"unknown","command":"head -1 go.mod"}],"aggregatedOutput":"module x\n","exitCode":0},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"thread/tokenUsage/updated","params":{"threadId":"thr-1","turnId":"turn-1","tokenUsage":{"total":{"inputTokens":1000,"outputTokens":100},"last":{"inputTokens":100,"outputTokens":10}}}}
{"id":"srv-1","method":"item/permissions/requestApproval","params":{"threadId":"thr-1","turnId":"turn-1","itemId":"exec-2"}}
{"method":"item/started","params":{"item":{"type":"reasoning","id":"rs-1","summary":[],"content":[]},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"reasoning","id":"rs-1","summary":[],"content":[]},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/started","params":{"item":{"type":"commandExecution","id":"exec-2","command":"/bin/zsh -lc 'rg nothing'","status":"inProgress","commandActions":[],"aggregatedOutput":null,"exitCode":null},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"commandExecution","id":"exec-2","command":"/bin/zsh -lc 'rg nothing'","status":"failed","commandActions":[],"aggregatedOutput":"rg: nothing: No such file\n","exitCode":2},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/started","params":{"item":{"type":"mcpToolCall","id":"mcp-1","server":"xpo","tool":"list","status":"inProgress","arguments":{"match":"login","status":["DOING"]},"result":null,"error":null},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"mcpToolCall","id":"mcp-1","server":"xpo","tool":"list","status":"completed","arguments":{"match":"login","status":["DOING"]},"result":{"content":[{"type":"text","text":"1 issue(s)"}]},"error":null},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/started","params":{"item":{"type":"mcpToolCall","id":"mcp-2","server":"xpo","tool":"show","status":"inProgress","arguments":{"id":"xpo-9"},"result":null,"error":null},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"mcpToolCall","id":"mcp-2","server":"xpo","tool":"show","status":"failed","arguments":{"id":"xpo-9"},"result":null,"error":{"message":"issue xpo-9 not found"}},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/started","params":{"item":{"type":"agentMessage","id":"m-2","text":"","phase":"final_answer"},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/agentMessage/delta","params":{"delta":"You have 1 open issue.","itemId":"m-2","threadId":"thr-1","turnId":"turn-1"}}
{"method":"item/completed","params":{"item":{"type":"agentMessage","id":"m-2","text":"You have 1 open issue.","phase":"final_answer"},"threadId":"thr-1","turnId":"turn-1"}}
{"method":"thread/tokenUsage/updated","params":{"threadId":"thr-1","turnId":"turn-1","tokenUsage":{"total":{"inputTokens":1200,"outputTokens":120},"last":{"inputTokens":200,"outputTokens":20},"modelContextWindow":258400}}}
{"method":"thread/status/changed","params":{"threadId":"thr-1","status":{"type":"idle"}}}
{"method":"turn/completed","params":{"threadId":"thr-1","turn":{"id":"turn-1","status":"completed","error":null,"durationMs":7521}}}
`
)

// fakeCodex writes a fake `codex` that records its args, the token env var
// and every line it reads in dir, and answers each phase from the files
// init.jsonl, thread.jsonl and turn.jsonl. After the turn phase it runs
// after (for example a sleep), then reads stdin until it closes.
func fakeCodex(t *testing.T, dir, after string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake codex is a shell script")
	}
	script := filepath.Join(dir, "codex")
	content := "#!/bin/sh\n" +
		`for a in "$@"; do printf '%s\n' "$a"; done > "` + dir + `/args.txt"` + "\n" +
		`printf '%s' "$` + codexTokenEnv + `" > "` + dir + `/token.txt"` + "\n" +
		`for phase in init thread turn; do` + "\n" +
		`  if [ "$phase" = thread ]; then IFS= read -r line; printf '%s\n' "$line" >> "` + dir + `/stdin.txt"; fi` + "\n" +
		`  IFS= read -r line || exit 0; printf '%s\n' "$line" >> "` + dir + `/stdin.txt"` + "\n" +
		`  cat "` + dir + `/$phase.jsonl"` + "\n" +
		`done` + "\n" +
		after + "\n" +
		`cat >> "` + dir + `/stdin.txt"` + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func writeCodexFixtures(t *testing.T, dir, init, thread, turn string) {
	t.Helper()
	for name, body := range map[string]string{"init": init, "thread": thread, "turn": turn} {
		if err := os.WriteFile(filepath.Join(dir, name+".jsonl"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func newCodexRunner(t *testing.T, after string) (*Runner, *Store, string) {
	t.Helper()
	dir := t.TempDir()
	writeCodexFixtures(t, dir, codexInitFixture, codexThreadFixture, codexTurnFixture)
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{
		Store:   store,
		Drivers: map[string]Driver{"codex": &CodexDriver{Binary: fakeCodex(t, dir, after), InterruptGrace: 200 * time.Millisecond}},
		Dir:     dir,
		MCP:     MCPTarget{Executable: "/bin/xpo"},
	})
	t.Cleanup(r.Shutdown)
	return r, store, dir
}

// sentRequests decodes the JSON-RPC messages the driver wrote, by method.
// Responses to server requests are keyed "response".
func sentRequests(t *testing.T, dir string) map[string][]map[string]any {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join(dir, "stdin.txt"))
	out := map[string][]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("driver wrote non-JSON line %q", line)
		}
		key, _ := m["method"].(string)
		if key == "" {
			key = "response"
		}
		out[key] = append(out[key], m)
	}
	return out
}

func params(m map[string]any) map[string]any {
	p, _ := m["params"].(map[string]any)
	return p
}

func TestCodexDriver_Turn(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	m, _ := store.Create()

	if err := r.Send(m.ID, SendRequest{Text: "how many?", Provider: "codex", Model: "gpt-6.1-sol", Effort: "low"}); err != nil {
		t.Fatal(err)
	}
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	var got []string
	for _, e := range events {
		switch e.Type {
		case EventAssistantText:
			got = append(got, "text:"+e.Text)
		case EventToolCall:
			got = append(got, "tool:"+e.Name+"|"+e.Arg+"|"+e.Status+"|"+e.Message)
		default:
			got = append(got, e.Type)
		}
	}
	want := []string{
		EventUserMessage,
		"text:Checking issues.",
		"tool:Bash|head -1 go.mod|done|",
		"tool:Bash|rg nothing|error|rg: nothing: No such file",
		"tool:mcp__xpo__list|login|done|",
		"tool:mcp__xpo__show|xpo-9|error|issue xpo-9 not found",
		"text:You have 1 open issue.",
		EventTurnDone,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("events =\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	if meta.ResolvedModel != "gpt-6.1-sol" || meta.ContextWindow != 258400 {
		t.Errorf("resolved = %q, %d; want the thread's model and its context window", meta.ResolvedModel, meta.ContextWindow)
	}
	done := events[len(events)-1]
	if done.TokensIn != 300 || done.TokensOut != 30 || done.DurationMS != 7521 || done.CostUSD != 0 {
		t.Errorf("turn done = %+v, want this turn's tokens only (300/30) and 7521ms", done)
	}
	// Codex reports no cache TTL; xpo assumes 5 minutes.
	if done.CacheTTL != 300 || meta.CacheExpiresAt == nil {
		t.Errorf("cache ttl %d, expires %v; want 300s", done.CacheTTL, meta.CacheExpiresAt)
	}
	if meta.Agent != "codex" || meta.SessionID != "thr-1" || meta.State != StateIdle {
		t.Errorf("meta = %+v", meta)
	}

	sent := sentRequests(t, dir)
	start := params(sent["thread/start"][0])
	if start["sandbox"] != "read-only" || start["approvalPolicy"] != "on-request" || start["cwd"] != dir {
		t.Errorf("thread/start params = %v", start)
	}
	if instr := start["developerInstructions"].(string); !strings.Contains(instr, "xpo") || !strings.Contains(instr, "shell commands") || strings.Contains(instr, "Glob") {
		t.Errorf("developerInstructions is not the Codex system prompt: %v", instr)
	}
	turn := params(sent["turn/start"][0])
	if turn["threadId"] != "thr-1" || turn["model"] != "gpt-6.1-sol" || turn["effort"] != "low" {
		t.Errorf("turn/start params = %v", turn)
	}
	input, _ := json.Marshal(turn["input"])
	if !strings.Contains(string(input), "how many?") {
		t.Errorf("turn/start input = %s", input)
	}
	if len(sent["response"]) != 1 || sent["response"][0]["id"] != "srv-1" || sent["response"][0]["error"] == nil {
		t.Errorf("permissions request answered with %v, want an error response to srv-1", sent["response"])
	}

	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	for _, a := range []string{"app-server\n", `mcp_servers.xpo.command="/bin/xpo"`, `mcp_servers.xpo.args=["mcp"]`, `mcp_servers.xpo.default_tools_approval_mode="approve"`} {
		if !strings.Contains(string(args), a) {
			t.Errorf("args missing %q:\n%s", a, args)
		}
	}
}

// In Work Mode the thread runs in the worktree, may write there, and merge
// asks before it runs.
func TestCodexDriver_WorkProfile(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	r.cfg.CanWork = true
	wt := filepath.Join(dir, "wt")
	os.Mkdir(wt, 0o755)
	m, _ := store.Create()
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-1", Worktree: wt}); err != nil {
		t.Fatal(err)
	}
	r.Send(m.ID, SendRequest{Text: "go", Provider: "codex"})
	r.Wait()

	start := params(sentRequests(t, dir)["thread/start"][0])
	if start["cwd"] != wt || start["sandbox"] != "workspace-write" || start["approvalPolicy"] != "on-request" {
		t.Errorf("thread/start params = %v", start)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), `mcp_servers.xpo.tools.merge.approval_mode="prompt"`) || strings.Contains(string(args), "disabled_tools") {
		t.Errorf("args = %s, want merge to ask and nothing disabled", args)
	}
}

func TestCodexDriver_LiveEvents(t *testing.T) {
	r, store, _ := newCodexRunner(t, "")
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

	r.Send(m.ID, SendRequest{Text: "x", Provider: "codex"})
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
	for _, want := range []string{EventTextDelta, EventThinking, EventToolStart} {
		if !slices.Contains(types, want) {
			t.Errorf("published events %v missing %s", types, want)
		}
	}
}

func TestCodexDriver_SecondTurnResumesThread(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one", Provider: "codex"})
	r.Wait()
	os.Remove(filepath.Join(dir, "stdin.txt"))
	r.Send(m.ID, SendRequest{Text: "two", Provider: "codex"})
	r.Wait()

	sent := sentRequests(t, dir)
	if len(sent["thread/start"]) != 0 || len(sent["thread/resume"]) != 1 {
		t.Fatalf("second turn sent %v", sent)
	}
	resume := params(sent["thread/resume"][0])
	if resume["threadId"] != "thr-1" || resume["sandbox"] != "read-only" || resume["approvalPolicy"] != "on-request" {
		t.Errorf("thread/resume params = %v", resume)
	}
	turn := params(sent["turn/start"][0])
	if _, ok := turn["model"]; ok {
		t.Errorf("turn/start sent a model although none was picked: %v", turn)
	}
}

func TestCodexDriver_RemoteTokenOnlyInEnv(t *testing.T) {
	dir := t.TempDir()
	writeCodexFixtures(t, dir, codexInitFixture, codexThreadFixture, codexTurnFixture)
	d := &CodexDriver{Binary: fakeCodex(t, dir, "")}
	_, err := d.Turn(context.Background(), TurnInput{
		Prompt: "x",
		Dir:    dir,
		MCP:    MCPTarget{RemoteURL: "https://xpo.example.com/", RemoteToken: "s3cret"},
	}, func(ev exponential.StreamEvent) {})
	if err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if strings.Contains(string(args), "s3cret") {
		t.Fatalf("token on the command line:\n%s", args)
	}
	for _, a := range []string{`mcp_servers.xpo.url="https://xpo.example.com/mcp"`, `mcp_servers.xpo.bearer_token_env_var="` + codexTokenEnv + `"`} {
		if !strings.Contains(string(args), a) {
			t.Errorf("args missing %q:\n%s", a, args)
		}
	}
	if strings.Contains(string(args), "mcp_servers.xpo.command") {
		t.Errorf("remote mode still starts a local xpo mcp:\n%s", args)
	}
	token, _ := os.ReadFile(filepath.Join(dir, "token.txt"))
	if string(token) != "s3cret" {
		t.Errorf("token env = %q", token)
	}
}

func TestCodexDriver_TurnFailed(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	os.WriteFile(filepath.Join(dir, "turn.jsonl"), []byte(`{"id":3,"result":{"turn":{"id":"turn-1","status":"inProgress"}}}
{"method":"error","params":{"threadId":"thr-1","turnId":"turn-1","error":{"message":"stream disconnected"},"willRetry":true}}
{"method":"turn/completed","params":{"threadId":"thr-1","turn":{"id":"turn-1","status":"failed","error":{"message":"You've hit your usage limit."},"durationMs":10}}}
`), 0o644)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x", Provider: "codex"})
	r.Wait()

	_, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventError || last.Message != "You've hit your usage limit." {
		t.Fatalf("last event = %+v", last)
	}
}

func TestCodexDriver_ResumeFails(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "one", Provider: "codex"})
	r.Wait()

	os.WriteFile(filepath.Join(dir, "thread.jsonl"), []byte(`{"id":2,"error":{"code":-32600,"message":"no rollout found for thread id thr-1"}}
`), 0o644)
	r.Send(m.ID, SendRequest{Text: "two", Provider: "codex"})
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	last := events[len(events)-1]
	if last.Type != EventError || !strings.Contains(last.Message, "no rollout found") || !strings.Contains(last.Message, "new chat") {
		t.Fatalf("last event = %+v", last)
	}
	if meta.SessionID != "thr-1" {
		t.Errorf("session ID = %q, want the thread kept", meta.SessionID)
	}
}

func TestCodexDriver_ExitsEarly(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "codex")
	os.WriteFile(script, []byte("#!/bin/sh\necho 'error: not logged in' >&2\nexit 1\n"), 0o755)
	if runtime.GOOS == "windows" {
		t.Skip("fake codex is a shell script")
	}
	d := &CodexDriver{Binary: script}
	res, err := d.Turn(context.Background(), TurnInput{Prompt: "x", Dir: dir}, func(exponential.StreamEvent) {})
	if res != nil || err == nil || !strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("res = %+v, err = %v", res, err)
	}
}

func TestCodexDriver_Cancel(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	// Stop after the first text delta; the fake never completes the turn.
	turn := strings.SplitAfter(codexTurnFixture, "\n")
	os.WriteFile(filepath.Join(dir, "turn.jsonl"), []byte(strings.Join(turn[:7], "")), 0o644)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "x", Provider: "codex"})

	timeout := time.After(5 * time.Second)
	for streamed := false; !streamed; {
		select {
		case e := <-ch:
			streamed = e.Type == EventTextDelta
		case <-timeout:
			t.Fatal("no text streamed")
		}
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
	if last := events[len(events)-1]; last.Type != EventTurnStopped {
		t.Fatalf("last event = %+v", last)
	}
	if meta.SessionID != "thr-1" {
		t.Errorf("session ID = %q", meta.SessionID)
	}
	sent := sentRequests(t, dir)
	if len(sent["turn/interrupt"]) != 1 || params(sent["turn/interrupt"][0])["turnId"] != "turn-1" {
		t.Errorf("turn/interrupt not sent: %v", sent)
	}
}

const codexModelsFixture = `{"id":1,"result":{"userAgent":"xpo/0.162.1"}}
{"method":"account/updated","params":{"authMode":"chatgpt"}}
{"id":2,"result":{"data":[
{"id":"gpt-6.1-sol","model":"gpt-6.1-sol","displayName":"GPT-6.1-Sol","hidden":false,"isDefault":true,"defaultReasoningEffort":"low","supportedReasoningEfforts":[{"reasoningEffort":"low"},{"reasoningEffort":"medium"},{"reasoningEffort":"high"}]},
{"id":"gpt-old","model":"gpt-old","displayName":"Old","hidden":true,"isDefault":false,"supportedReasoningEfforts":[{"reasoningEffort":"low"}]},
{"id":"gpt-6-astra","model":"gpt-6-astra","displayName":"GPT-6-Astra","hidden":false,"isDefault":false,"supportedReasoningEfforts":[{"reasoningEffort":"medium"},{"reasoningEffort":"xhigh"}]}
]}}
`

func TestCodexModels(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake codex is a shell script")
	}
	dir := t.TempDir()
	// The fixture's model/list response spans several lines; join them so
	// the fake prints one JSON message per line.
	lines := strings.Split(strings.TrimSpace(codexModelsFixture), "\n")
	fixture := lines[0] + "\n" + lines[1] + "\n" + strings.Join(lines[2:], "") + "\n"
	script := filepath.Join(dir, "codex")
	os.WriteFile(filepath.Join(dir, "init.jsonl"), []byte(lines[0]+"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "models.jsonl"), []byte(strings.TrimPrefix(fixture, lines[0]+"\n")), 0o644)
	os.WriteFile(script, []byte("#!/bin/sh\nIFS= read -r a\ncat \""+dir+"/init.jsonl\"\nIFS= read -r b\nIFS= read -r c\ncat \""+dir+"/models.jsonl\"\ncat > /dev/null\n"), 0o755)

	models, err := CodexModels(context.Background(), script)
	if err != nil {
		t.Fatal(err)
	}
	want := []ModelOption{
		{ID: "", Label: "Default model", Name: "GPT-6.1-Sol", Efforts: []string{"low", "medium", "high"}},
		{ID: "gpt-6.1-sol", Label: "GPT-6.1-Sol", Name: "GPT-6.1-Sol", Efforts: []string{"low", "medium", "high"}},
		{ID: "gpt-6-astra", Label: "GPT-6-Astra", Name: "GPT-6-Astra", Efforts: []string{"medium", "xhigh"}},
	}
	got, _ := json.Marshal(models)
	exp, _ := json.Marshal(want)
	if string(got) != string(exp) {
		t.Fatalf("models =\n%s\nwant\n%s", got, exp)
	}
}

func TestCodexModels_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake codex is a shell script")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "codex")
	os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o755)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := CodexModels(ctx, script); err == nil {
		t.Fatal("want an error when codex does not answer")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("CodexModels took %v after its context expired", time.Since(start))
	}
}

func TestUnwrapShell(t *testing.T) {
	tests := map[string]string{
		`/bin/zsh -lc 'rg nothing'`:     "rg nothing",
		`/bin/bash -lc "git log -1"`:    "git log -1",
		`ls -la`:                        "ls -la",
		`/bin/zsh -lc 'echo "it'\''s"'`: `echo "it'\''s"`,
	}
	for in, want := range tests {
		if got := unwrapShell(in); got != want {
			t.Errorf("unwrapShell(%q) = %q, want %q", in, got, want)
		}
	}
}
