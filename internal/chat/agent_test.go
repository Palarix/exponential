package chat

import (
	"errors"
	"strings"
	"testing"
)

func found(string) (string, error)   { return "/usr/local/bin/claude", nil }
func missing(string) (string, error) { return "", errors.New("not found") }

func TestCheckAgent_Defaults(t *testing.T) {
	got := CheckAgent("", "", found)
	if !got.Available || got.Agent != "claude" || got.Model != "" || got.Reason != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestCheckAgent_Model(t *testing.T) {
	if got := CheckAgent("claude", "sonnet", found); got.Model != "sonnet" {
		t.Fatalf("got %+v", got)
	}
}

func TestCheckAgent_NotOnPath(t *testing.T) {
	got := CheckAgent("", "", missing)
	if got.Available || !strings.Contains(got.Reason, "claude") {
		t.Fatalf("got %+v", got)
	}
}

func TestCheckAgent_UnsupportedAgent(t *testing.T) {
	got := CheckAgent("codex", "", found)
	if got.Available || !strings.Contains(got.Reason, "codex") {
		t.Fatalf("got %+v", got)
	}
}
