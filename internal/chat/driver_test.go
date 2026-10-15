package chat

import (
	"context"
	"sync"
	"testing"

	"github.com/palarix/exponential/internal/exponential"
)

// recordingHandler is a SessionHandler for driver tests. It passes stream
// events to emit and reports each turn's end on ended.
type recordingHandler struct {
	emit  exponential.StreamCallback
	ended chan turnEnd

	mu         sync.Mutex
	agentTurns []string
	tasks      [][]BackgroundTask
}

type turnEnd struct {
	res *TurnResult
	err error
}

func newRecordingHandler(emit exponential.StreamCallback) *recordingHandler {
	if emit == nil {
		emit = func(exponential.StreamEvent) {}
	}
	return &recordingHandler{emit: emit, ended: make(chan turnEnd, 8)}
}

func (h *recordingHandler) AgentTurn(reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agentTurns = append(h.agentTurns, reason)
}
func (h *recordingHandler) Stream(ev exponential.StreamEvent)    { h.emit(ev) }
func (h *recordingHandler) TurnEnded(res *TurnResult, err error) { h.ended <- turnEnd{res, err} }
func (h *recordingHandler) Tasks(tasks []BackgroundTask) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tasks = append(h.tasks, tasks)
}
func (h *recordingHandler) Exited(error) {}

// runTurn opens a session on d, runs one turn with prompt, and closes the
// session.
func runTurn(ctx context.Context, t *testing.T, d Driver, in SessionInput, prompt string, emit exponential.StreamCallback) (*TurnResult, error) {
	t.Helper()
	h := newRecordingHandler(emit)
	s, err := d.Open(in, h)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	if err := s.Send(Prompt{Text: prompt}, in.Model, in.Effort); err != nil {
		return nil, err
	}
	select {
	case e := <-h.ended:
		return e.res, e.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
