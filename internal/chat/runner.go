package chat

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// ErrBusy is returned when a chat already has a turn running.
var ErrBusy = errors.New("a reply is already running")

// RunnerConfig configures how turns are run.
type RunnerConfig struct {
	Store        *Store
	Binary       string // agent CLI; "claude" when empty
	Dir          string // working directory of the agent: the project root
	MCP          MCPTarget
	DefaultModel string // chat.model from config; used when a chat has no model
}

// SendRequest is one user message.
type SendRequest struct {
	Text    string
	Context *Context
	Model   string
	Effort  string
}

// Runner runs chat turns, one agent process per turn, and fans their events
// out to subscribers.
type Runner struct {
	cfg RunnerConfig

	mu    sync.Mutex
	turns map[string]*turn
	subs  map[string]map[chan Event]struct{}
	wg    sync.WaitGroup
}

type turn struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stopped bool
	done    chan struct{} // closed when the turn has finished
}

// NewRunner returns a runner for cfg.
func NewRunner(cfg RunnerConfig) *Runner {
	if cfg.Binary == "" {
		cfg.Binary = "claude"
	}
	return &Runner{cfg: cfg, turns: map[string]*turn{}, subs: map[string]map[chan Event]struct{}{}}
}

// Send records the user message and starts a turn in the background.
func (r *Runner) Send(id string, req SendRequest) error {
	meta, err := r.cfg.Store.Meta(id)
	if err != nil {
		return err
	}

	r.mu.Lock()
	if _, busy := r.turns[id]; busy {
		r.mu.Unlock()
		return ErrBusy
	}
	t := &turn{done: make(chan struct{})}
	r.turns[id] = t
	r.wg.Add(1)
	r.mu.Unlock()

	if meta.Title == "" {
		meta.Title = TitleFrom(req.Text)
	}
	meta.Model = req.Model
	meta.Effort = req.Effort
	meta.State = StateRunning
	meta.UpdatedAt = time.Now().UTC()
	if err := r.cfg.Store.SaveMeta(meta); err != nil {
		r.finish(id)
		return err
	}
	r.settle(id, Event{Type: EventUserMessage, Text: req.Text, Context: req.Context})

	go r.run(id, meta, buildPrompt(req.Text, req.Context), t)
	return nil
}

// Running reports whether a chat has a turn in progress.
func (r *Runner) Running(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.turns[id]
	return ok
}

// Cancel stops a chat's running turn. It returns false when none is running.
func (r *Runner) Cancel(id string) bool {
	r.mu.Lock()
	t, ok := r.turns[id]
	r.mu.Unlock()
	if !ok {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
	if t.cmd != nil && t.cmd.Process != nil {
		killProcessGroup(t.cmd)
	}
	return true
}

// CancelAndWait stops a chat's running turn, if any, and waits until it has
// finished writing its events.
func (r *Runner) CancelAndWait(id string) {
	r.mu.Lock()
	t, ok := r.turns[id]
	r.mu.Unlock()
	if !ok {
		return
	}
	r.Cancel(id)
	<-t.done
}

// Wait blocks until no turn is running.
func (r *Runner) Wait() { r.wg.Wait() }

// Shutdown stops every running turn and waits for them to end.
func (r *Runner) Shutdown() {
	r.mu.Lock()
	ids := make([]string, 0, len(r.turns))
	for id := range r.turns {
		ids = append(ids, id)
	}
	r.mu.Unlock()
	for _, id := range ids {
		r.Cancel(id)
	}
	r.Wait()
}

// Subscribe returns a channel of every event published for a chat, settled
// and live. A slow subscriber misses events rather than blocking the turn.
func (r *Runner) Subscribe(id string) (<-chan Event, func()) {
	ch := make(chan Event, 256)
	r.mu.Lock()
	if r.subs[id] == nil {
		r.subs[id] = map[chan Event]struct{}{}
	}
	r.subs[id][ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.subs[id], ch)
		if len(r.subs[id]) == 0 {
			delete(r.subs, id)
		}
		r.mu.Unlock()
	}
}

func (r *Runner) publish(id string, e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.subs[id] {
		select {
		case ch <- e:
		default:
		}
	}
}

// settle persists events and publishes them as stored (with Seq).
func (r *Runner) settle(id string, events ...Event) {
	stored, err := r.cfg.Store.Append(id, events...)
	if err != nil {
		stored = events
	}
	for _, e := range stored {
		r.publish(id, e)
	}
}

func (r *Runner) live(id string, e Event) {
	e.Time = time.Now().UTC()
	r.publish(id, e)
}

func (r *Runner) finish(id string) {
	r.mu.Lock()
	if t, ok := r.turns[id]; ok {
		close(t.done)
		delete(r.turns, id)
	}
	r.mu.Unlock()
	r.wg.Done()
}

func (r *Runner) run(id string, meta Meta, prompt string, t *turn) {
	defer r.finish(id)

	sessionID := meta.ClaudeSessionID
	defer func() {
		if m, err := r.cfg.Store.Meta(id); err == nil {
			m.State = StateIdle
			m.ClaudeSessionID = sessionID
			m.UpdatedAt = time.Now().UTC()
			_ = r.cfg.Store.SaveMeta(m)
		}
	}()

	mcpPath, err := writeMCPConfig(r.cfg.MCP)
	if err != nil {
		r.settle(id, Event{Type: EventError, Message: "write MCP config: " + err.Error()})
		return
	}
	defer os.Remove(mcpPath)

	model := meta.Model
	if model == "" {
		model = r.cfg.DefaultModel
	}
	cmd := exec.Command(r.cfg.Binary, claudeArgs(turnOptions{
		MCPConfigPath: mcpPath,
		SessionID:     sessionID,
		Model:         model,
		Effort:        meta.Effort,
	})...)
	cmd.Dir = r.cfg.Dir
	cmd.Stdin = strings.NewReader(prompt)
	stderr := &tailBuffer{max: 4096}
	cmd.Stderr = stderr
	setProcessGroup(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.settle(id, Event{Type: EventError, Message: err.Error()})
		return
	}

	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		r.settle(id, Event{Type: EventTurnStopped})
		return
	}
	if err := cmd.Start(); err != nil {
		t.mu.Unlock()
		r.settle(id, Event{Type: EventError, Message: fmt.Sprintf("start %s: %v", r.cfg.Binary, err)})
		return
	}
	t.cmd = cmd
	t.mu.Unlock()

	var text strings.Builder
	flushText := func() {
		if text.Len() > 0 {
			r.settle(id, Event{Type: EventAssistantText, Text: text.String()})
			text.Reset()
		}
	}

	var parser exponential.ClaudeStreamParser
	var result *exponential.ClaudeTurnResult
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		res := parser.Parse(sc.Bytes(), func(ev exponential.StreamEvent) {
			switch ev.Kind {
			case exponential.StreamSession:
				sessionID = ev.Text
			case exponential.StreamText:
				text.WriteString(ev.Text)
				r.live(id, Event{Type: EventTextDelta, Text: ev.Text})
			case exponential.StreamThinking:
				r.live(id, Event{Type: EventThinking})
			case exponential.StreamToolStart:
				flushText()
				r.live(id, Event{Type: EventToolStart, Name: ev.Text, Arg: ev.ToolArg})
			case exponential.StreamToolDone:
				if ev.Text == "" {
					flushText() // end of a text or thinking block
					return
				}
				r.settle(id, Event{Type: EventToolCall, Name: ev.Text, Arg: ev.ToolArg, Status: "done"})
			}
		})
		if res != nil {
			result = res
			if res.SessionID != "" {
				sessionID = res.SessionID
			}
		}
	}
	waitErr := cmd.Wait()
	flushText()

	t.mu.Lock()
	stopped := t.stopped
	t.mu.Unlock()

	switch {
	case stopped:
		r.settle(id, Event{Type: EventTurnStopped})
	case result != nil && result.IsError:
		r.settle(id, Event{Type: EventError, Message: result.Output})
	case result != nil:
		r.settle(id, Event{
			Type:       EventTurnDone,
			DurationMS: result.Duration.Milliseconds(),
			CostUSD:    result.CostUSD,
			TokensIn:   result.TokensIn,
			TokensOut:  result.TokensOut,
		})
	default:
		msg := fmt.Sprintf("%s exited without a reply", r.cfg.Binary)
		if waitErr != nil {
			msg = fmt.Sprintf("%s exited: %v", r.cfg.Binary, waitErr)
		}
		if tail := strings.TrimSpace(stderr.String()); tail != "" {
			msg += ": " + tail
		}
		r.settle(id, Event{Type: EventError, Message: msg})
	}
}

// writeMCPConfig writes the agent's MCP config to a private temp file, so a
// remote token never shows up in the process list.
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

// tailBuffer keeps the last max bytes written to it.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = b.buf[len(b.buf)-b.max:]
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}
