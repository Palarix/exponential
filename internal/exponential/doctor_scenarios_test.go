package exponential

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/version"
)

// --- Agent health check building blocks ---

func setupProject(t *testing.T, prefix string) string {
	t.Helper()
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(orig) })

	InitProject(false, prefix)
	return dir
}

func setupAgentFull(t *testing.T, agent AgentConfig, prefix string) {
	t.Helper()
	AppendAgentInstructions(agent, prefix)
	if agent.MCPConfig.HasMCPConfig() {
		EnsureMCPConfigFor(agent.MCPConfig)
	}
	if agent.SkillDir != "" {
		WriteAgentSkill(agent, "")
	}
}

var testClaudeCode = AgentConfig{
	Name:     "Claude Code",
	File:     "CLAUDE.md",
	Binary:   "claude",
	SkillDir: ".claude/skills",
	Format:   "markdown",
	MCPConfig: MCPConfigSpec{
		File:      ".mcp.json",
		ServerKey: "mcpServers",
		Format:    "json",
	},
}

var testCodex = AgentConfig{
	Name:     "Codex",
	File:     "AGENTS.md",
	Binary:   "codex",
	SkillDir: ".codex/skills",
	Format:   "markdown",
	MCPConfig: MCPConfigSpec{
		File:      ".codex/config.toml",
		ServerKey: "mcp_servers",
		Format:    "toml",
		NeedsDir:  true,
	},
}

var testOpenCode = AgentConfig{
	Name:     "OpenCode",
	File:     "AGENTS.md",
	Binary:   "opencode",
	SkillDir: ".opencode/skills",
	Format:   "markdown",
	MCPConfig: MCPConfigSpec{
		File:      "opencode.json",
		ServerKey: "mcp",
		Format:    "json",
		EntryStyle: "local-array",
	},
}

// --- Managed block detection ---

func TestDoctor_ManagedBlock_Unedited(t *testing.T) {
	setupProject(t, "test")
	AppendAgentInstructions(testClaudeCode, "test")

	content, _ := os.ReadFile("CLAUDE.md")
	block := FindManagedBlock(string(content), "markdown")
	if block == nil {
		t.Fatal("expected managed block")
	}
	if BlockWasEdited(block) {
		t.Fatal("freshly written block should not be detected as edited")
	}
}

func TestDoctor_ManagedBlock_Edited(t *testing.T) {
	setupProject(t, "test")
	AppendAgentInstructions(testClaudeCode, "test")

	content, _ := os.ReadFile("CLAUDE.md")
	modified := strings.Replace(string(content), "## Exponential (xpo)", "# My Custom Instructions", 1)
	os.WriteFile("CLAUDE.md", []byte(modified), 0644)

	modContent, _ := os.ReadFile("CLAUDE.md")
	block := FindManagedBlock(string(modContent), "markdown")
	if block == nil {
		t.Fatal("expected managed block after edit")
	}
	if !BlockWasEdited(block) {
		t.Fatal("edited block should be detected as edited")
	}
}

func TestDoctor_ManagedBlock_LegacyHeading(t *testing.T) {
	setupProject(t, "test")

	// Write legacy format (heading without managed block markers)
	os.WriteFile("CLAUDE.md", []byte("# Agent Instructions\n\nOld content.\n"), 0644)

	content, _ := os.ReadFile("CLAUDE.md")

	// No managed block
	block := FindManagedBlock(string(content), "markdown")
	if block != nil {
		t.Fatal("legacy heading should not be detected as managed block")
	}

	// But legacy heading IS detected
	if !HasAgentInstructions(string(content)) {
		t.Fatal("legacy heading should be detected by HasAgentInstructions")
	}
}

func TestDoctor_ManagedBlock_Migration(t *testing.T) {
	setupProject(t, "test")

	// Start with legacy format
	os.WriteFile("CLAUDE.md", []byte("# My Project\n\nDocs.\n\n# Agent Instructions\n\nOld content.\n"), 0644)

	// AppendAgentInstructions should migrate to managed block
	AppendAgentInstructions(testClaudeCode, "test")

	content, _ := os.ReadFile("CLAUDE.md")
	s := string(content)

	if !strings.Contains(s, "<!-- xpo:begin") {
		t.Fatal("migration should add managed block markers")
	}
	if !strings.Contains(s, "# My Project") {
		t.Fatal("user content should be preserved during migration")
	}
	if strings.Contains(s, "Old content") {
		t.Fatal("old content should be replaced during migration")
	}
}

// --- MCP detection ---

func TestDoctor_MCP_Configured(t *testing.T) {
	setupProject(t, "test")
	EnsureMCPConfigFor(testClaudeCode.MCPConfig)

	status := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if !status.HasExponential {
		t.Fatal("MCP should be detected as configured")
	}
}

func TestDoctor_MCP_Missing(t *testing.T) {
	setupProject(t, "test")
	// Don't install MCP

	status := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if status.HasExponential {
		t.Fatal("MCP should not be detected when not installed")
	}
}

func TestDoctor_MCP_FileExistsButNoXpo(t *testing.T) {
	setupProject(t, "test")
	os.WriteFile(".mcp.json", []byte(`{"mcpServers":{"other":{"command":"other"}}}`), 0644)

	status := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if !status.Exists {
		t.Fatal("file should be detected as existing")
	}
	if status.HasExponential {
		t.Fatal("xpo entry should not be detected")
	}
}

// --- Skill detection ---

func TestDoctor_Skill_InstalledLocally(t *testing.T) {
	setupProject(t, "test")
	WriteAgentSkill(testClaudeCode, "")

	status := DetectSkillInstall(testClaudeCode)
	if !status.Local {
		t.Fatal("skill should be detected as locally installed")
	}
}

func TestDoctor_Skill_NotInstalled(t *testing.T) {
	setupProject(t, "test")
	// Don't install skills

	status := DetectSkillInstall(testClaudeCode)
	if status.Local {
		t.Fatal("skill should not be detected when not installed")
	}
}

func TestDoctor_Skill_Deleted(t *testing.T) {
	setupProject(t, "test")
	WriteAgentSkill(testClaudeCode, "")

	// Verify it's detected
	status := DetectSkillInstall(testClaudeCode)
	if !status.Local {
		t.Fatal("setup: skill should be installed")
	}

	// Delete the skill directory
	os.RemoveAll(".claude")

	status = DetectSkillInstall(testClaudeCode)
	if status.Local {
		t.Fatal("skill should not be detected after deletion")
	}
}

// --- Configured vs unconfigured agent distinction ---

func TestDoctor_Agent_FullyConfigured(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	// All three pieces present
	mcpStatus := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if !mcpStatus.HasExponential {
		t.Fatal("MCP should be configured")
	}

	content, _ := os.ReadFile("CLAUDE.md")
	block := FindManagedBlock(string(content), "markdown")
	if block == nil {
		t.Fatal("instructions should be present")
	}

	skillStatus := DetectSkillInstall(testClaudeCode)
	if !skillStatus.Local {
		t.Fatal("skill should be installed")
	}
}

func TestDoctor_Agent_PartiallyConfigured_MissingMCP(t *testing.T) {
	setupProject(t, "test")
	// Install instructions and skill but NOT MCP
	AppendAgentInstructions(testClaudeCode, "test")
	WriteAgentSkill(testClaudeCode, "")

	// Instructions present
	content, _ := os.ReadFile("CLAUDE.md")
	if FindManagedBlock(string(content), "markdown") == nil {
		t.Fatal("instructions should be present")
	}

	// MCP missing
	mcpStatus := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if mcpStatus.HasExponential {
		t.Fatal("MCP should NOT be configured")
	}
}

func TestDoctor_Agent_PartiallyConfigured_MissingSkill(t *testing.T) {
	setupProject(t, "test")
	// Install instructions and MCP but NOT skill
	AppendAgentInstructions(testClaudeCode, "test")
	EnsureMCPConfigFor(testClaudeCode.MCPConfig)

	content, _ := os.ReadFile("CLAUDE.md")
	if FindManagedBlock(string(content), "markdown") == nil {
		t.Fatal("instructions should be present")
	}

	mcpStatus := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if !mcpStatus.HasExponential {
		t.Fatal("MCP should be configured")
	}

	skillStatus := DetectSkillInstall(testClaudeCode)
	if skillStatus.Local {
		t.Fatal("skill should NOT be installed")
	}
}

func TestDoctor_Agent_NotConfigured(t *testing.T) {
	setupProject(t, "test")
	// Don't install anything for the agent

	mcpStatus := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if mcpStatus.HasExponential {
		t.Fatal("MCP should not be configured")
	}

	content, _ := os.ReadFile("CLAUDE.md")
	if content != nil && FindManagedBlock(string(content), "markdown") != nil {
		t.Fatal("instructions should not be present")
	}

	skillStatus := DetectSkillInstall(testClaudeCode)
	if skillStatus.Local {
		t.Fatal("skill should not be installed")
	}
}

// --- WriteAgentInstructions edit detection ---

func TestDoctor_WriteInstructions_SkipsEditedBlock(t *testing.T) {
	setupProject(t, "test")
	AppendAgentInstructions(testClaudeCode, "test")

	// Edit the managed block
	content, _ := os.ReadFile("CLAUDE.md")
	modified := strings.Replace(string(content), "## Exponential (xpo)", "# Custom Agent Instructions", 1)
	os.WriteFile("CLAUDE.md", []byte(modified), 0644)

	// Write without force should skip
	result, err := WriteAgentInstructions(testClaudeCode, "test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "skipped" {
		t.Fatalf("expected skipped, got %s", result.Action)
	}
	if !result.WasEdited {
		t.Fatal("expected WasEdited=true")
	}

	// Content should be unchanged
	afterContent, _ := os.ReadFile("CLAUDE.md")
	if !strings.Contains(string(afterContent), "# Custom Agent Instructions") {
		t.Fatal("edited content should be preserved when skipped")
	}
}

func TestDoctor_WriteInstructions_ForceOverwritesEditedBlock(t *testing.T) {
	setupProject(t, "test")
	AppendAgentInstructions(testClaudeCode, "test")

	// Edit the managed block
	content, _ := os.ReadFile("CLAUDE.md")
	modified := strings.Replace(string(content), "## Exponential (xpo)", "# Custom Agent Instructions", 1)
	os.WriteFile("CLAUDE.md", []byte(modified), 0644)

	// Write with force should overwrite
	result, err := WriteAgentInstructions(testClaudeCode, "test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "updated" {
		t.Fatalf("expected updated, got %s", result.Action)
	}

	// Content should be the standard template
	afterContent, _ := os.ReadFile("CLAUDE.md")
	if strings.Contains(string(afterContent), "# Custom Agent Instructions") {
		t.Fatal("edited content should be replaced on force")
	}
	if !strings.Contains(string(afterContent), "## Exponential (xpo)") {
		t.Fatal("standard content should be present after force")
	}
}

// --- Prefix handling ---

func TestDoctor_Prefix_StoredWithoutDash(t *testing.T) {
	setupProject(t, "myproject")

	configContent, _ := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
	if strings.Contains(string(configContent), "prefix: myproject-") {
		t.Fatal("config should store prefix without trailing dash")
	}
	if !strings.Contains(string(configContent), "prefix: myproject") {
		t.Fatal("config should store bare prefix")
	}
}

func TestDoctor_Prefix_InstructionsUseCorrectFormat(t *testing.T) {
	setupProject(t, "pay")
	AppendAgentInstructions(testClaudeCode, "pay")

	content, _ := os.ReadFile("CLAUDE.md")
	s := string(content)
	if !strings.Contains(s, "`pay`") {
		t.Fatal("instructions should show bare prefix")
	}
	if !strings.Contains(s, "`pay-a1b2c3`") {
		t.Fatal("instructions should show example ID with dash separator")
	}
}

// --- Version stamping ---

func TestDoctor_VersionStamp_InManagedBlock(t *testing.T) {
	setupProject(t, "test")
	AppendAgentInstructions(testClaudeCode, "test")

	// Version should be readable from the managed block, not config
	ver := DetectIntegrationVersion()
	if ver != version.CLIVersion {
		t.Fatalf("expected integration version %s from managed block, got %q", version.CLIVersion, ver)
	}

	// Config should NOT contain integration_version
	configContent, _ := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
	if strings.Contains(string(configContent), "integration_version") {
		t.Fatal("config should not contain integration_version — version lives in managed blocks")
	}
}

// --- Change plan ---

func TestDoctor_ChangePlan_SortsDotfilesFirst(t *testing.T) {
	setupProject(t, "test")

	plan := ComputeInitPlan([]AgentConfig{testClaudeCode})

	if len(plan.Changes) == 0 {
		t.Fatal("plan should have changes")
	}

	// Verify dotfiles come before non-dotfiles
	seenNonDot := false
	for _, c := range plan.Changes {
		isDot := strings.HasPrefix(c.Path, ".")
		if seenNonDot && isDot {
			t.Fatalf("dotfile %s appears after non-dotfile — sort is wrong", c.Path)
		}
		if !isDot {
			seenNonDot = true
		}
	}
}

func TestDoctor_ChangePlan_DetectsCreateVsModify(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// Fresh directory — everything should be "create"
	plan := ComputeInitPlan([]AgentConfig{testClaudeCode})
	for _, c := range plan.Changes {
		if c.Action != "create" {
			t.Fatalf("expected create for %s in empty dir, got %s", c.Path, c.Action)
		}
	}

	// Now create some files and re-plan
	InitProject(false, "test")
	EnsureMCPConfigFor(testClaudeCode.MCPConfig)

	plan2 := ComputeInitPlan([]AgentConfig{testClaudeCode})
	actions := map[string]string{}
	for _, c := range plan2.Changes {
		actions[c.Path] = c.Action
	}

	// .gitignore should now be "modify" since InitProject created it
	if actions[".gitignore"] != "modify" {
		t.Fatalf("expected .gitignore to be modify, got %s", actions[".gitignore"])
	}
	// .mcp.json should now be "modify" since we installed MCP
	if actions[".mcp.json"] != "modify" {
		t.Fatalf("expected .mcp.json to be modify, got %s", actions[".mcp.json"])
	}
}

// --- Multi-agent scenarios ---

func TestDoctor_SharedAGENTSmd_SingleManagedBlock(t *testing.T) {
	setupProject(t, "test")

	// Both agents write to AGENTS.md
	AppendAgentInstructions(testCodex, "test")
	content1, _ := os.ReadFile("AGENTS.md")

	AppendAgentInstructions(testOpenCode, "test")
	content2, _ := os.ReadFile("AGENTS.md")

	// Should be identical — same managed block, same prefix
	if string(content1) != string(content2) {
		t.Fatal("writing second agent to shared AGENTS.md should not change content")
	}

	// Should have exactly one managed block
	s := string(content2)
	count := strings.Count(s, "<!-- xpo:begin")
	if count != 1 {
		t.Fatalf("expected 1 managed block in shared AGENTS.md, got %d", count)
	}
}

func TestDoctor_FullSetup_AllAgents(t *testing.T) {
	setupProject(t, "test")

	agents := []AgentConfig{testClaudeCode, testCodex, testOpenCode}
	for _, agent := range agents {
		setupAgentFull(t, agent, "test")
	}

	// Verify each agent's pieces are in place
	for _, agent := range agents {
		if agent.MCPConfig.HasMCPConfig() {
			status := DetectMCPConfigFor(agent.MCPConfig)
			if !status.HasExponential {
				t.Fatalf("%s: MCP should be configured", agent.Name)
			}
		}

		content, err := os.ReadFile(agent.File)
		if err != nil {
			t.Fatalf("%s: could not read %s: %v", agent.Name, agent.File, err)
		}
		block := FindManagedBlock(string(content), agent.Format)
		if block == nil {
			t.Fatalf("%s: should have managed block in %s", agent.Name, agent.File)
		}
		if BlockWasEdited(block) {
			t.Fatalf("%s: freshly written block should not be detected as edited", agent.Name)
		}

		if agent.SkillDir != "" {
			status := DetectSkillInstall(agent)
			if !status.Local {
				t.Fatalf("%s: skill should be installed locally", agent.Name)
			}
		}
	}
}

func TestDoctor_FullSetup_DeleteSkill_Detected(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	// Verify fully configured
	status := DetectSkillInstall(testClaudeCode)
	if !status.Local {
		t.Fatal("setup: skill should be installed")
	}

	// Delete skill directory
	os.RemoveAll(".claude")

	// Skill should be detected as missing
	status = DetectSkillInstall(testClaudeCode)
	if status.Local {
		t.Fatal("skill should not be detected after deletion")
	}

	// Instructions should still be present
	content, _ := os.ReadFile("CLAUDE.md")
	block := FindManagedBlock(string(content), "markdown")
	if block == nil {
		t.Fatal("instructions should still be present after skill deletion")
	}

	// MCP should still be present
	mcpStatus := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if !mcpStatus.HasExponential {
		t.Fatal("MCP should still be configured after skill deletion")
	}
}

func TestDoctor_FullSetup_EditBlock_Detected(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	// Edit the managed block
	content, _ := os.ReadFile("CLAUDE.md")
	modified := strings.Replace(string(content), "Every code change must", "CUSTOM: every code change must", 1)
	os.WriteFile("CLAUDE.md", []byte(modified), 0644)

	// Block should be detected as edited
	modContent, _ := os.ReadFile("CLAUDE.md")
	block := FindManagedBlock(string(modContent), "markdown")
	if block == nil {
		t.Fatal("managed block should still be found")
	}
	if !BlockWasEdited(block) {
		t.Fatal("edited block should be detected")
	}

	// WriteAgentInstructions should skip without force
	result, _ := WriteAgentInstructions(testClaudeCode, "test", false)
	if result.Action != "skipped" {
		t.Fatalf("expected skipped for edited block, got %s", result.Action)
	}
}

func TestDoctor_FullSetup_DeleteMCP_Detected(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	// Delete MCP config
	os.Remove(".mcp.json")

	mcpStatus := DetectMCPConfigFor(testClaudeCode.MCPConfig)
	if mcpStatus.HasExponential {
		t.Fatal("MCP should not be detected after deletion")
	}

	// Instructions and skill should still be present
	content, _ := os.ReadFile("CLAUDE.md")
	if FindManagedBlock(string(content), "markdown") == nil {
		t.Fatal("instructions should still be present")
	}
	if !DetectSkillInstall(testClaudeCode).Local {
		t.Fatal("skill should still be installed")
	}
}

// --- Issues.db validation ---

func TestDoctor_IssuesDB_Valid(t *testing.T) {
	setupProject(t, "test")

	// Empty db is valid
	lineNum, err := validateIssuesDB(t)
	if err != nil {
		t.Fatalf("empty issues.db should be valid: %v", err)
	}
	if lineNum != 0 {
		t.Fatalf("expected 0 events, got %d", lineNum)
	}
}

func TestDoctor_IssuesDB_Invalid(t *testing.T) {
	setupProject(t, "test")

	// Write garbage to issues.db
	os.WriteFile(filepath.Join(".xpo", "issues.db"), []byte("not json\n"), 0644)

	lineNum, err := validateIssuesDB(t)
	if err == nil {
		t.Fatal("garbage issues.db should fail validation")
	}
	if lineNum != 1 {
		t.Fatalf("expected error at line 1, got %d", lineNum)
	}
}

func TestDoctor_IssuesDB_InvalidAtLine3(t *testing.T) {
	setupProject(t, "test")

	lines := []string{
		`{"type":"CREATE","issue_id":"test-abc123","timestamp":"2024-01-01T00:00:00Z","actor":"test","payload":{}}`,
		`{"type":"CREATE","issue_id":"test-def456","timestamp":"2024-01-01T00:00:01Z","actor":"test","payload":{}}`,
		`this is not json`,
		`{"type":"CREATE","issue_id":"test-ghi789","timestamp":"2024-01-01T00:00:02Z","actor":"test","payload":{}}`,
	}
	os.WriteFile(filepath.Join(".xpo", "issues.db"), []byte(strings.Join(lines, "\n")+"\n"), 0644)

	lineNum, err := validateIssuesDB(t)
	if err == nil {
		t.Fatal("should detect invalid entry")
	}
	if lineNum != 3 {
		t.Fatalf("expected error at line 3, got %d", lineNum)
	}
}

// validateIssuesDB mirrors storage.ValidateEvents without the import cycle.
func validateIssuesDB(t *testing.T) (int, error) {
	t.Helper()
	f, err := os.Open(filepath.Join(".xpo", "issues.db"))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		var raw json.RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			return line, fmt.Errorf("line %d: %w", line, err)
		}
	}
	return line, scanner.Err()
}
