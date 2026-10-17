package chat

import (
	"bytes"
	"context"
	"encoding/base64"
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

// ClaudeDriver runs a chat on one `claude -p` process that reads and
// writes stream-json. The process lives across turns: each user message is
// one more line on its stdin, and each `result` line ends a turn.
type ClaudeDriver struct {
	Binary string // "claude" when empty
	// StopGrace is how long Close waits after SIGTERM before it kills the
	// process group. 3s when zero.
	StopGrace time.Duration
}

func (d *ClaudeDriver) binary() string {
	if d.Binary == "" {
		return "claude"
	}
	return d.Binary
}

// controlTimeout bounds how long Send waits for Claude Code to confirm a
// model or effort switch.
const controlTimeout = 10 * time.Second

// Open implements Driver.
func (d *ClaudeDriver) Open(in SessionInput, h SessionHandler) (Session, error) {
	mcpPath, err := writeMCPConfig(in.MCP)
	if err != nil {
		return nil, fmt.Errorf("write MCP config: %w", err)
	}
	cmd := exec.Command(d.binary(), claudeArgs(turnOptions{
		MCPConfigPath: mcpPath,
		SessionID:     in.SessionID,
		Model:         in.Model,
		Effort:        in.Effort,
		Grants:        in.Grants,
		Profile:       in.Profile,
	})...)
	cmd.Dir = in.Dir
	p, err := startAgent(cmd)
	if err != nil {
		os.Remove(mcpPath)
		return nil, err
	}
	grace := d.StopGrace
	if grace == 0 {
		grace = 3 * time.Second
	}
	s := &claudeSession{
		p: p, in: in, h: h, binary: d.binary(), grace: grace,
		model: in.Model, effort: in.Effort,
		tasks:   map[string]BackgroundTask{},
		control: map[string]chan error{},
		done:    make(chan struct{}),
	}
	s.exitCtx, s.exitCancel = context.WithCancel(context.Background())
	go s.read(mcpPath)
	return s, nil
}

// claudeSession is a running `claude` process.
type claudeSession struct {
	p      *agentProc
	in     SessionInput
	h      SessionHandler
	binary string
	grace  time.Duration
	done   chan struct{} // closed after Exited

	// exitCtx ends with the process. Approval requests outside a turn use it.
	exitCtx    context.Context
	exitCancel context.CancelFunc

	// Send waits for its control requests; the reader resolves them.
	ctlMu   sync.Mutex
	control map[string]chan error // by request ID
	nextCtl int

	// mu guards the turn state below. The reader holds it while it handles
	// a line; Send holds it while it starts a turn.
	mu            sync.Mutex
	model, effort string // what the session runs now
	inTurn        bool
	parser        *exponential.ClaudeStreamParser
	endApprovals  context.CancelFunc
	approveCtx    context.Context
	approvals     sync.WaitGroup
	tasks         map[string]BackgroundTask // by task ID
	taskOrder     []string
	wake          string // why the agent may start a turn next: the last task that ended
}

func (s *claudeSession) PID() int { return s.p.pid() }

// Send implements Session.
func (s *claudeSession) Send(prompt Prompt, model, effort string) error {
	select {
	case <-s.p.waited:
		return agentExited{s.exitError()}
	default:
	}
	msg, err := claudeUserMessage(prompt)
	if err != nil {
		return err
	}
	s.mu.Lock()
	switchModel, switchEffort := model != s.model, effort != s.effort
	s.mu.Unlock()
	if switchModel {
		req := map[string]any{"subtype": "set_model", "model": nil}
		if model != "" {
			req["model"] = model
		}
		if err := s.controlRequest(req); err != nil {
			return fmt.Errorf("switch the model to %q: %w", model, err)
		}
	}
	if switchEffort {
		var level any
		if effort != "" {
			level = effort
		}
		if err := s.controlRequest(map[string]any{"subtype": "apply_flag_settings", "settings": map[string]any{"effortLevel": level}}); err != nil {
			return fmt.Errorf("switch the effort to %q: %w", effort, err)
		}
	}
	s.mu.Lock()
	s.model, s.effort = model, effort
	if !s.inTurn {
		s.beginTurn()
	}
	s.mu.Unlock()
	s.p.stdin.write(msg)
	return nil
}

// Interrupt implements Session.
func (s *claudeSession) Interrupt() {
	s.ctlMu.Lock()
	s.nextCtl++
	id := fmt.Sprintf("xpo-%d", s.nextCtl)
	s.ctlMu.Unlock()
	s.p.stdin.write(claudeControlLine(id, map[string]any{"subtype": "interrupt"}))
}

// Close implements Session.
func (s *claudeSession) Close() {
	s.p.stop(s.grace)
	<-s.done
}

// controlRequest sends a control request and waits for its response.
func (s *claudeSession) controlRequest(req map[string]any) error {
	ch := make(chan error, 1)
	s.ctlMu.Lock()
	s.nextCtl++
	id := fmt.Sprintf("xpo-%d", s.nextCtl)
	s.control[id] = ch
	s.ctlMu.Unlock()
	defer func() {
		s.ctlMu.Lock()
		delete(s.control, id)
		s.ctlMu.Unlock()
	}()
	s.p.stdin.write(claudeControlLine(id, req))
	select {
	case err := <-ch:
		return err
	case <-s.p.exited:
		return fmt.Errorf("%s exited", s.binary)
	case <-time.After(controlTimeout):
		return errors.New("no answer from " + s.binary)
	}
}

// beginTurn resets the turn state. s.mu must be held.
func (s *claudeSession) beginTurn() {
	s.inTurn = true
	s.parser = &exponential.ClaudeStreamParser{}
	s.approveCtx, s.endApprovals = context.WithCancel(context.Background())
}

// endTurn ends open approval requests with the turn, and waits until
// their cards are resolved, so they settle before the turn's end. s.mu must
// be held.
func (s *claudeSession) endTurn() {
	s.inTurn = false
	if s.endApprovals != nil {
		s.endApprovals()
	}
	s.approvals.Wait()
}

// read handles the agent's output until it exits.
func (s *claudeSession) read(mcpPath string) {
	defer close(s.done)
	defer os.Remove(mcpPath)
	for line := range s.p.lines {
		s.mu.Lock()
		s.handle(line)
		s.mu.Unlock()
	}
	<-s.p.exited
	err := s.exitError()
	s.exitCancel()
	s.mu.Lock()
	if s.inTurn {
		s.endTurn()
		s.h.TurnEnded(nil, err)
	}
	if len(s.tasks) > 0 {
		clear(s.tasks)
		s.taskOrder = nil
		s.h.Tasks(nil)
	}
	s.mu.Unlock()
	s.approvals.Wait()
	s.h.Exited(err)
}

// exitError explains why the agent exited.
func (s *claudeSession) exitError() error {
	msg := fmt.Sprintf("%s exited without a reply", s.binary)
	if s.p.waitErr != nil {
		msg = fmt.Sprintf("%s exited: %v", s.binary, s.p.waitErr)
	}
	if tail := strings.TrimSpace(s.p.stderr.String()); tail != "" {
		msg += ": " + tail
	}
	return errors.New(msg)
}

// claudeLine is the part of an output line the session reads before the
// stream parser does.
type claudeLine struct {
	Type     string `json:"type"`
	Subtype  string `json:"subtype"`
	Response struct {
		Subtype   string `json:"subtype"`
		RequestID string `json:"request_id"`
		Error     string `json:"error"`
	} `json:"response"`
	TaskID      string          `json:"task_id"`
	TaskType    string          `json:"task_type"`
	Description string          `json:"description"`
	Summary     string          `json:"summary"`
	Status      string          `json:"status"`
	Patch       json.RawMessage `json:"patch"`
	Tasks       []struct {
		TaskID      string `json:"task_id"`
		TaskType    string `json:"task_type"`
		Description string `json:"description"`
	} `json:"tasks"`
}

// handle processes one output line. s.mu is held.
func (s *claudeSession) handle(line []byte) {
	if cr, ok := parseClaudeControlRequest(line); ok {
		ctx := s.exitCtx
		if s.inTurn {
			ctx = s.approveCtx
		}
		s.approvals.Add(1)
		go func() {
			defer s.approvals.Done()
			s.p.stdin.write(answerClaudeControlRequest(ctx, s.in, cr))
		}()
		return
	}
	var msg claudeLine
	if json.Unmarshal(line, &msg) != nil {
		return
	}
	switch msg.Type {
	case "control_response":
		s.resolveControl(msg.Response.RequestID, msg.Response.Subtype, msg.Response.Error)
		return
	case "system":
		if s.handleTask(msg) {
			return
		}
	}
	// Turn content while no turn runs: the agent started one on its own.
	if !s.inTurn && startsTurn(msg) {
		s.beginTurn()
		reason := s.wake
		s.wake = ""
		s.h.AgentTurn(reason)
	}
	if !s.inTurn {
		return
	}
	if res := s.parser.Parse(line, s.h.Stream); res != nil {
		s.endTurn()
		s.h.TurnEnded(&TurnResult{
			SessionID: res.SessionID,
			IsError:   res.IsError,
			Output:    res.Output,
			Duration:  res.Duration,
			CostUSD:   res.CostUSD,
			TokensIn:  res.TokensIn,
			TokensOut: res.TokensOut,

			Model:         res.Model,
			ContextWindow: res.ContextWindow,
			CacheTTL:      res.CacheTTL,
		}, nil)
	}
}

// startsTurn reports whether a line belongs to a turn.
func startsTurn(msg claudeLine) bool {
	switch msg.Type {
	case "assistant", "stream_event", "user":
		return true
	case "system":
		return msg.Subtype == "init"
	}
	return false
}

func (s *claudeSession) resolveControl(id, subtype, errText string) {
	s.ctlMu.Lock()
	ch, ok := s.control[id]
	s.ctlMu.Unlock()
	if !ok {
		return
	}
	if subtype == "error" {
		if errText == "" {
			errText = "refused"
		}
		ch <- errors.New(errText)
		return
	}
	ch <- nil
}

// handleTask tracks background tasks from task_* and
// background_tasks_changed lines. It reports whether the line was one.
func (s *claudeSession) handleTask(msg claudeLine) bool {
	switch msg.Subtype {
	case "background_tasks_changed":
		// The roster: Claude Code's own list of live tasks.
		clear(s.tasks)
		s.taskOrder = nil
		for _, t := range msg.Tasks {
			s.addTask(t.TaskID, t.TaskType, t.Description)
		}
	case "task_started":
		s.addTask(msg.TaskID, msg.TaskType, msg.Description)
	case "task_updated":
		var patch struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(msg.Patch, &patch)
		if !terminalTaskStatus(patch.Status) {
			return true
		}
		s.removeTask(msg.TaskID)
	case "task_notification":
		desc := s.tasks[msg.TaskID].Description
		s.removeTask(msg.TaskID)
		s.wake = msg.Summary
		if s.wake == "" {
			s.wake = desc
		}
	default:
		return false
	}
	s.h.Tasks(s.taskList())
	return true
}

func (s *claudeSession) addTask(id, taskType, desc string) {
	if id == "" || inertTaskTypes[taskType] {
		return
	}
	kind := TaskAgent
	if monitorTaskTypes[taskType] {
		kind = TaskMonitor
	}
	if _, ok := s.tasks[id]; !ok {
		s.taskOrder = append(s.taskOrder, id)
	}
	s.tasks[id] = BackgroundTask{ID: id, Kind: kind, Description: desc}
}

func (s *claudeSession) removeTask(id string) {
	if _, ok := s.tasks[id]; !ok {
		return
	}
	delete(s.tasks, id)
	s.taskOrder = removeString(s.taskOrder, id)
}

func (s *claudeSession) taskList() []BackgroundTask {
	out := make([]BackgroundTask, 0, len(s.taskOrder))
	for _, id := range s.taskOrder {
		out = append(out, s.tasks[id])
	}
	return out
}

func terminalTaskStatus(status string) bool {
	switch status {
	case "completed", "failed", "stopped", "cancelled", "killed", "interrupted":
		return true
	}
	return false
}

func removeString(list []string, s string) []string {
	out := list[:0]
	for _, v := range list {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

// claudeUserMessage is the prompt as a stream-json user message. Images go
// in as base64 image blocks before the text.
func claudeUserMessage(prompt Prompt) ([]byte, error) {
	var content any = prompt.Text
	if len(prompt.Images) > 0 {
		blocks := make([]map[string]any, 0, len(prompt.Images)+1)
		for _, img := range prompt.Images {
			data, err := os.ReadFile(img.Path)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, map[string]any{
				"type":   "image",
				"source": map[string]string{"type": "base64", "media_type": img.MIME, "data": base64.StdEncoding.EncodeToString(data)},
			})
		}
		content = append(blocks, map[string]any{"type": "text", "text": prompt.Text})
	}
	return json.Marshal(map[string]any{
		"type":    "user",
		"message": map[string]any{"role": "user", "content": content},
	})
}

// claudeControlLine is a control_request from xpo to Claude Code.
func claudeControlLine(id string, req map[string]any) []byte {
	data, _ := json.Marshal(map[string]any{"type": "control_request", "request_id": id, "request": req})
	return data
}

// parseClaudeControlRequest reports whether line is a control_request.
func parseClaudeControlRequest(line []byte) (claudeControlRequest, bool) {
	var cr struct {
		Type string `json:"type"`
		claudeControlRequest
	}
	if !bytes.Contains(line, []byte(`"control_request"`)) || json.Unmarshal(line, &cr) != nil || cr.Type != "control_request" {
		return claudeControlRequest{}, false
	}
	return cr.claudeControlRequest, true
}

// answerClaudeControlRequest asks the user about a can_use_tool request, an
// AskUserQuestion call included, and returns the control_response line.
// Other requests get an error.
func answerClaudeControlRequest(ctx context.Context, in SessionInput, cr claudeControlRequest) []byte {
	if cr.Request.Subtype != "can_use_tool" {
		return claudeControlError(cr.RequestID, "xpo chat does not handle "+cr.Request.Subtype+" requests")
	}
	return claudePermissionResponse(cr, in.approve(ctx, claudeApproval(cr)))
}

// writeMCPConfig writes the agent's MCP config to a private temp file.
func writeMCPConfig(t MCPTarget) (string, error) {
	data, err := mcpConfig(t)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "xpo-chat-mcp-*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
