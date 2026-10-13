package chat

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// ModelOption is one model the composer offers for a provider, with the
// effort levels it supports. An empty ID or effort means "the default".
type ModelOption struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Efforts []string `json:"efforts"`
}

// ProviderStatus describes one agent CLI chat can run, and why not when it
// cannot.
type ProviderStatus struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Available bool          `json:"available"`
	Reason    string        `json:"reason,omitempty"`
	Models    []ModelOption `json:"models"`
}

// AgentStatus tells the UI which agents chat can run. Available is true when
// at least one provider is; Reason then explains why none is.
type AgentStatus struct {
	Available       bool             `json:"available"`
	DefaultProvider string           `json:"default_provider"`
	Reason          string           `json:"reason,omitempty"`
	Providers       []ProviderStatus `json:"providers"`
}

// Providers chat knows, in the order the composer lists them.
const (
	ProviderClaude = "claude"
	ProviderCodex  = "codex"
)

var (
	allEfforts   = []string{"", "low", "medium", "high", "xhigh", "max"}
	claudeModels = []ModelOption{
		{ID: "", Label: "Default model", Efforts: allEfforts},
		{ID: "haiku", Label: "Haiku", Efforts: allEfforts},
		{ID: "sonnet", Label: "Sonnet", Efforts: allEfforts},
		{ID: "opus", Label: "Opus", Efforts: allEfforts},
	}
	// codexFallbackModels is what Codex offers until, or unless, its
	// model/list call answers.
	codexFallbackModels = []ModelOption{
		{ID: "", Label: "Default model", Efforts: []string{"", "low", "medium", "high"}},
	}
)

type providerInfo struct {
	id, name, binary, install string
	models                    []ModelOption
}

var knownProviders = []providerInfo{
	{ProviderClaude, "Claude Code", "claude", "Install Claude Code from https://code.claude.com", claudeModels},
	{ProviderCodex, "Codex", "codex", "Install Codex from https://github.com/openai/codex", codexFallbackModels},
}

// Catalog holds the providers chat can run. The board builds it at startup
// and fills in model lists that arrive later (SetModels).
type Catalog struct {
	mu     sync.RWMutex
	status AgentStatus
}

// NewCatalog checks which agent CLIs are on PATH. agent is chat.agent from
// the config: the default provider when it is installed. Otherwise the
// first installed provider is the default.
func NewCatalog(agent string, lookPath func(string) (string, error)) *Catalog {
	var st AgentStatus
	var hints []string
	for _, p := range knownProviders {
		ps := ProviderStatus{ID: p.id, Name: p.name, Models: p.models}
		if _, err := lookPath(p.binary); err != nil {
			ps.Reason = fmt.Sprintf("%s (%s) is not on PATH. %s and restart xpo board.", p.name, p.binary, p.install)
			hints = append(hints, p.install)
		} else {
			ps.Available = true
			if st.DefaultProvider == "" || p.id == agent {
				st.DefaultProvider = p.id
			}
		}
		st.Providers = append(st.Providers, ps)
	}
	st.Available = st.DefaultProvider != ""
	if !st.Available {
		st.Reason = "No supported agent CLI is on PATH. " + strings.Join(hints, ", or ") + ", then restart xpo board."
	}
	return &Catalog{status: st}
}

// Status returns a copy of the providers.
func (c *Catalog) Status() AgentStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	st := c.status
	st.Providers = slices.Clone(st.Providers)
	for i := range st.Providers {
		st.Providers[i].Models = slices.Clone(st.Providers[i].Models)
	}
	return st
}

// SetModels replaces a provider's model list.
func (c *Catalog) SetModels(provider string, models []ModelOption) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.status.Providers {
		if c.status.Providers[i].ID == provider {
			c.status.Providers[i].Models = models
		}
	}
}

// Resolve checks a message's provider, model and effort. An empty provider
// means the default one. It returns the provider to run.
func (c *Catalog) Resolve(provider, model, effort string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if provider == "" {
		provider = c.status.DefaultProvider
	}
	i := slices.IndexFunc(c.status.Providers, func(p ProviderStatus) bool { return p.ID == provider })
	if i < 0 {
		return "", fmt.Errorf("unknown provider %q", provider)
	}
	p := c.status.Providers[i]
	if !p.Available {
		return "", fmt.Errorf("%s", p.Reason)
	}
	j := slices.IndexFunc(p.Models, func(m ModelOption) bool { return m.ID == model })
	if j < 0 {
		return "", fmt.Errorf("unknown model %q for %s", model, p.Name)
	}
	if !slices.Contains(p.Models[j].Efforts, effort) {
		return "", fmt.Errorf("unknown effort %q for %s", effort, p.Name)
	}
	return provider, nil
}
