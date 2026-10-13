package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	})...)
	cmd.Dir = in.Dir
	cmd.Stdin = strings.NewReader(in.Prompt)
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
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", d.binary(), err)
	}

	var parser exponential.ClaudeStreamParser
	var result *exponential.ClaudeTurnResult
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		if res := parser.Parse(sc.Bytes(), emit); res != nil {
			result = res
		}
	}
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
	}, nil
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
