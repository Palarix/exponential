package chat

import (
	"errors"
	"strings"
	"testing"
)

func lookPathFor(installed ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		for _, n := range installed {
			if n == name {
				return "/usr/local/bin/" + name, nil
			}
		}
		return "", errors.New("not found")
	}
}

func provider(st AgentStatus, id string) *ProviderStatus {
	for i := range st.Providers {
		if st.Providers[i].ID == id {
			return &st.Providers[i]
		}
	}
	return nil
}

func TestCatalog_BothInstalled(t *testing.T) {
	st := NewCatalog("", lookPathFor("claude", "codex")).Status()
	if !st.Available || st.DefaultProvider != "claude" || st.Reason != "" {
		t.Fatalf("status = %+v", st)
	}
	if len(st.Providers) != 2 || st.Providers[0].ID != "claude" || st.Providers[1].ID != "codex" {
		t.Fatalf("providers = %+v", st.Providers)
	}
	claude := provider(st, "claude")
	if !claude.Available || claude.Name != "Claude Code" || len(claude.Models) != 4 || claude.Models[0].ID != "" || claude.Models[3].ID != "opus" {
		t.Errorf("claude = %+v", claude)
	}
	if got := strings.Join(claude.Models[1].Efforts, ","); got != ",low,medium,high,xhigh,max" {
		t.Errorf("claude efforts = %q", got)
	}
	codex := provider(st, "codex")
	if !codex.Available || codex.Name != "Codex" || len(codex.Models) != 1 || codex.Models[0].ID != "" {
		t.Errorf("codex before the model list = %+v", codex)
	}
}

func TestCatalog_ConfiguredDefault(t *testing.T) {
	st := NewCatalog("codex", lookPathFor("claude", "codex")).Status()
	if st.DefaultProvider != "codex" {
		t.Fatalf("default = %q, want codex", st.DefaultProvider)
	}
}

func TestCatalog_DefaultFallsBackToInstalled(t *testing.T) {
	for _, agent := range []string{"", "claude", "gemini"} {
		st := NewCatalog(agent, lookPathFor("codex")).Status()
		if !st.Available || st.DefaultProvider != "codex" {
			t.Errorf("chat.agent=%q: status = %+v, want codex as the default", agent, st)
		}
		claude := provider(st, "claude")
		if claude.Available || !strings.Contains(claude.Reason, "claude") {
			t.Errorf("claude = %+v", claude)
		}
	}
}

func TestCatalog_NoneInstalled(t *testing.T) {
	st := NewCatalog("", lookPathFor()).Status()
	if st.Available || st.DefaultProvider != "" {
		t.Fatalf("status = %+v", st)
	}
	if !strings.Contains(st.Reason, "Claude Code") || !strings.Contains(st.Reason, "Codex") {
		t.Errorf("reason = %q, want install hints for both", st.Reason)
	}
}

func TestCatalog_SetModels(t *testing.T) {
	c := NewCatalog("", lookPathFor("claude", "codex"))
	c.SetModels("codex", []ModelOption{
		{ID: "", Label: "Default model", Efforts: []string{"", "low"}},
		{ID: "gpt-x", Label: "GPT-X", Efforts: []string{"", "high"}},
	})
	codex := provider(c.Status(), "codex")
	if len(codex.Models) != 2 || codex.Models[1].ID != "gpt-x" {
		t.Fatalf("codex = %+v", codex)
	}
	// Status returns a copy.
	c.Status().Providers[1].Models[1].ID = "changed"
	if provider(c.Status(), "codex").Models[1].ID != "gpt-x" {
		t.Fatal("Status shares its model slice")
	}
}

func TestCatalog_Resolve(t *testing.T) {
	c := NewCatalog("", lookPathFor("claude", "codex"))
	c.SetModels("codex", []ModelOption{
		{ID: "", Label: "Default model", Efforts: []string{"", "low"}},
		{ID: "gpt-x", Label: "GPT-X", Efforts: []string{"", "high"}},
	})
	tests := []struct {
		provider, model, effort string
		want                    string // resolved provider, or the error substring
		ok                      bool
	}{
		{"", "sonnet", "max", "claude", true},
		{"claude", "", "", "claude", true},
		{"codex", "gpt-x", "high", "codex", true},
		{"codex", "", "low", "codex", true},
		{"codex", "gpt-x", "low", `effort "low"`, false},
		{"codex", "sonnet", "", `model "sonnet"`, false},
		{"claude", "gpt-x", "", `model "gpt-x"`, false},
		{"gemini", "", "", `provider "gemini"`, false},
		{"claude", "opus; rm -rf", "", `model "opus; rm -rf"`, false},
		{"claude", "--dangerously-skip-permissions", "", `model "--dangerously-skip-permissions"`, false},
		{"claude", "", "ultra", `effort "ultra"`, false},
	}
	for _, tt := range tests {
		got, err := c.Resolve(tt.provider, tt.model, tt.effort)
		if tt.ok && (err != nil || got != tt.want) {
			t.Errorf("Resolve(%q,%q,%q) = %q, %v; want %q", tt.provider, tt.model, tt.effort, got, err, tt.want)
		}
		if !tt.ok && (err == nil || !strings.Contains(err.Error(), tt.want)) {
			t.Errorf("Resolve(%q,%q,%q) err = %v; want it to mention %s", tt.provider, tt.model, tt.effort, err, tt.want)
		}
	}
}

func TestCatalog_ResolveUnavailable(t *testing.T) {
	c := NewCatalog("", lookPathFor("claude"))
	if _, err := c.Resolve("codex", "", ""); err == nil || !strings.Contains(err.Error(), "codex") {
		t.Fatalf("err = %v, want the codex install reason", err)
	}
}
