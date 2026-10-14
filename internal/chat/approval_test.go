package chat

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// A can_use_tool request as Claude Code 2.1.296 prints it, for a Bash call.
const claudeBashRequest = `{"type":"control_request","request_id":"req-1","request":{"subtype":"can_use_tool","tool_name":"Bash","display_name":"Bash","input":{"command":"echo hi > out.txt","description":"Write hi into out.txt"},"description":"Write hi into out.txt","permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Bash","ruleContent":"echo hi *"}],"behavior":"allow","destination":"localSettings"},{"type":"addDirectories","directories":["/repo/w"],"destination":"session"}],"blocked_path":"/repo/w/out.txt","tool_use_id":"toolu_1"}}`

// MCP tools come without suggestions.
const claudeMergeRequest = `{"type":"control_request","request_id":"req-2","request":{"subtype":"can_use_tool","tool_name":"mcp__xpo__merge","input":{"id":"xpo-1"},"tool_use_id":"toolu_2"}}`

const claudeResultLine = `{"type":"result","subtype":"success","result":"done","session_id":"sess-42","duration_ms":10,"usage":{"input_tokens":1,"output_tokens":1}}`

func parseControlRequest(t *testing.T, line string) claudeControlRequest {
	t.Helper()
	var cr claudeControlRequest
	if err := json.Unmarshal([]byte(line), &cr); err != nil {
		t.Fatal(err)
	}
	return cr
}

func TestClaudeApprovalRequest(t *testing.T) {
	req := claudeApproval(parseControlRequest(t, claudeBashRequest))
	if req.Tool != "Bash" || req.Detail != "Write hi into out.txt" || req.ToolID != "toolu_1" || req.Rule != "Bash(echo hi *)" {
		t.Errorf("bash request = %+v", req)
	}
	if !strings.Contains(req.Input, `"command":"echo hi > out.txt"`) {
		t.Errorf("input = %s", req.Input)
	}

	req = claudeApproval(parseControlRequest(t, claudeMergeRequest))
	if req.Tool != "mcp__xpo__merge" || req.Detail != "xpo-1" || req.Rule != "mcp__xpo__merge" {
		t.Errorf("merge request = %+v, want the tool name as the rule", req)
	}
}

func decodeResponse(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var msg struct {
		Type     string `json:"type"`
		Response struct {
			Subtype   string         `json:"subtype"`
			RequestID string         `json:"request_id"`
			Response  map[string]any `json:"response"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("response %s: %v", data, err)
	}
	if msg.Type != "control_response" || msg.Response.Subtype != "success" {
		t.Fatalf("response envelope = %s", data)
	}
	msg.Response.Response["request_id"] = msg.Response.RequestID
	return msg.Response.Response
}

func TestClaudePermissionResponse(t *testing.T) {
	cr := parseControlRequest(t, claudeBashRequest)

	allow := decodeResponse(t, claudePermissionResponse(cr, DecisionAllow))
	if allow["behavior"] != "allow" || allow["request_id"] != "req-1" || allow["updatedPermissions"] != nil {
		t.Errorf("allow = %v", allow)
	}
	if in, _ := allow["updatedInput"].(map[string]any); in["command"] != "echo hi > out.txt" {
		t.Errorf("allow must echo the input unchanged: %v", allow["updatedInput"])
	}

	session := decodeResponse(t, claudePermissionResponse(cr, DecisionAllowSession))
	perms, _ := session["updatedPermissions"].([]any)
	if session["behavior"] != "allow" || len(perms) != 2 {
		t.Fatalf("allow_session = %v", session)
	}
	for _, p := range perms {
		if d := p.(map[string]any)["destination"]; d != "session" {
			t.Errorf("permission update %v has destination %v; a chat grant must never write settings files", p, d)
		}
	}

	for _, d := range []string{DecisionDeny, DecisionCancelled} {
		deny := decodeResponse(t, claudePermissionResponse(cr, d))
		if deny["behavior"] != "deny" || deny["message"] == "" || deny["updatedInput"] != nil {
			t.Errorf("%s = %v", d, deny)
		}
	}
}

func TestClaudePermissionResponse_SessionWithoutSuggestions(t *testing.T) {
	cr := parseControlRequest(t, claudeMergeRequest)
	resp := decodeResponse(t, claudePermissionResponse(cr, DecisionAllowSession))
	perms, _ := resp["updatedPermissions"].([]any)
	if len(perms) != 1 {
		t.Fatalf("updatedPermissions = %v, want one rule for the tool", resp["updatedPermissions"])
	}
	p := perms[0].(map[string]any)
	rules, _ := p["rules"].([]any)
	if p["type"] != "addRules" || p["behavior"] != "allow" || p["destination"] != "session" || len(rules) != 1 || rules[0].(map[string]any)["toolName"] != "mcp__xpo__merge" {
		t.Errorf("permission update = %v", p)
	}
}

// approvalBody is a fake claude turn that asks once, records the answer in
// responses.txt, then finishes.
func approvalBody(dir, request string) string {
	return `printf '%s\n' '` + request + `'` + "\n" +
		`IFS= read -r resp; printf '%s\n' "$resp" >> "` + dir + `/responses.txt"` + "\n" +
		`printf '%s\n' '` + claudeResultLine + `'`
}

// newApprovalRunner returns a runner whose fake claude asks with request.
func newApprovalRunner(t *testing.T, request string) (*Runner, *Store, string) {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{
		Store:   store,
		Drivers: map[string]Driver{"claude": &ClaudeDriver{Binary: fakeClaude(t, dir, approvalBody(dir, request))}},
		Dir:     dir,
		MCP:     MCPTarget{Executable: "/bin/xpo"},
	})
	t.Cleanup(r.Shutdown)
	return r, store, dir
}

// waitForEvent returns the first event of type typ on ch.
func waitForEvent(t *testing.T, ch <-chan Event, typ string) Event {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case e := <-ch:
			if e.Type == typ {
				return e
			}
		case <-timeout:
			t.Fatalf("no %s event", typ)
		}
	}
}

func waitDone(t *testing.T, r *Runner) {
	t.Helper()
	done := make(chan struct{})
	go func() { r.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not finish")
	}
}

func recordedResponses(t *testing.T, dir string) []map[string]any {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join(dir, "responses.txt"))
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			out = append(out, decodeResponse(t, []byte(line)))
		}
	}
	return out
}

func eventsOfType(events []Event, typ string) []Event {
	var out []Event
	for _, e := range events {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

func TestRunner_ApprovalAllowOnce(t *testing.T) {
	r, store, dir := newApprovalRunner(t, claudeMergeRequest)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "merge it"})

	req := waitForEvent(t, ch, EventApprovalRequested)
	if req.ApprovalID == "" || req.Name != "mcp__xpo__merge" || req.Arg != "xpo-1" || req.ToolID != "toolu_2" || req.Input != `{"id":"xpo-1"}` {
		t.Errorf("approval_requested = %+v", req)
	}
	if !r.Waiting(m.ID) {
		t.Error("Waiting = false while a card is open")
	}
	if err := r.Decide(m.ID, req.ApprovalID, DecisionAllow); err != nil {
		t.Fatal(err)
	}
	waitDone(t, r)

	if r.Waiting(m.ID) {
		t.Error("Waiting = true after the decision")
	}
	resps := recordedResponses(t, dir)
	if len(resps) != 1 || resps[0]["behavior"] != "allow" || resps[0]["request_id"] != "req-2" {
		t.Errorf("responses = %v", resps)
	}
	meta, events, _ := store.Get(m.ID)
	resolved := eventsOfType(events, EventApprovalResolved)
	if len(resolved) != 1 || resolved[0].ApprovalID != req.ApprovalID || resolved[0].Decision != DecisionAllow {
		t.Errorf("approval_resolved = %+v", resolved)
	}
	if len(meta.Grants) != 0 {
		t.Errorf("allow once added grants %v", meta.Grants)
	}
	if err := r.Decide(m.ID, req.ApprovalID, DecisionAllow); !errors.Is(err, ErrApprovalNotFound) {
		t.Errorf("second Decide err = %v, want ErrApprovalNotFound", err)
	}
}

func TestRunner_ApprovalDeny(t *testing.T) {
	r, store, dir := newApprovalRunner(t, claudeMergeRequest)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "merge it"})

	req := waitForEvent(t, ch, EventApprovalRequested)
	if err := r.Decide(m.ID, req.ApprovalID, DecisionDeny); err != nil {
		t.Fatal(err)
	}
	waitDone(t, r)

	resps := recordedResponses(t, dir)
	if len(resps) != 1 || resps[0]["behavior"] != "deny" {
		t.Errorf("responses = %v", resps)
	}
	_, events, _ := store.Get(m.ID)
	if resolved := eventsOfType(events, EventApprovalResolved); len(resolved) != 1 || resolved[0].Decision != DecisionDeny {
		t.Errorf("approval_resolved = %+v", resolved)
	}
	if last := events[len(events)-1]; last.Type != EventTurnDone {
		t.Errorf("a denial must not end the turn; last event = %+v", last)
	}
}

func TestRunner_ApprovalAllowSessionGrantsLaterTurns(t *testing.T) {
	r, store, dir := newApprovalRunner(t, claudeBashRequest)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "write a file"})

	req := waitForEvent(t, ch, EventApprovalRequested)
	if err := r.Decide(m.ID, req.ApprovalID, DecisionAllowSession); err != nil {
		t.Fatal(err)
	}
	waitDone(t, r)

	meta, _, _ := store.Get(m.ID)
	if !slices.Equal(meta.Grants, []string{"Bash(echo hi *)"}) {
		t.Fatalf("grants = %v", meta.Grants)
	}
	if resps := recordedResponses(t, dir); len(resps) != 1 || resps[0]["updatedPermissions"] == nil {
		t.Errorf("responses = %v, want session permissions for this turn", resps)
	}

	// On the live agent, a request that matches the grant is allowed
	// without a card.
	r.Send(m.ID, SendRequest{Text: "again"})
	waitDone(t, r)
	_, events, _ := store.Get(m.ID)
	if n := len(eventsOfType(events, EventApprovalRequested)); n != 1 {
		t.Errorf("approval_requested events = %d, want 1 (the second request matches the grant)", n)
	}
	if resps := recordedResponses(t, dir); len(resps) != 2 || resps[1]["behavior"] != "allow" {
		t.Errorf("responses = %v", resps)
	}

	// A new agent process gets the grant on its command line.
	r.StopBackground(m.ID)
	r.Send(m.ID, SendRequest{Text: "once more"})
	waitDone(t, r)
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.Contains(string(args), "\nBash(echo hi *)\n") {
		t.Errorf("the new agent's args do not carry the grant:\n%s", args)
	}
}

func TestRunner_ApprovalCancelResolvesOpenRequests(t *testing.T) {
	r, store, _ := newApprovalRunner(t, claudeMergeRequest)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "merge it"})

	req := waitForEvent(t, ch, EventApprovalRequested)
	r.Cancel(m.ID)
	waitDone(t, r)

	_, events, _ := store.Get(m.ID)
	resolved := eventsOfType(events, EventApprovalResolved)
	if len(resolved) != 1 || resolved[0].ApprovalID != req.ApprovalID || resolved[0].Decision != DecisionCancelled {
		t.Errorf("approval_resolved = %+v", resolved)
	}
	if last := events[len(events)-1]; last.Type != EventTurnStopped {
		t.Errorf("last event = %+v", last)
	}
	if r.Waiting(m.ID) {
		t.Error("Waiting = true after cancel")
	}
}

func TestRunner_DecideErrors(t *testing.T) {
	r, store, _ := newApprovalRunner(t, claudeMergeRequest)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "merge it"})
	req := waitForEvent(t, ch, EventApprovalRequested)

	if err := r.Decide(m.ID, req.ApprovalID, "maybe"); !errors.Is(err, ErrBadDecision) {
		t.Errorf("bad decision err = %v", err)
	}
	if err := r.Decide(m.ID, "nope", DecisionAllow); !errors.Is(err, ErrApprovalNotFound) {
		t.Errorf("unknown ID err = %v", err)
	}
	if err := r.Decide("other-chat", req.ApprovalID, DecisionAllow); !errors.Is(err, ErrApprovalNotFound) {
		t.Errorf("other chat err = %v", err)
	}
	r.Decide(m.ID, req.ApprovalID, DecisionAllow)
	waitDone(t, r)
}

func TestRunner_UnknownControlRequestGetsError(t *testing.T) {
	r, store, dir := newApprovalRunner(t, `{"type":"control_request","request_id":"req-9","request":{"subtype":"elicitation","message":"?"}}`)
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "x"})
	waitDone(t, r)

	data, _ := os.ReadFile(filepath.Join(dir, "responses.txt"))
	var msg struct {
		Type     string `json:"type"`
		Response struct {
			Subtype   string `json:"subtype"`
			RequestID string `json:"request_id"`
			Error     string `json:"error"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &msg); err != nil || msg.Type != "control_response" || msg.Response.Subtype != "error" || msg.Response.RequestID != "req-9" || msg.Response.Error == "" {
		t.Errorf("response = %s (%v)", data, err)
	}
	_, events, _ := store.Get(m.ID)
	if n := len(eventsOfType(events, EventApprovalRequested)); n != 0 {
		t.Errorf("an unknown request showed %d cards", n)
	}
}

// Codex approval requests as codex-cli 0.162.1 prints them.
const (
	codexCommandApproval = `{"method":"item/commandExecution/requestApproval","id":7,"params":{"kind":"command","threadId":"thr-1","turnId":"turn-1","itemId":"exec-9","command":"/bin/zsh -lc \"rm -rf build\"","cwd":"/repo","commandActions":[{"type":"unknown","command":"rm -rf build"}],"availableDecisions":["accept","acceptForSession","decline","cancel"]}}`
	codexMCPApproval     = `{"method":"mcpServer/elicitation/request","id":8,"params":{"threadId":"thr-1","turnId":"turn-1","serverName":"xpo","mode":"form","_meta":{"codex_approval_kind":"mcp_tool_call","tool_description":"Merge","tool_params":{"id":"xpo-1"}},"message":"Allow the xpo MCP server to run tool \"merge\"?","requestedSchema":{"type":"object","properties":{}}}}`
	codexTurnStartOnly   = `{"id":3,"result":{"turn":{"id":"turn-1","status":"inProgress","items":[]}}}
{"method":"turn/started","params":{"threadId":"thr-1","turn":{"id":"turn-1","status":"inProgress"}}}
`
	codexTurnCompleted = `{"method":"turn/completed","params":{"threadId":"thr-1","turn":{"id":"turn-1","status":"completed","error":null,"durationMs":5}}}`
)

// codexAsk prints each request, then records the answer to it.
func codexAsk(dir string, requests ...string) string {
	var b strings.Builder
	for _, req := range requests {
		b.WriteString(`printf '%s\n' '` + req + `'` + "\n")
		b.WriteString(`IFS= read -r resp; printf '%s\n' "$resp" >> "` + dir + `/responses.txt"` + "\n")
	}
	b.WriteString(`printf '%s\n' '` + codexTurnCompleted + `'`)
	return b.String()
}

func TestCodexDriver_ForwardsApprovals(t *testing.T) {
	dir := t.TempDir()
	writeCodexFixtures(t, dir, codexInitFixture, codexThreadFixture, codexTurnStartOnly)
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{
		Store:   store,
		Drivers: map[string]Driver{"codex": &CodexDriver{Binary: fakeCodex(t, dir, codexAsk(dir, codexCommandApproval, codexMCPApproval, codexCommandApproval))}},
		Dir:     dir,
		MCP:     MCPTarget{Executable: "/bin/xpo"},
	})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "clean up", Provider: "codex"})

	cmd := waitForEvent(t, ch, EventApprovalRequested)
	if cmd.Name != "Bash" || cmd.Arg != "rm -rf build" || cmd.ToolID != "exec-9" {
		t.Errorf("command approval = %+v", cmd)
	}
	r.Decide(m.ID, cmd.ApprovalID, DecisionAllowSession)

	mcp := waitForEvent(t, ch, EventApprovalRequested)
	if mcp.Name != "mcp__xpo__merge" || mcp.Arg != "xpo-1" || mcp.Input != `{"id":"xpo-1"}` {
		t.Errorf("MCP approval = %+v", mcp)
	}
	r.Decide(m.ID, mcp.ApprovalID, DecisionDeny)

	// The same command again matches the grant: no card.
	waitDone(t, r)

	var resps []map[string]any
	data, _ := os.ReadFile(filepath.Join(dir, "responses.txt"))
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var m map[string]any
		json.Unmarshal([]byte(line), &m)
		resps = append(resps, m)
	}
	if len(resps) != 3 {
		t.Fatalf("responses = %v", resps)
	}
	result := func(i int) map[string]any { r, _ := resps[i]["result"].(map[string]any); return r }
	if resps[0]["id"] != float64(7) || result(0)["decision"] != "acceptForSession" {
		t.Errorf("command response = %v", resps[0])
	}
	if resps[1]["id"] != float64(8) || result(1)["action"] != "decline" {
		t.Errorf("MCP response = %v", resps[1])
	}
	if result(2)["decision"] != "accept" {
		t.Errorf("granted command response = %v", resps[2])
	}
	_, events, _ := store.Get(m.ID)
	if n := len(eventsOfType(events, EventApprovalRequested)); n != 2 {
		t.Errorf("approval_requested = %d, want 2", n)
	}
}

func TestCodexApprovalDecisions(t *testing.T) {
	offered := []any{"accept", "acceptForSession", "cancel"}
	for _, c := range []struct {
		decision string
		offered  []any
		want     string
	}{
		{DecisionAllow, offered, "accept"},
		{DecisionAllowSession, offered, "acceptForSession"},
		{DecisionAllowSession, []any{"accept", "cancel"}, "accept"},
		{DecisionCancelled, []any{"accept", "decline"}, "decline"},
		{DecisionDeny, nil, "decline"},
		// codex-cli 0.162.1 offers no "decline" for some commands.
		{DecisionDeny, offered, "cancel"},
	} {
		if got := codexDecision(c.decision, c.offered); got != c.want {
			t.Errorf("codexDecision(%s, %v) = %q, want %q", c.decision, c.offered, got, c.want)
		}
	}
}

func TestRunner_DeniedApprovalMarksToolCallDenied(t *testing.T) {
	dir := t.TempDir()
	// Claude Code reports a tool call the user denied as an error result.
	body := `printf '%s\n' '{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_2","name":"mcp__xpo__merge"}}}'` + "\n" +
		`printf '%s\n' '{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"id\":\"xpo-1\"}"}}}'` + "\n" +
		`printf '%s\n' '{"type":"stream_event","event":{"type":"content_block_stop","index":0}}'` + "\n" +
		`printf '%s\n' '` + claudeMergeRequest + `'` + "\n" +
		`IFS= read -r resp` + "\n" +
		`printf '%s\n' '{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_2","is_error":true,"content":"The user declined this tool call."}]}}'` + "\n" +
		`printf '%s\n' '` + claudeResultLine + `'`
	store := NewStore(filepath.Join(dir, "chats"))
	r := NewRunner(RunnerConfig{
		Store:   store,
		Drivers: map[string]Driver{"claude": &ClaudeDriver{Binary: fakeClaude(t, dir, body)}},
		Dir:     dir,
		MCP:     MCPTarget{Executable: "/bin/xpo"},
	})
	t.Cleanup(r.Shutdown)
	m, _ := store.Create()
	ch, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()
	r.Send(m.ID, SendRequest{Text: "merge it"})

	req := waitForEvent(t, ch, EventApprovalRequested)
	r.Decide(m.ID, req.ApprovalID, DecisionDeny)
	waitDone(t, r)

	_, events, _ := store.Get(m.ID)
	calls := eventsOfType(events, EventToolCall)
	if len(calls) != 1 || calls[0].ToolID != "toolu_2" || calls[0].Status != "denied" {
		t.Errorf("tool calls = %+v, want toolu_2 denied", calls)
	}
}
