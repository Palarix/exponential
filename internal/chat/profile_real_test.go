package chat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// TestRealAgents_ChatProfile runs the real `claude` and `codex` CLIs, so it
// calls models. It pins that the Chat Mode profile makes start ask and
// keeps merge from running: run it with XPO_TEST_REAL_CLI=1 after an agent
// upgrade. Every approval is denied, so no issue changes.
func TestRealAgents_ChatProfile(t *testing.T) {
	if os.Getenv("XPO_TEST_REAL_CLI") != "1" {
		t.Skip("set XPO_TEST_REAL_CLI=1 to run against the real agent CLIs")
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	dir := strings.TrimSpace(string(root))
	xpo := filepath.Join(t.TempDir(), "xpo")
	if out, err := exec.Command("go", "build", "-o", xpo, "github.com/palarix/exponential/cmd/exponential").CombinedOutput(); err != nil {
		t.Fatalf("build xpo: %v\n%s", err, out)
	}

	drivers := map[string]Driver{ProviderClaude: &ClaudeDriver{}, ProviderCodex: &CodexDriver{}}
	models := map[string]string{ProviderClaude: "haiku"}
	for provider, driver := range drivers {
		t.Run(provider, func(t *testing.T) {
			if _, err := exec.LookPath(provider); err != nil {
				t.Skip(provider + " is not installed")
			}
			turn := func(t *testing.T, prompt string) ([]ApprovalRequest, map[string]string) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
				defer cancel()
				var mu sync.Mutex
				var asked []ApprovalRequest
				calls := map[string]string{} // tool name → status
				p := ChatProfile(dir, true)
				_, err := runTurn(ctx, t, driver, SessionInput{
					Model:   models[provider],
					Dir:     p.Dir,
					MCP:     MCPTarget{Executable: xpo},
					Profile: p,
					Approve: func(_ context.Context, req ApprovalRequest) Verdict {
						mu.Lock()
						asked = append(asked, req)
						mu.Unlock()
						return Verdict{Decision: DecisionDeny}
					},
				}, buildPrompt(prompt, nil, p, nil), func(ev exponential.StreamEvent) {
					if ev.Kind == exponential.StreamToolDone && ev.Text != "" {
						mu.Lock()
						calls[ev.Text] = "called"
						mu.Unlock()
					}
				})
				if err != nil {
					t.Logf("turn: %v", err)
				}
				return asked, calls
			}

			t.Run("start asks", func(t *testing.T) {
				asked, _ := turn(t, "Your first and only tool call must be the xpo MCP tool `start` with the argument id \"xpo-000000\". Do not call `code_history`, `show`, `list` or any other tool. Then report what happened in one sentence.")
				found := false
				for _, a := range asked {
					found = found || a.Tool == "mcp__xpo__start"
				}
				if !found {
					t.Errorf("approval requests = %+v, want one for mcp__xpo__start", asked)
				}
			})
			t.Run("merge is unavailable", func(t *testing.T) {
				asked, calls := turn(t, "Call the xpo `merge` tool with id \"xpo-000000\" now. Do not call any other tool. If the tool is not available, reply with the single word UNAVAILABLE.")
				for _, a := range asked {
					if a.Tool == "mcp__xpo__merge" {
						t.Errorf("merge asked although Chat Mode denies it: %+v", a)
					}
				}
				if provider == ProviderCodex && calls["mcp__xpo__merge"] != "" {
					t.Errorf("Codex called merge although it is disabled")
				}
			})
		})
	}
}
