package exponential

import (
	"errors"
	"testing"

	"github.com/palarix/exponential/internal/config"
)

// Compile-time interface compliance checks.
var _ Transport = (*LocalTransport)(nil)
var _ Transport = (*RemoteTransport)(nil)

func TestNewClient_LocalByDefault(t *testing.T) {
	cfg := &config.Config{Prefix: "test-"}
	c := NewClient(cfg)

	if c.local == nil {
		t.Fatal("expected local transport for empty Remote config")
	}
	if _, ok := c.Transport.(*LocalTransport); !ok {
		t.Fatal("expected Transport to be *LocalTransport")
	}
}

func TestNewClient_RemoteWhenURLSet(t *testing.T) {
	cfg := &config.Config{
		Prefix: "test-",
		Remote: config.RemoteConfig{
			URL:   "https://xpo.example.com",
			Token: "test-token",
		},
	}
	c := NewClient(cfg)

	if c.local != nil {
		t.Fatal("expected local to be nil for remote config")
	}
	rt, ok := c.Transport.(*RemoteTransport)
	if !ok {
		t.Fatal("expected Transport to be *RemoteTransport")
	}
	if rt.BaseURL != "https://xpo.example.com" {
		t.Errorf("expected BaseURL to be set, got %q", rt.BaseURL)
	}
	if rt.Token != "test-token" {
		t.Errorf("expected Token to be set, got %q", rt.Token)
	}
}

func TestSyncLocal_PropagatesFields(t *testing.T) {
	cfg := &config.Config{Prefix: "test-"}
	c := NewClient(cfg)
	c.Collapse = true
	c.UserOverride = "Agent <agent@test.com>"

	c.syncLocal()

	if !c.local.Collapse {
		t.Error("expected Collapse to propagate to LocalTransport")
	}
	if c.local.UserOverride != "Agent <agent@test.com>" {
		t.Errorf("expected UserOverride to propagate, got %q", c.local.UserOverride)
	}
}

func TestErrLocalOnly_MergeIssue(t *testing.T) {
	cfg := &config.Config{
		Prefix: "test-",
		Remote: config.RemoteConfig{URL: "https://xpo.example.com"},
	}
	c := NewClient(cfg)

	_, err := c.MergeIssue("test-123", MergeOptions{})
	if !errors.Is(err, ErrLocalOnly) {
		t.Errorf("expected ErrLocalOnly, got %v", err)
	}
}

func TestErrLocalOnly_GetArchiveStats(t *testing.T) {
	cfg := &config.Config{
		Prefix: "test-",
		Remote: config.RemoteConfig{URL: "https://xpo.example.com"},
	}
	c := NewClient(cfg)

	_, err := c.GetArchiveStats(30, 3)
	if !errors.Is(err, ErrLocalOnly) {
		t.Errorf("expected ErrLocalOnly, got %v", err)
	}
}

func TestErrLocalOnly_PerformArchive(t *testing.T) {
	cfg := &config.Config{
		Prefix: "test-",
		Remote: config.RemoteConfig{URL: "https://xpo.example.com"},
	}
	c := NewClient(cfg)

	err := c.PerformArchive(&ArchiveStats{})
	if !errors.Is(err, ErrLocalOnly) {
		t.Errorf("expected ErrLocalOnly, got %v", err)
	}
}

func TestErrLocalOnly_IssueIDFromBranch(t *testing.T) {
	cfg := &config.Config{
		Prefix: "test-",
		Remote: config.RemoteConfig{URL: "https://xpo.example.com"},
	}
	c := NewClient(cfg)

	_, err := c.IssueIDFromBranch("test-123/feature")
	if !errors.Is(err, ErrLocalOnly) {
		t.Errorf("expected ErrLocalOnly, got %v", err)
	}
}
