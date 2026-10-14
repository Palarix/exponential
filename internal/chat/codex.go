package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// CodexDriver runs a chat on one `codex app-server` process and one
// thread, speaking its JSON-RPC protocol over stdio. The process lives
// across turns. The session's profile picks the sandbox, the approval
// policy, and which xpo MCP tools ask or are disabled; the others are
// pre-approved. Approval requests go to the user.
type CodexDriver struct {
	Binary string // "codex" when empty
	// StopGrace is how long Close waits after SIGTERM before it kills the
	// process group. 3s when zero.
	StopGrace time.Duration
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
	// Live web search in both modes, like Claude's WebSearch (xpo-80bf05).
	// It is OpenAI's hosted tool, so the read-only sandbox does not block it.
	args = append(args, "-c", "web_search="+tomlValue("live"))
	return args
}

// tomlValue encodes a string or string list as TOML. JSON strings are valid
// TOML basic strings.
func tomlValue(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// codexCacheTTL is the prompt cache TTL Codex turns record. Codex does not
// report one; OpenAI keeps cached prefixes for 5 to 10 minutes of
// inactivity, so xpo takes the lower bound.
const codexCacheTTL = 5 * time.Minute

// Open implements Driver. It starts app-server and its thread; a Send
// before the thread is ready starts its turn once it is.
func (d *CodexDriver) Open(in SessionInput, h SessionHandler) (Session, error) {
	p, err := startCodex(d.binary(), codexArgs(in.MCP, in.Profile), in.Dir, in.MCP.RemoteToken)
	if err != nil {
		return nil, err
	}
	grace := d.StopGrace
	if grace == 0 {
		grace = 3 * time.Second
	}
	s := &codexSession{
		p: p, in: in, h: h, grace: grace,
		usage: map[string][2]int{}, streamed: map[string]bool{},
		commands: map[string]BackgroundTask{}, background: map[string]BackgroundTask{},
		done: make(chan struct{}),
	}
	s.exitCtx, s.exitCancel = context.WithCancel(context.Background())
	s.mu.Lock()
	s.initID = p.call("initialize", map[string]any{"clientInfo": map[string]string{"name": "xpo", "title": "xpo", "version": "1"}})
	s.mu.Unlock()
	go s.read()
	return s, nil
}

// codexSession is a running `codex app-server` with one thread.
type codexSession struct {
	p     *codexProc
	in    SessionInput
	h     SessionHandler
	grace time.Duration
	done  chan struct{} // closed after Exited

	// exitCtx ends with the process. Approval requests outside a turn use it.
	exitCtx    context.Context
	exitCancel context.CancelFunc
	approvals  sync.WaitGroup

	// mu guards everything below. The reader holds it while it handles a
	// line; Send holds it while it starts a turn.
	mu                  sync.Mutex
	initID, threadReqID int
	turnReqID           int
	threadID            string
	ready               bool        // the thread started or resumed
	failed              bool        // the thread could not start; the session ends
	queued              *codexInput // a turn sent before the thread was ready
	inTurn              bool
	turnID              string
	approveCtx          context.Context
	endApprovals        context.CancelFunc
	model               string                    // from the thread response
	window              int                       // the latest modelContextWindow
	usage               map[string][2]int         // input and output tokens by turn ID
	streamed            map[string]bool           // agent message items that sent deltas
	commands            map[string]BackgroundTask // commands of the running turn still in progress
	background          map[string]BackgroundTask // commands that outlived their turn
	backgroundOrder     []string
}

// codexInput is one user turn.
type codexInput struct{ prompt, model, effort string }

func (s *codexSession) PID() int { return s.p.pid() }

// Send implements Session. Codex takes the model and effort on each
// turn/start, so a switch needs no extra request.
func (s *codexSession) Send(prompt, model, effort string) error {
	select {
	case <-s.p.waited:
		return agentExited{s.p.exitError()}
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed {
		return errAgentExited
	}
	s.beginTurn()
	in := &codexInput{prompt, model, effort}
	if !s.ready {
		s.queued = in
		return nil
	}
	s.startTurn(in)
	return nil
}

// startTurn sends turn/start. s.mu must be held.
func (s *codexSession) startTurn(in *codexInput) {
	params := map[string]any{
		"threadId": s.threadID,
		"input":    []map[string]any{{"type": "text", "text": in.prompt}},
	}
	if in.model != "" {
		params["model"] = in.model
	}
	if in.effort != "" {
		params["effort"] = in.effort
	}
	s.turnReqID = s.p.call("turn/start", params)
}

// Interrupt implements Session.
func (s *codexSession) Interrupt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inTurn && s.turnID != "" {
		s.p.call("turn/interrupt", map[string]string{"threadId": s.threadID, "turnId": s.turnID})
	}
}

// Close implements Session.
func (s *codexSession) Close() {
	s.p.stop(s.grace)
	<-s.done
}

// beginTurn resets the turn state. s.mu must be held.
func (s *codexSession) beginTurn() {
	s.inTurn = true
	s.turnID = ""
	clear(s.commands)
	s.approveCtx, s.endApprovals = context.WithCancel(context.Background())
}

// endTurn ends the turn and reports it. s.mu must be held.
func (s *codexSession) endTurn(res *TurnResult, err error) {
	s.inTurn = false
	s.endApprovals()
	s.approvals.Wait() // their cards settle before the turn's end
	// Commands still in progress outlive the turn: they are background
	// tasks now (codex-cli 0.162.1 completes them later, outside a turn).
	if len(s.commands) > 0 {
		for id, t := range s.commands {
			if _, ok := s.background[id]; !ok {
				s.backgroundOrder = append(s.backgroundOrder, id)
			}
			s.background[id] = t
		}
		clear(s.commands)
		s.h.Tasks(s.taskList())
	}
	s.h.TurnEnded(res, err)
}

func (s *codexSession) taskList() []BackgroundTask {
	out := make([]BackgroundTask, 0, len(s.backgroundOrder))
	for _, id := range s.backgroundOrder {
		out = append(out, s.background[id])
	}
	return out
}

func (s *codexSession) failedResult(msg string) *TurnResult {
	return &TurnResult{SessionID: s.threadID, IsError: true, Output: msg}
}

// read handles app-server's output until it exits.
func (s *codexSession) read() {
	defer close(s.done)
	for line := range s.p.lines {
		s.mu.Lock()
		s.handle(line)
		s.mu.Unlock()
	}
	err := s.p.exitError()
	s.exitCancel()
	s.mu.Lock()
	if s.inTurn {
		s.inTurn = false
		s.endApprovals()
		s.approvals.Wait()
		s.h.TurnEnded(nil, err)
	}
	if len(s.background) > 0 {
		clear(s.background)
		s.backgroundOrder = nil
		s.h.Tasks(nil)
	}
	s.mu.Unlock()
	s.approvals.Wait()
	s.h.Exited(err)
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

// handle processes one line from app-server. s.mu is held.
func (s *codexSession) handle(line []byte) {
	var msg rpcMessage
	if json.Unmarshal(line, &msg) != nil {
		return
	}
	switch {
	case msg.Method != "" && len(msg.ID) > 0:
		// A server request. Approvals go to the user; anything else is
		// refused, so the turn cannot hang on it.
		if req, answer, ok := codexApproval(msg.Method, msg.Params); ok {
			ctx := s.exitCtx
			if s.inTurn {
				ctx = s.approveCtx
			}
			s.approvals.Add(1)
			go func() {
				defer s.approvals.Done()
				result := answer(s.in.approve(ctx, req).Decision)
				s.p.send(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": result})
			}()
			return
		}
		s.p.send(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "error": map[string]any{"code": -32000, "message": "xpo chat declines this request"}})
		return
	case msg.Method != "":
		s.notification(msg.Method, msg.Params)
		return
	}

	var id int
	if json.Unmarshal(msg.ID, &id) != nil {
		return
	}
	var errText string
	if msg.Error != nil {
		errText = msg.Error.Message
	}
	switch id {
	case s.initID:
		if errText != "" {
			s.fail("Codex did not start: " + errText)
			return
		}
		s.p.notify("initialized")
		params := map[string]any{
			"cwd":                   s.in.Dir,
			"sandbox":               s.in.Profile.codexSandbox(),
			"approvalPolicy":        s.in.Profile.codexApprovalPolicy(),
			"developerInstructions": systemPromptFor(ProviderCodex),
		}
		method := "thread/start"
		if s.in.SessionID != "" {
			method = "thread/resume"
			params["threadId"] = s.in.SessionID
			params["excludeTurns"] = true
		}
		s.threadReqID = s.p.call(method, params)
	case s.threadReqID:
		if errText != "" {
			if s.in.SessionID != "" {
				s.fail(fmt.Sprintf("Codex could not resume this chat (%s). Start a new chat.", errText))
				return
			}
			s.fail("Codex could not start a thread: " + errText)
			return
		}
		var res struct {
			Thread struct {
				ID string `json:"id"`
			} `json:"thread"`
			Model string `json:"model"`
		}
		json.Unmarshal(msg.Result, &res)
		s.threadID = res.Thread.ID
		s.model = res.Model
		s.ready = true
		s.h.Stream(exponential.StreamEvent{Kind: exponential.StreamSession, Text: s.threadID})
		if s.queued != nil {
			s.startTurn(s.queued)
			s.queued = nil
		}
	case s.turnReqID:
		if errText != "" {
			if s.inTurn {
				s.endTurn(s.failedResult("Codex could not start the turn: "+errText), nil)
			}
			return
		}
		var res struct {
			Turn struct {
				ID string `json:"id"`
			} `json:"turn"`
		}
		json.Unmarshal(msg.Result, &res)
		if s.inTurn && s.turnID == "" {
			s.turnID = res.Turn.ID
		}
	}
}

// fail ends the session: the thread could not start. A waiting turn ends
// with msg. s.mu is held.
func (s *codexSession) fail(msg string) {
	s.failed = true
	if s.inTurn {
		s.queued = nil
		s.endTurn(s.failedResult(msg), nil)
	}
	s.p.stdin.close()
}

func (s *codexSession) notification(method string, raw json.RawMessage) {
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
		return
	}

	switch method {
	case "turn/started":
		switch {
		case s.inTurn && (s.turnID == "" || s.turnID == p.Turn.ID):
			s.turnID = p.Turn.ID
		case !s.inTurn:
			// Not a turn xpo asked for: the agent started it.
			s.beginTurn()
			s.turnID = p.Turn.ID
			s.h.AgentTurn("")
		}
		return
	case "thread/tokenUsage/updated":
		// thread/resume replays the last update of the previous turn, so
		// usage is kept per turn.
		u := s.usage[p.TurnID]
		s.usage[p.TurnID] = [2]int{u[0] + p.TokenUsage.Last.InputTokens, u[1] + p.TokenUsage.Last.OutputTokens}
		if p.TokenUsage.ModelContextWindow > 0 {
			s.window = p.TokenUsage.ModelContextWindow
		}
		return
	case "item/completed":
		// A background command that outlived its turn ended.
		if _, ok := s.background[p.Item.ID]; ok {
			delete(s.background, p.Item.ID)
			s.backgroundOrder = removeString(s.backgroundOrder, p.Item.ID)
			s.h.Tasks(s.taskList())
			return
		}
	}
	if !s.inTurn || (p.TurnID != "" && s.turnID != "" && p.TurnID != s.turnID) {
		return
	}

	switch method {
	case "item/agentMessage/delta":
		s.streamed[p.ItemID] = true
		s.h.Stream(exponential.StreamEvent{Kind: exponential.StreamText, Text: p.Delta})
	case "item/started":
		s.itemStarted(p.Item)
	case "item/completed":
		s.itemCompleted(p.Item)
	case "turn/completed":
		if s.turnID != "" && p.Turn.ID != s.turnID {
			return
		}
		u := s.usage[p.Turn.ID]
		res := &TurnResult{
			SessionID: s.threadID,
			Duration:  time.Duration(p.Turn.DurationMS) * time.Millisecond,
			TokensIn:  u[0],
			TokensOut: u[1],

			Model:         s.model,
			ContextWindow: s.window,
			CacheTTL:      codexCacheTTL,
		}
		if p.Turn.Status == "failed" {
			res.IsError = true
			res.Output = "Codex turn failed"
			if p.Turn.Error != nil && p.Turn.Error.Message != "" {
				res.Output = p.Turn.Error.Message
			}
		}
		s.endTurn(res, nil)
	}
}

// codexApproval turns an app-server approval request into an
// ApprovalRequest, and returns how to answer it for a decision. ok is false
// for requests that are not approvals xpo handles.
func codexApproval(method string, raw json.RawMessage) (req ApprovalRequest, answer func(decision string) any, ok bool) {
	var p struct {
		ItemID  string `json:"itemId"`
		Command string `json:"command"`
		Actions []struct {
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

func (s *codexSession) itemStarted(it codexItem) {
	if it.Type == "reasoning" {
		s.h.Stream(exponential.StreamEvent{Kind: exponential.StreamThinking})
		return
	}
	name, arg, ok := codexToolName(it)
	if !ok {
		return
	}
	if it.Type == "commandExecution" {
		s.commands[it.ID] = BackgroundTask{ID: it.ID, Kind: TaskMonitor, Description: arg}
	}
	s.h.Stream(exponential.StreamEvent{Kind: exponential.StreamToolStart, Text: name, ToolArg: arg, ToolID: it.ID})
	s.h.Stream(exponential.StreamEvent{Kind: exponential.StreamToolDone, Text: name, ToolArg: arg, ToolID: it.ID})
}

func (s *codexSession) itemCompleted(it codexItem) {
	if it.Type == "agentMessage" {
		if !s.streamed[it.ID] && it.Text != "" {
			s.h.Stream(exponential.StreamEvent{Kind: exponential.StreamText, Text: it.Text})
		}
		s.h.Stream(exponential.StreamEvent{Kind: exponential.StreamToolDone}) // ends the text block
		return
	}
	if _, _, ok := codexToolName(it); !ok {
		return
	}
	delete(s.commands, it.ID)
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
	s.h.Stream(ev)
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
	defer func() {
		go func() {
			for range p.lines { // stop needs the output drained
			}
		}()
		p.stop(time.Second)
	}()

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

// codexProc is a running `codex app-server`: a JSON-RPC writer on its
// stdin and its stdout lines on a channel.
type codexProc struct {
	*agentProc

	mu     sync.Mutex
	nextID int
}

func startCodex(binary string, args []string, dir, token string) (*codexProc, error) {
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	if token != "" {
		cmd.Env = append(os.Environ(), codexTokenEnv+"="+token)
	}
	p, err := startAgent(cmd)
	if err != nil {
		return nil, err
	}
	return &codexProc{agentProc: p}, nil
}

func (p *codexProc) send(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	p.stdin.write(data)
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

// exitError waits for app-server to exit and explains why it stopped.
func (p *codexProc) exitError() error {
	<-p.exited
	msg := "codex app-server exited without a reply"
	if p.waitErr != nil {
		msg = fmt.Sprintf("codex app-server exited: %v", p.waitErr)
	}
	if tail := strings.TrimSpace(p.stderr.String()); tail != "" {
		msg += ": " + tail
	}
	return errors.New(msg)
}
