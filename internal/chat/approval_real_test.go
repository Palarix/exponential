package chat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// TestRealClaude_Approval runs the real `claude` CLI, so it calls a model.
// It pins the control_request / control_response protocol: run it with
// XPO_TEST_REAL_CLI=1 after a Claude Code upgrade.
func TestRealClaude_Approval(t *testing.T) {
	if os.Getenv("XPO_TEST_REAL_CLI") != "1" {
		t.Skip("set XPO_TEST_REAL_CLI=1 to run against the real claude CLI")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude is not installed")
	}

	// Claude Code asks before it reads a file outside the working directory.
	work, outside := t.TempDir(), t.TempDir()
	secret := filepath.Join(outside, "note.txt")
	os.WriteFile(secret, []byte("pelican-42\n"), 0o644)

	prompt := "Use the Read tool on " + secret + " and reply with its exact content. If you cannot read it, reply with the single word DENIED."
	turn := func(t *testing.T, grants []string, approve Approver) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		var reply strings.Builder
		res, err := (&ClaudeDriver{}).Turn(ctx, TurnInput{
			Prompt:  prompt,
			Model:   "haiku",
			Dir:     work,
			MCP:     MCPTarget{Executable: "/nonexistent/xpo"},
			Approve: approve,
			Grants:  grants,
		}, func(ev exponential.StreamEvent) {
			if ev.Kind == exponential.StreamText {
				reply.WriteString(ev.Text)
			}
		})
		if err != nil || res == nil || res.IsError {
			t.Fatalf("turn: %+v, %v", res, err)
		}
		return reply.String()
	}

	for _, c := range []struct {
		decision string
		want     bool // the reply contains the file content
	}{
		{DecisionAllow, true},
		{DecisionDeny, false},
	} {
		t.Run(c.decision, func(t *testing.T) {
			var asked []ApprovalRequest
			reply := turn(t, nil, func(_ context.Context, req ApprovalRequest) string {
				asked = append(asked, req)
				return c.decision
			})
			if len(asked) == 0 || asked[0].Tool != "Read" || asked[0].Detail != secret || asked[0].ToolID == "" || asked[0].Rule == "" {
				t.Fatalf("approval requests = %+v, want one for Read %s", asked, secret)
			}
			if got := strings.Contains(reply, "pelican-42"); got != c.want {
				t.Errorf("reply %q: content shown = %v, want %v", reply, got, c.want)
			}
		})
	}

	// A rule granted for the chat goes to the next turn as an --allowedTools
	// value, and Claude Code no longer asks.
	t.Run("grant", func(t *testing.T) {
		var rule string
		turn(t, nil, func(_ context.Context, req ApprovalRequest) string {
			rule = req.Rule
			return DecisionAllowSession
		})
		if rule == "" {
			t.Fatal("no approval request")
		}
		reply := turn(t, []string{rule}, func(_ context.Context, req ApprovalRequest) string {
			t.Errorf("asked again for %+v although %q is granted", req, rule)
			return DecisionDeny
		})
		if !strings.Contains(reply, "pelican-42") {
			t.Errorf("reply %q, want the content", reply)
		}
	})
}
