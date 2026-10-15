package chat

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestRealAgents_Title runs the real `claude` and `codex` CLIs, so it calls
// models. It pins that the title calls' flags still work: run it with
// XPO_TEST_REAL_CLI=1 after an agent upgrade.
func TestRealAgents_Title(t *testing.T) {
	if os.Getenv("XPO_TEST_REAL_CLI") != "1" {
		t.Skip("set XPO_TEST_REAL_CLI=1 to run against the real agent CLIs")
	}
	prompt := titlePrompt("", "", "Can the chat chip show a shield icon while a turn waits for approval?", "Yes: ChatChip.tsx renders the spinner while the turn runs.")
	catalog := NewCatalog("", exec.LookPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if models, err := CodexModels(ctx, "codex"); err == nil {
		catalog.SetModels(ProviderCodex, models)
	}
	for provider, titler := range map[string]Titler{ProviderClaude: &ClaudeTitler{}, ProviderCodex: &CodexTitler{}} {
		t.Run(provider, func(t *testing.T) {
			if _, err := exec.LookPath(provider); err != nil {
				t.Skipf("%s is not installed", provider)
			}
			model := catalog.TitleModel(provider)
			reply, err := titler.Title(ctx, t.TempDir(), model, prompt)
			if err != nil {
				t.Fatalf("title with %q: %v", model, err)
			}
			title := cleanTitle(reply)
			if n := len(strings.Fields(title)); n < 2 || n > 10 {
				t.Errorf("title %q has %d words", title, n)
			}
			t.Logf("%s (%s): %q", provider, model, title)
		})
	}
}
