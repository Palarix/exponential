package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// ClaudeDriver runs a turn as one `claude -p` process that streams JSON.
type ClaudeDriver struct {
	Binary string // "claude" when empty
}

func (d *ClaudeDriver) binary() string {
	if d.Binary == "" {
		return "claude"
	}
	return d.Binary
}

// Turn implements Driver.
func (d *ClaudeDriver) Turn(ctx context.Context, in TurnInput, emit exponential.StreamCallback) (*TurnResult, error) {
	mcpPath, err := writeMCPConfig(in.MCP)
	if err != nil {
		return nil, fmt.Errorf("write MCP config: %w", err)
	}
	defer os.Remove(mcpPath)

	cmd := exec.CommandContext(ctx, d.binary(), claudeArgs(turnOptions{
		MCPConfigPath: mcpPath,
		SessionID:     in.SessionID,
		Model:         in.Model,
		Effort:        in.Effort,
		Grants:        in.Grants,
	})...)
	cmd.Dir = in.Dir
	stderr := &tailBuffer{max: 4096}
	cmd.Stderr = stderr
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		killProcessGroup(cmd)
		return nil
	}
	cmd.WaitDelay = 5 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", d.binary(), err)
	}

	// Claude Code reads the prompt and the answers to its permission
	// requests from stdin, and ends once stdin closes.
	stdin := &lineWriter{w: stdinPipe}
	stdin.write(claudeUserMessage(in.Prompt))

	// Open approval requests end with the turn.
	approveCtx, endApprovals := context.WithCancel(ctx)
	var approvals sync.WaitGroup
	defer func() {
		endApprovals()
		approvals.Wait()
	}()

	var parser exponential.ClaudeStreamParser
	var result *exponential.ClaudeTurnResult
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if cr, ok := parseClaudeControlRequest(line); ok {
			approvals.Add(1)
			go func() {
				defer approvals.Done()
				stdin.write(answerClaudeControlRequest(approveCtx, in, cr))
			}()
			continue
		}
		if res := parser.Parse(line, emit); res != nil {
			result = res
			stdin.close()
		}
	}
	stdin.close()
	waitErr := cmd.Wait()

	if result == nil {
		msg := fmt.Sprintf("%s exited without a reply", d.binary())
		if waitErr != nil {
			msg = fmt.Sprintf("%s exited: %v", d.binary(), waitErr)
		}
		if tail := strings.TrimSpace(stderr.String()); tail != "" {
			msg += ": " + tail
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return &TurnResult{
		SessionID: result.SessionID,
		IsError:   result.IsError,
		Output:    result.Output,
		Duration:  result.Duration,
		CostUSD:   result.CostUSD,
		TokensIn:  result.TokensIn,
		TokensOut: result.TokensOut,

		Model:         result.Model,
		ContextWindow: result.ContextWindow,
	}, nil
}

// claudeUserMessage is the prompt as a stream-json user message.
func claudeUserMessage(prompt string) []byte {
	data, _ := json.Marshal(map[string]any{
		"type":    "user",
		"message": map[string]string{"role": "user", "content": prompt},
	})
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

// answerClaudeControlRequest asks the user about a can_use_tool request and
// returns the control_response line. Other requests get an error.
func answerClaudeControlRequest(ctx context.Context, in TurnInput, cr claudeControlRequest) []byte {
	if cr.Request.Subtype != "can_use_tool" {
		return claudeControlError(cr.RequestID, "xpo chat does not handle "+cr.Request.Subtype+" requests")
	}
	return claudePermissionResponse(cr, in.approve(ctx, claudeApproval(cr)))
}

// lineWriter writes JSON lines to a pipe from several goroutines, and
// ignores writes after close.
type lineWriter struct {
	mu     sync.Mutex
	w      io.WriteCloser
	closed bool
}

func (l *lineWriter) write(line []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	_, _ = l.w.Write(append(line, '\n'))
}

func (l *lineWriter) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		_ = l.w.Close()
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
