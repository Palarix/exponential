package exponential

import (
	"os"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/version"
)

// These tests verify the behavior the spec requires, targeting bugs
// found during manual testing. Each test documents which bug it prevents.

// Bug: unconfigured agents (CLI on PATH but never set up) were included
// in the doctor --fix plan, overwriting files the user intentionally skipped.
func TestHealth_UnconfiguredAgent_NotFixable(t *testing.T) {
	setupProject(t, "test")
	// Agent binary might be on PATH but nothing installed — simulate by not calling setup

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if h.Configured {
		t.Fatal("agent with nothing installed should not be considered configured")
	}
	if len(h.AutoFix) > 0 {
		t.Fatalf("unconfigured agent should have no fixable problems, got: %v", h.AutoFix)
	}
	if len(h.Interactive) > 0 {
		t.Fatalf("unconfigured agent should have no manual items, got: %v", h.Interactive)
	}
}

// Bug: fully configured agents were included in --fix plan even though
// nothing was wrong, causing unnecessary rewrites.
func TestHealth_FullyConfiguredAgent_NoProblem(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if !h.Configured {
		t.Fatal("fully set up agent should be configured")
	}
	if len(h.AutoFix) > 0 {
		t.Fatalf("fully configured agent should have no problems, got: %v", h.AutoFix)
	}
	if len(h.Interactive) > 0 {
		t.Fatalf("fully configured agent should have no manual items, got: %v", h.Interactive)
	}
}

// Bug: editing a managed block was treated as a fixable problem (Problems),
// causing --fix to silently overwrite user edits. It should be Manual.
func TestHealth_EditedBlock_IsManualNotFixable(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	// Edit the managed block
	content, _ := os.ReadFile("CLAUDE.md")
	modified := strings.Replace(string(content), "Every code change must", "CUSTOM RULE", 1)
	os.WriteFile("CLAUDE.md", []byte(modified), 0644)

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if len(h.AutoFix) > 0 {
		t.Fatalf("edited block should NOT be in Problems (fixable), got: %v", h.AutoFix)
	}
	if len(h.Interactive) == 0 {
		t.Fatal("edited block should be in Manual (needs user attention)")
	}
	if !strings.Contains(h.Interactive[0], "local edits") {
		t.Fatalf("manual message should mention local edits, got: %s", h.Interactive[0])
	}
}

// Bug: legacy format was treated as fixable, but overwriting a legacy section
// needs the interactive replace/keep/diff prompt from xpo init.
func TestHealth_LegacyFormat_IsManualNotFixable(t *testing.T) {
	setupProject(t, "test")

	// Write legacy heading without managed block markers
	os.WriteFile("CLAUDE.md", []byte("# Agent Instructions\n\nCustom legacy content.\n"), 0644)
	// Also install MCP so the agent counts as configured
	EnsureMCPConfigFor(testClaudeCode.MCPConfig)

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if !h.Configured {
		t.Fatal("agent with MCP + legacy instructions should be configured")
	}
	if len(h.Interactive) == 0 {
		t.Fatal("legacy format should be in Manual")
	}
	if !strings.Contains(h.Interactive[0], "legacy format") {
		t.Fatalf("manual message should mention legacy format, got: %s", h.Interactive[0])
	}
	// Legacy format itself should NOT be in Problems
	for _, p := range h.AutoFix {
		if strings.Contains(p, "legacy") || strings.Contains(p, "Legacy") {
			t.Fatalf("legacy format should not be in Problems, found: %s", p)
		}
	}
}

// Bug: deleting .claude/ (skills directory) was not detected because the
// global skill check at ~/.claude/skills/ returned true.
func TestHealth_DeletedSkill_ReportsAsFixable(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	// Delete skill directory
	os.RemoveAll(".claude")

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if !h.Configured {
		t.Fatal("agent with MCP + instructions should still be configured after skill deletion")
	}
	found := false
	for _, p := range h.AutoFix {
		if strings.Contains(p, "Skill not installed") {
			found = true
		}
	}
	if !found {
		t.Fatalf("deleted skill should appear in Problems, got: %v", h.AutoFix)
	}
}

// Bug: deleting .mcp.json was detected but the fix plan included all files
// for the agent, not just the MCP config.
func TestHealth_MissingMCP_ReportsAsFixable(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	os.Remove(".mcp.json")

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if !h.Configured {
		t.Fatal("agent with instructions + skill should still be configured")
	}
	found := false
	for _, p := range h.AutoFix {
		if strings.Contains(p, "MCP missing") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing MCP should appear in Problems, got: %v", h.AutoFix)
	}
}

// Verify that an agent with ONLY MCP configured (no instructions, no skill)
// counts as configured and reports the missing pieces.
func TestHealth_OnlyMCP_ReportsMissingPieces(t *testing.T) {
	setupProject(t, "test")
	EnsureMCPConfigFor(testClaudeCode.MCPConfig)

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if !h.Configured {
		t.Fatal("agent with only MCP should be configured")
	}
	if !h.HasMCP {
		t.Fatal("HasMCP should be true")
	}

	var hasInstrProblem, hasSkillProblem bool
	for _, p := range h.AutoFix {
		if strings.Contains(p, "Instructions missing") {
			hasInstrProblem = true
		}
		if strings.Contains(p, "Skill not installed") {
			hasSkillProblem = true
		}
	}
	if !hasInstrProblem {
		t.Fatal("should report Instructions missing")
	}
	if !hasSkillProblem {
		t.Fatal("should report Skill not installed")
	}
}

// Verify stale integrations are reported as fixable.
func TestHealth_StaleVersion_ReportsUpdate(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	h := CheckAgentHealth(testClaudeCode, "0.1.0")

	found := false
	for _, p := range h.AutoFix {
		if strings.Contains(p, "Update available") && strings.Contains(p, "0.1.0") {
			found = true
		}
	}
	if !found {
		t.Fatalf("stale version should report Update available, got: %v", h.AutoFix)
	}
}

// Verify up-to-date integrations report nothing.
func TestHealth_CurrentVersion_NoProblem(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if len(h.AutoFix) > 0 {
		t.Fatalf("current version should have no problems, got: %v", h.AutoFix)
	}
}

// Bug: the fix plan for one broken agent included changes for ALL agents.
// This test verifies the change plan only contains files for the given agents.
func TestHealth_FixPlan_OnlyBrokenAgent(t *testing.T) {
	setupProject(t, "test")
	// Set up Claude Code fully
	setupAgentFull(t, testClaudeCode, "test")
	// Set up Codex partially (MCP only)
	EnsureMCPConfigFor(testCodex.MCPConfig)

	claudeHealth := CheckAgentHealth(testClaudeCode, version.CLIVersion)
	codexHealth := CheckAgentHealth(testCodex, version.CLIVersion)

	if len(claudeHealth.AutoFix) > 0 {
		t.Fatalf("Claude Code should have no problems, got: %v", claudeHealth.AutoFix)
	}
	if len(codexHealth.AutoFix) == 0 {
		t.Fatal("Codex should have problems (missing instructions, skill)")
	}

	// Only Codex should be in the fix plan
	var fixAgents []AgentConfig
	if len(codexHealth.AutoFix) > 0 {
		fixAgents = append(fixAgents, testCodex)
	}
	// Claude Code should NOT be added
	if len(claudeHealth.AutoFix) > 0 {
		fixAgents = append(fixAgents, testClaudeCode)
	}

	plan := ComputeInitPlan(fixAgents)

	for _, c := range plan.Changes {
		if strings.Contains(c.Path, ".claude/") {
			t.Fatalf("fix plan for Codex should not include Claude Code files, found: %s", c.Path)
		}
		if c.Path == ".mcp.json" {
			t.Fatalf("fix plan for Codex should not include .mcp.json (Claude Code's MCP), found: %s", c.Path)
		}
	}
}
