package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// codexTokenEnv is the environment variable that carries the remote xpo
// token to Codex, so the token never shows up in the process list.
const codexTokenEnv = "XPO_MCP_BEARER_TOKEN"

// CodexDriver runs a turn on `codex app-server`, one process per turn,
// speaking its JSON-RPC protocol over stdio. The turn's profile picks the
// sandbox, the approval policy, and which xpo MCP tools ask or are disabled;
// the others are pre-approved. Approval requests go to the user.
type CodexDriver struct {
	Binary string // "codex" when empty
	// InterruptGrace is how long a cancelled turn may take to stop after
	// turn/interrupt before the process is killed. 3s when zero.
	InterruptGrace time.Duration
}

func (d *CodexDriver) binary() string {
	if d.Binary == "" {
		return "codex"
	}
	return d.Binary
}

// codexArgs starts app-server with the xpo MCP server: this binary's stdio
// server in local mode, or the remote server's /mcp in remote mode. The
// whole xpo table is replaced, so an xpo entry in a Codex config file
// cannot mix in. The profile disables tools and makes others ask.
func codexArgs(t MCPTarget, p Profile) []string {
	server := map[string]any{"default_tools_approval_mode": "approve"}
	if t.RemoteURL != "" {
		server["url"] = strings.TrimSuffix(t.RemoteURL, "/") + "/mcp"
		if t.RemoteToken != "" {
			server["bearer_token_env_var"] = codexTokenEnv
		}
	} else {
		server["command"] = t.Executable
		server["args"] = []string{"mcp"}
	}
	args := []string{"app-server"}
	for _, key := range []string{"command", "args", "url", "bearer_token_env_var", "default_tools_approval_mode"} {
		if v, ok := server[key]; ok {
			args = append(args, "-c", "mcp_servers.xpo."+key+"="+tomlValue(v))
		}
	}
	if denied := p.deniedTools(); len(denied) > 0 {
		args = append(args, "-c", "mcp_servers.xpo.disabled_tools="+tomlValue(denied))
	}
	for _, tool := range p.askTools() {
		args = append(args, "-c", "mcp_servers.xpo.tools."+tool+".approval_mode="+tomlValue("prompt"))
	}
	return args
}

// tomlValue encodes a string or string list as TOML. JSON strings are valid
// TOML basic strings.
func tomlValue(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// Turn implements Driver.
// codexCacheTTL is the prompt cache TTL Codex turns record. Codex does not
// report one; OpenAI keeps cached prefixes for 5 to 10 minutes of
// inactivity, so xpo takes the lower bound.
const codexCacheTTL = 5 * time.Minute

func (d *CodexDriver) Turn(ctx context.Context, in TurnInput, emit exponential.StreamCallback) (*TurnResult, error) {
	grace := d.InterruptGrace
	if grace == 0 {
		grace = 3 * time.Second
	}
	p, err := startCodex(d.binary(), codexArgs(in.MCP, in.Profile), in.Dir, in.MCP.RemoteToken)
	if err != nil {
		return nil, err
	}
	defer p.stop(grace)

	// Open approval requests end with the turn.
	approveCtx, endApprovals := context.WithCancel(ctx)
	t := &codexTurn{p: p, in: in, emit: emit, usage: map[string][2]int{}, streamed: map[string]bool{}, approveCtx: approveCtx}
	defer func() {
		endApprovals()
		t.approvals.Wait()
	}()
	t.initID = p.call("initialize", map[string]any{"clientInfo": map[string]string{"name": "xpo", "title": "xpo", "version": "1"}})

	done := ctx.Done()
	var kill <-chan time.Time
	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				return nil, p.exitError()
			}
			if res, finished := t.handle(line); finished {
				return res, nil
			}
		case <-done:
			done = nil
			if t.turnID == "" {
				return nil, ctx.Err()
			}
			p.call("turn/interrupt", map[string]string{"threadId": t.threadID, "turnId": t.turnID})
			kill = time.After(grace)
		case <-kill:
			return nil, ctx.Err()
		}
	}
}

// codexTurn is the state of one turn's JSON-RPC conversation.
type codexTurn struct {
	p    *codexProc
	in   TurnInput
	emit exponential.StreamCallback

	initID, threadReqID, turnReqID int
	threadID, turnID               string

	model    string            // from the thread response
	window   int               // the latest modelContextWindow
	usage    map[string][2]int // input and output tokens by turn ID
	streamed map[string]bool   // agent message items that sent deltas

	approveCtx context.Context // ends when the turn ends
	approvals  sync.WaitGroup  // approval requests waiting for the user
}

type rpcMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// codexItem is the part of a thread item the driver reads.
type codexItem struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Text    string `json:"text"`
	Status  string `json:"status"`
	Command string `json:"command"`
	Actions []struct {
		Command string `json:"command"`
	} `json:"commandActions"`
	Output    *string        `json:"aggregatedOutput"`
	ExitCode  *int           `json:"exitCode"`
	Server    string         `json:"server"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Result    *struct {
		IsError bool            `json:"isError"`
		Content json.RawMessage `json:"content"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Changes []struct {
		Path string `json:"path"`
	} `json:"changes"`
	Query string `json:"query"`
}

// handle processes one line from app-server. It returns the turn's result
// and true once the turn has ended.
func (t *codexTurn) handle(line []byte) (*TurnResult, bool) {
	var msg rpcMessage
	if json.Unmarshal(line, &msg) != nil {
		return nil, false
	}
	switch {
	case msg.Method != "" && len(msg.ID) > 0:
		// A server request. Approvals go to the user; anything else is
		// refused, so the turn cannot hang on it.
		if req, answer, ok := codexApproval(msg.Method, msg.Params); ok {
			t.approvals.Add(1)
			go func() {
				defer t.approvals.Done()
				result := answer(t.in.approve(t.approveCtx, req))
				t.p.send(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": result})
			}()
			return nil, false
		}
		t.p.send(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "error": map[string]any{"code": -32000, "message": "xpo chat declines this request"}})
		return nil, false
	case msg.Method != "":
		return t.notification(msg.Method, msg.Params)
	}

	var id int
	if json.Unmarshal(msg.ID, &id) != nil {
		return nil, false
	}
	var errText string
	if msg.Error != nil {
		errText = msg.Error.Message
	}
	switch id {
	case t.initID:
		if errText != "" {
			return t.failed("Codex did not start: " + errText), true
		}
		t.p.notify("initialized")
		params := map[string]any{
			"cwd":                   t.in.Dir,
			"sandbox":               t.in.Profile.codexSandbox(),
			"approvalPolicy":        t.in.Profile.codexApprovalPolicy(),
			"developerInstructions": systemPromptFor(ProviderCodex),
		}
		method := "thread/start"
		if t.in.SessionID != "" {
			method = "thread/resume"
			params["threadId"] = t.in.SessionID
			params["excludeTurns"] = true
		}
		t.threadReqID = t.p.call(method, params)
	case t.threadReqID:
		if errText != "" {
			if t.in.SessionID != "" {
				return t.failed(fmt.Sprintf("Codex could not resume this chat (%s). Start a new chat.", errText)), true
			}
			return t.failed("Codex could not start a thread: " + errText), true
		}
		var res struct {
			Thread struct {
				ID string `json:"id"`
			} `json:"thread"`
			Model string `json:"model"`
		}
		json.Unmarshal(msg.Result, &res)
		t.threadID = res.Thread.ID
		t.model = res.Model
		t.emit(exponential.StreamEvent{Kind: exponential.StreamSession, Text: t.threadID})
		params := map[string]any{
			"threadId": t.threadID,
			"input":    []map[string]any{{"type": "text", "text": t.in.Prompt}},
		}
		if t.in.Model != "" {
			params["model"] = t.in.Model
		}
		if t.in.Effort != "" {
			params["effort"] = t.in.Effort
		}
		t.turnReqID = t.p.call("turn/start", params)
	case t.turnReqID:
		if errText != "" {
			return t.failed("Codex could not start the turn: " + errText), true
		}
		var res struct {
			Turn struct {
				ID string `json:"id"`
			} `json:"turn"`
		}
		json.Unmarshal(msg.Result, &res)
		t.turnID = res.Turn.ID
	}
	return nil, false
}

func (t *codexTurn) failed(msg string) *TurnResult {
	return &TurnResult{SessionID: t.threadID, IsError: true, Output: msg}
}

func (t *codexTurn) notification(method string, raw json.RawMessage) (*TurnResult, bool) {
	var p struct {
		Item       codexItem `json:"item"`
		ItemID     string    `json:"itemId"`
		Delta      string    `json:"delta"`
		TurnID     string    `json:"turnId"`
		TokenUsage struct {
			Last struct {
				InputTokens  int `json:"inputTokens"`
				OutputTokens int `json:"outputTokens"`
			} `json:"last"`
			ModelContextWindow int `json:"modelContextWindow"`
		} `json:"tokenUsage"`
		Turn struct {
			ID         string `json:"id"`
			Status     string `json:"status"`
			DurationMS int64  `json:"durationMs"`
			Error      *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"turn"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return nil, false
	}

	switch method {
	case "item/agentMessage/delta":
		t.streamed[p.ItemID] = true
		t.emit(exponential.StreamEvent{Kind: exponential.StreamText, Text: p.Delta})
	case "item/started":
		t.itemStarted(p.Item)
	case "item/completed":
		t.itemCompleted(p.Item)
	case "thread/tokenUsage/updated":
		// thread/resume replays the last update of the previous turn, so
		// usage is kept per turn.
		u := t.usage[p.TurnID]
		t.usage[p.TurnID] = [2]int{u[0] + p.TokenUsage.Last.InputTokens, u[1] + p.TokenUsage.Last.OutputTokens}
		if p.TokenUsage.ModelContextWindow > 0 {
			t.window = p.TokenUsage.ModelContextWindow
		}
	case "turn/completed":
		if t.turnID != "" && p.Turn.ID != t.turnID {
			return nil, false
		}
		u := t.usage[p.Turn.ID]
		res := &TurnResult{
			SessionID: t.threadID,
			Duration:  time.Duration(p.Turn.DurationMS) * time.Millisecond,
			TokensIn:  u[0],
			TokensOut: u[1],

			Model:         t.model,
			ContextWindow: t.window,
			CacheTTL:      codexCacheTTL,
		}
		if p.Turn.Status == "failed" {
			res.IsError = true
			res.Output = "Codex turn failed"
			if p.Turn.Error != nil && p.Turn.Error.Message != "" {
				res.Output = p.Turn.Error.Message
			}
		}
		return res, true
	}
	return nil, false
}

// codexApproval turns an app-server approval request into an
// ApprovalRequest, and returns how to answer it for a decision. ok is false
// for requests that are not approvals xpo handles.
func codexApproval(method string, raw json.RawMessage) (req ApprovalRequest, answer func(decision string) any, ok bool) {
	var p struct {
		ItemID    string `json:"itemId"`
		Command   string `json:"command"`
		Actions   []struct {
			Command string `json:"command"`
		} `json:"commandActions"`
		Offered    []any  `json:"availableDecisions"`
		ServerName string `json:"serverName"`
		Message    string `json:"message"`
		Meta       struct {
			Kind   string         `json:"codex_approval_kind"`
			Params map[string]any `json:"tool_params"`
		} `json:"_meta"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return req, nil, false
	}
	decide := func(decision string) any { return map[string]string{"decision": codexDecision(decision, p.Offered)} }
	switch method {
	case "item/commandExecution/requestApproval":
		name, arg, _ := codexToolName(codexItem{Type: "commandExecution", Command: p.Command, Actions: p.Actions})
		input, _ := json.Marshal(map[string]string{"command": arg})
		return ApprovalRequest{Tool: name, Detail: arg, Input: string(input), ToolID: p.ItemID, Rule: name + "(" + arg + ")"}, decide, true
	case "item/fileChange/requestApproval":
		return ApprovalRequest{Tool: "Edit", ToolID: p.ItemID, Rule: "Edit"}, decide, true
	case "mcpServer/elicitation/request":
		if p.Meta.Kind != "mcp_tool_call" {
			return req, nil, false
		}
		// The tool name is only in the message text. When Codex words it
		// differently, the card names the server alone.
		name := "mcp__" + p.ServerName
		prefix := "Allow the " + p.ServerName + " MCP server to run tool \""
		if rest, found := strings.CutPrefix(p.Message, prefix); found {
			if tool, _, closed := strings.Cut(rest, "\""); closed && tool != "" {
				name += "__" + tool
			}
		}
		input, _ := json.Marshal(p.Meta.Params)
		return ApprovalRequest{Tool: name, Detail: exponential.MCPToolDetail(p.Meta.Params), Input: string(input), Rule: name}, func(decision string) any {
			if decision == DecisionAllow || decision == DecisionAllowSession {
				return map[string]any{"action": "accept", "content": map[string]any{}}
			}
			return map[string]any{"action": "decline"}
		}, true
	}
	return req, nil, false
}

// codexToolName maps a Codex tool item to the name the work list knows,
// and the argument it shows. ok is false for items that are not tool calls.
func codexToolName(it codexItem) (name, arg string, ok bool) {
	switch it.Type {
	case "commandExecution":
		arg = unwrapShell(it.Command)
		if len(it.Actions) == 1 && it.Actions[0].Command != "" {
			arg = it.Actions[0].Command
		}
		return "Bash", arg, true
	case "mcpToolCall":
		return "mcp__" + it.Server + "__" + it.Tool, exponential.MCPToolDetail(it.Arguments), true
	case "fileChange":
		if len(it.Changes) > 0 {
			arg = it.Changes[0].Path
		}
		return "Edit", arg, true
	case "webSearch":
		return "WebSearch", it.Query, true
	}
	return "", "", false
}

// unwrapShell strips the `<shell> -lc '<command>'` wrapper Codex puts
// around shell commands.
func unwrapShell(command string) string {
	_, inner, ok := strings.Cut(command, " -lc ")
	if !ok {
		return command
	}
	if n := len(inner); n >= 2 && (inner[0] == '\'' || inner[0] == '"') && inner[n-1] == inner[0] {
		inner = inner[1 : n-1]
	}
	return inner
}

func (t *codexTurn) itemStarted(it codexItem) {
	if it.Type == "reasoning" {
		t.emit(exponential.StreamEvent{Kind: exponential.StreamThinking})
		return
	}
	name, arg, ok := codexToolName(it)
	if !ok {
		return
	}
	t.emit(exponential.StreamEvent{Kind: exponential.StreamToolStart, Text: name, ToolArg: arg, ToolID: it.ID})
	t.emit(exponential.StreamEvent{Kind: exponential.StreamToolDone, Text: name, ToolArg: arg, ToolID: it.ID})
}

func (t *codexTurn) itemCompleted(it codexItem) {
	if it.Type == "agentMessage" {
		if !t.streamed[it.ID] && it.Text != "" {
			t.emit(exponential.StreamEvent{Kind: exponential.StreamText, Text: it.Text})
		}
		t.emit(exponential.StreamEvent{Kind: exponential.StreamToolDone}) // ends the text block
		return
	}
	if _, _, ok := codexToolName(it); !ok {
		return
	}
	ev := exponential.StreamEvent{Kind: exponential.StreamToolResult, ToolID: it.ID, ToolStatus: exponential.ToolStatusDone}
	switch {
	case it.Status == "declined":
		ev.ToolStatus = exponential.ToolStatusDenied
	case it.Status == "failed" || (it.ExitCode != nil && *it.ExitCode != 0) || (it.Result != nil && it.Result.IsError):
		ev.ToolStatus = exponential.ToolStatusError
	}
	if ev.ToolStatus != exponential.ToolStatusDone {
		switch {
		case it.Error != nil && it.Error.Message != "":
			ev.ToolMessage = it.Error.Message
		case it.Result != nil:
			ev.ToolMessage = exponential.ToolResultText(it.Result.Content)
		case it.Output != nil:
			text, _ := json.Marshal(*it.Output)
			ev.ToolMessage = exponential.ToolResultText(text)
		}
	}
	t.emit(ev)
}

// CodexModels asks `codex app-server` for its models. The first option is
// "Default model", with the efforts of Codex's default model.
func CodexModels(ctx context.Context, binary string) ([]ModelOption, error) {
	if binary == "" {
		binary = "codex"
	}
	p, err := startCodex(binary, []string{"app-server"}, "", "")
	if err != nil {
		return nil, err
	}
	defer p.stop(time.Second)

	initID := p.call("initialize", map[string]any{"clientInfo": map[string]string{"name": "xpo", "title": "xpo", "version": "1"}})
	listID := -1
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case line, ok := <-p.lines:
			if !ok {
				return nil, p.exitError()
			}
			var msg rpcMessage
			var id int
			if json.Unmarshal(line, &msg) != nil || msg.Method != "" || json.Unmarshal(msg.ID, &id) != nil {
				continue
			}
			if msg.Error != nil {
				return nil, errors.New(msg.Error.Message)
			}
			switch id {
			case initID:
				p.notify("initialized")
				listID = p.call("model/list", map[string]any{})
			case listID:
				return codexModelOptions(msg.Result)
			}
		}
	}
}

func codexModelOptions(raw json.RawMessage) ([]ModelOption, error) {
	var res struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Hidden      bool   `json:"hidden"`
			IsDefault   bool   `json:"isDefault"`
			Efforts     []struct {
				Effort string `json:"reasoningEffort"`
			} `json:"supportedReasoningEfforts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	opts := []ModelOption{codexFallbackModels[0]}
	for _, m := range res.Data {
		if m.Hidden || m.ID == "" {
			continue
		}
		label := m.DisplayName
		if label == "" {
			label = m.ID
		}
		var efforts []string
		for _, e := range m.Efforts {
			efforts = append(efforts, e.Effort)
		}
		opts = append(opts, ModelOption{ID: m.ID, Label: label, Name: label, Efforts: efforts})
		if m.IsDefault {
			opts[0] = ModelOption{ID: "", Label: "Default model", Name: label, Efforts: efforts}
		}
	}
	return opts, nil
}

// codexProc is a running `codex app-server`: a JSON-RPC writer on its stdin
// and its stdout lines on a channel.
type codexProc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan []byte
	stderr *tailBuffer

	mu     sync.Mutex
	nextID int
}

func startCodex(binary string, args []string, dir, token string) (*codexProc, error) {
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	if token != "" {
		cmd.Env = append(os.Environ(), codexTokenEnv+"="+token)
	}
	p := &codexProc{cmd: cmd, lines: make(chan []byte, 64), stderr: &tailBuffer{max: 4096}}
	cmd.Stderr = p.stderr
	setProcessGroup(cmd)
	var err error
	if p.stdin, err = cmd.StdinPipe(); err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", binary, err)
	}
	go func() {
		defer close(p.lines)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
		for sc.Scan() {
			p.lines <- append([]byte(nil), sc.Bytes()...)
		}
	}()
	return p, nil
}

func (p *codexProc) send(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = p.stdin.Write(append(data, '\n'))
}

// call sends a request and returns its ID.
func (p *codexProc) call(method string, params any) int {
	p.mu.Lock()
	p.nextID++
	id := p.nextID
	p.mu.Unlock()
	p.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	return id
}

func (p *codexProc) notify(method string) {
	p.send(map[string]any{"jsonrpc": "2.0", "method": method})
}

// exitError explains why app-server stopped before the turn ended.
func (p *codexProc) exitError() error {
	err := p.cmd.Wait()
	msg := "codex app-server exited without a reply"
	if err != nil {
		msg = fmt.Sprintf("codex app-server exited: %v", err)
	}
	if tail := strings.TrimSpace(p.stderr.String()); tail != "" {
		msg += ": " + tail
	}
	return errors.New(msg)
}

// stop closes app-server's stdin and gives it wait to exit, then ends its
// process group, including MCP servers it started.
func (p *codexProc) stop(wait time.Duration) {
	p.stdin.Close()
	exited := make(chan struct{})
	go func() {
		for range p.lines { // drain so the reader goroutine can exit
		}
		_ = p.cmd.Wait()
		close(exited)
	}()
	select {
	case <-exited:
	case <-time.After(wait):
	}
	killProcessGroup(p.cmd)
	<-exited
}
