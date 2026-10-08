package identity

import "testing"

func TestIsAgent(t *testing.T) {
	cases := map[string]bool{
		"Claude Code <agent@macbook.local>":     true,
		"claude-code/2.1.263 <agent@mcp>":       true,
		"claude <agent@NICSPC>":                 true,
		"Codex <AGENT@MBPM1X.local>":            true,
		"Nicolas Bettenburg <nicbet@gmail.com>": false,
		"Agent Smith <agent.smith@example.com>": false,
		"beats migrate <system>":                false,
		"":                                      false,
		"someone":                               false,
	}
	for in, want := range cases {
		if got := IsAgent(in); got != want {
			t.Errorf("IsAgent(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestAgentName(t *testing.T) {
	cases := map[string]string{
		"claude-code/2.1.263 <agent@mcp>":      "Claude Code",
		"codex-mcp-client/0.154.0 <agent@mcp>": "Codex",
		"Claude Code <agent@macbook.local>":    "Claude Code",
		"claude <agent@NICSPC>":                "Claude Code",
		"windsurf/1.2 <agent@mcp>":             "windsurf",
		"mcp-client <agent@mcp>":               "mcp-client",
		"":                                     "",
	}
	for in, want := range cases {
		if got := AgentName(in); got != want {
			t.Errorf("AgentName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveAssignment(t *testing.T) {
	const (
		nic   = "Nicolas Bettenburg <nicbet@gmail.com>"
		alice = "Alice <alice@example.com>"
		cc    = "Claude Code <agent@macbook.local>"
		mcp   = "claude-code/2.1.263 <agent@mcp>"
	)
	cases := []struct {
		name                   string
		raw, createdBy, behalf string
		wantAssignee, wantVia  string
	}{
		{"legacy agent self-assign with principal", cc, mcp, nic, nic, cc},
		{"human assigns an agent in the UI", cc, nic, "", nic, cc},
		{"agent with no human anywhere", cc, "claude <agent@NICSPC>", "", cc, cc},
		{"agent auto-assigns its principal", nic, mcp, nic, nic, mcp},
		{"agent assigns a different human", alice, mcp, nic, alice, ""},
		{"human assigns themselves", nic, nic, "", nic, ""},
		{"human assigns someone else", alice, nic, "", alice, ""},
		{"unassign", "", mcp, nic, "", ""},
		{"principal match ignores email case", "Nicolas Bettenburg <NICBET@gmail.com>", mcp, nic, "Nicolas Bettenburg <NICBET@gmail.com>", mcp},
		{"non-agent@ override identity acting for principal", nic, "Bot <bot@example.com>", nic, nic, "Bot <bot@example.com>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotA, gotV := ResolveAssignment(c.raw, c.createdBy, c.behalf)
			if gotA != c.wantAssignee || gotV != c.wantVia {
				t.Errorf("ResolveAssignment(%q, %q, %q) = (%q, %q), want (%q, %q)",
					c.raw, c.createdBy, c.behalf, gotA, gotV, c.wantAssignee, c.wantVia)
			}
		})
	}
}

func TestSame(t *testing.T) {
	if !Same("A <a@x.com>", "Other Name <A@X.com>") {
		t.Error("same email should match")
	}
	if Same("A <a@x.com>", "A <b@x.com>") {
		t.Error("different email should not match")
	}
	if !Same("alice", "Alice") {
		t.Error("bare names compare case-insensitively")
	}
	if Same("", "") {
		t.Error("empty never matches")
	}
}

func TestName(t *testing.T) {
	if got := Name("Nicolas Bettenburg <nicbet@gmail.com>"); got != "Nicolas Bettenburg" {
		t.Errorf("Name = %q", got)
	}
	if got := Name("alice"); got != "alice" {
		t.Errorf("Name = %q", got)
	}
}

func TestActor(t *testing.T) {
	if got := Actor("claude-code/2.1.263 <agent@mcp>", "Nicolas <nic@example.com>"); got != "Nicolas (via Claude Code)" {
		t.Errorf("Actor = %q", got)
	}
	if got := Actor("Alice <alice@example.com>", ""); got != "Alice" {
		t.Errorf("Actor = %q", got)
	}
}
