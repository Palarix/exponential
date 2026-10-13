package chat

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"
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
	if !claude.Available || claude.Name != "Claude Code" || len(claude.Models) != 5 || claude.Models[0].ID != "" || claude.Models[3].ID != "opus" {
		t.Errorf("claude = %+v", claude)
	}
	if got := strings.Join(claude.Models[1].Efforts, ","); got != "low,medium,high,xhigh,max" {
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
		{ID: "", Label: "Default model", Efforts: []string{"low", "medium"}},
		{ID: "gpt-x", Label: "GPT-X", Efforts: []string{"medium", "high"}},
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
		{ID: "", Label: "Default model", Efforts: []string{"low", "medium"}},
		{ID: "gpt-x", Label: "GPT-X", Efforts: []string{"medium", "high"}},
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
		{"codex", "gpt-x", "", "codex", true}, // empty effort: the CLI's default
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

func TestCatalog_FableModel(t *testing.T) {
	claude := provider(NewCatalog("", lookPathFor("claude")).Status(), "claude")
	ids := make([]string, len(claude.Models))
	for i, m := range claude.Models {
		ids[i] = m.ID
	}
	if strings.Join(ids, ",") != ",haiku,sonnet,opus,fable" {
		t.Fatalf("claude models = %v", ids)
	}
}

func TestCatalog_Learn(t *testing.T) {
	c := NewCatalog("", lookPathFor("claude", "codex"))
	c.Learn("claude", "", "claude-opus-5-5[1m]", 1000000)
	c.Learn("claude", "haiku", "claude-haiku-5-5", 200000)
	c.Learn("claude", "gone", "claude-x-1", 1) // not in the list: ignored
	c.Learn("gemini", "", "gemini-3", 1)       // unknown provider: ignored
	c.Learn("claude", "sonnet", "claude-sonnet-5-5", 0)

	claude := provider(c.Status(), "claude")
	got := map[string]string{}
	for _, m := range claude.Models {
		got[m.ID] = fmt.Sprintf("%s|%d", m.Name, m.ContextWindow)
	}
	want := map[string]string{
		"":       "Claude Opus 5.5|1000000",
		"haiku":  "Claude Haiku 5.5|200000",
		"sonnet": "Claude Sonnet 5.5|0",
		"opus":   "|0",
		"fable":  "|0",
	}
	if !maps.Equal(got, want) {
		t.Fatalf("models = %v, want %v", got, want)
	}
}

func TestCatalog_LearnCodexUsesDisplayName(t *testing.T) {
	c := NewCatalog("", lookPathFor("codex"))
	// Learned before the model list arrives: the name resolves once it does.
	c.Learn("codex", "", "gpt-x", 258400)
	if m := provider(c.Status(), "codex").Models[0]; m.Name != "gpt-x" || m.ContextWindow != 258400 {
		t.Fatalf("before the list: %+v", m)
	}
	c.SetModels("codex", []ModelOption{
		{ID: "", Label: "Default model", Name: "GPT-Y", Efforts: []string{"medium"}},
		{ID: "gpt-x", Label: "GPT-X", Name: "GPT-X", Efforts: []string{"medium"}},
	})
	if m := provider(c.Status(), "codex").Models[0]; m.Name != "GPT-X" || m.ContextWindow != 258400 {
		t.Fatalf("after the list: %+v, want the learned model's display name", m)
	}
}

func TestCatalog_LearnFromMetas(t *testing.T) {
	c := NewCatalog("", lookPathFor("claude", "codex"))
	now := time.Now()
	c.LearnFrom([]Meta{
		{Agent: "claude", ResolvedModel: "claude-opus-5-5", ContextWindow: 1000000, UpdatedAt: now},
		{Agent: "claude", ResolvedModel: "claude-opus-4-1", ContextWindow: 200000, UpdatedAt: now.Add(-time.Hour)},
		{Agent: "claude", Model: "haiku", UpdatedAt: now}, // no turn finished yet
		{Agent: "", ResolvedModel: "claude-sonnet-5-5"},   // legacy chat without an agent
	})
	claude := provider(c.Status(), "claude")
	if m := claude.Models[0]; m.Name != "Claude Opus 5.5" || m.ContextWindow != 1000000 {
		t.Errorf("default = %+v, want the newest chat's model", m)
	}
	if m := claude.Models[1]; m.Name != "" {
		t.Errorf("haiku = %+v, want nothing learned", m)
	}
}

func TestModelName(t *testing.T) {
	for _, tt := range []struct{ id, want string }{
		{"claude-opus-5-5", "Claude Opus 5.5"},
		{"claude-opus-5-5[1m]", "Claude Opus 5.5"},
		{"claude-fable-5-1", "Claude Fable 5.1"},
		{"claude-haiku-4-5-20251001", "Claude Haiku 4.5"},
		{"claude-sonnet-4", "Claude Sonnet 4"},
		{"claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20241022"},
		{"opus", "opus"},
		{"", ""},
	} {
		if got := claudeModelName(tt.id); got != tt.want {
			t.Errorf("claudeModelName(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}
