package exponential

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- xpo init scenarios ---

func TestScenario_Init_CleanProject(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	res, err := InitProject(false, "test")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	if !res.Created {
		t.Fatal("expected Created=true on fresh init")
	}

	if _, err := os.Stat(".xpo"); err != nil {
		t.Fatal(".xpo directory not created")
	}
	if _, err := os.Stat(filepath.Join(".xpo", "config.yaml")); err != nil {
		t.Fatal("config.yaml not created")
	}
	if _, err := os.Stat(filepath.Join(".xpo", "issues.db")); err != nil {
		t.Fatal("issues.db not created")
	}

	gitignore, _ := os.ReadFile(".gitignore")
	for _, entry := range []string{".xpo/issues.snapshot.json", ".xpo/git.lock", ".xpo/worktrees/"} {
		if !strings.Contains(string(gitignore), entry) {
			t.Fatalf(".gitignore missing entry: %s", entry)
		}
	}

	gitattrs, _ := os.ReadFile(".gitattributes")
	if !strings.Contains(string(gitattrs), ".xpo/issues.db merge=union") {
		t.Fatal(".gitattributes missing merge=union rule")
	}

	// Config should store the bare prefix
	configContent, _ := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
	if !strings.Contains(string(configContent), "prefix: test") {
		t.Fatal("config should contain bare prefix")
	}
}

func TestScenario_Init_ExistingProject_Idempotent(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	res1, err := InitProject(false, "test")
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if !res1.Created {
		t.Fatal("expected Created=true on first init")
	}

	os.WriteFile(filepath.Join(".xpo", "issues.db"), []byte("test-data\n"), 0644)
	configBefore, _ := os.ReadFile(filepath.Join(".xpo", "config.yaml"))

	res2, err := InitProject(false, "test")
	if err != nil {
		t.Fatalf("second init failed: %v", err)
	}
	if res2.Created {
		t.Fatal("expected Created=false on re-init")
	}

	issuesData, _ := os.ReadFile(filepath.Join(".xpo", "issues.db"))
	if !strings.Contains(string(issuesData), "test-data") {
		t.Fatal("issues.db was destroyed on re-init")
	}

	configAfter, _ := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
	if string(configBefore) != string(configAfter) {
		t.Fatal("config.yaml was rewritten on re-init without --force")
	}
}

func TestScenario_Init_Force_RewritesConfig(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	InitProject(false, "test")

	os.WriteFile(filepath.Join(".xpo", "config.yaml"), []byte("corrupted"), 0644)

	res, err := InitProject(true, "test")
	if err != nil {
		t.Fatalf("force init failed: %v", err)
	}
	if res.Created {
		t.Fatal("expected Created=false since .xpo already exists")
	}

	config, _ := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
	if string(config) == "corrupted" {
		t.Fatal("config was not rewritten with --force")
	}
	if !strings.Contains(string(config), "version:") {
		t.Fatal("rewritten config missing expected content")
	}
}

// --- xpo init mcp scenarios ---

func TestScenario_InitMCP_Clean(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"}

	status := DetectMCPConfigFor(spec)
	if status.Exists {
		t.Fatal("expected no file initially")
	}

	err := EnsureMCPConfigFor(spec)
	if err != nil {
		t.Fatalf("EnsureMCPConfigFor failed: %v", err)
	}

	status = DetectMCPConfigFor(spec)
	if !status.Exists || !status.HasExponential {
		t.Fatal("MCP config should exist with xpo entry after install")
	}
}

func TestScenario_InitMCP_AlreadyConfigured_NoForce(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"}

	EnsureMCPConfigFor(spec)
	dataBefore, _ := os.ReadFile(".mcp.json")

	status := DetectMCPConfigFor(spec)
	if !status.HasExponential {
		t.Fatal("should detect existing xpo entry")
	}

	EnsureMCPConfigFor(spec)
	dataAfter, _ := os.ReadFile(".mcp.json")

	if string(dataBefore) != string(dataAfter) {
		t.Fatal("MCP config was modified on re-install (should be idempotent for JSON)")
	}
}

func TestScenario_InitMCP_PreservesOtherEntries(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(".mcp.json", []byte(`{"mcpServers":{"other-tool":{"command":"other","args":["serve"]}}}`), 0644)

	spec := MCPConfigSpec{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"}
	EnsureMCPConfigFor(spec)

	data, _ := os.ReadFile(".mcp.json")
	var doc map[string]map[string]interface{}
	json.Unmarshal(data, &doc)

	if _, ok := doc["mcpServers"]["other-tool"]; !ok {
		t.Fatal("existing server entry was lost")
	}
	if _, ok := doc["mcpServers"]["xpo"]; !ok {
		t.Fatal("xpo entry was not added")
	}
}

func TestScenario_InitMCP_TOML_Clean(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}

	err := EnsureMCPConfigFor(spec)
	if err != nil {
		t.Fatalf("EnsureMCPConfigFor TOML failed: %v", err)
	}

	data, _ := os.ReadFile(".codex/config.toml")
	content := string(data)
	if !strings.Contains(content, "[mcp_servers.xpo]") {
		t.Fatal("TOML file missing [mcp_servers.xpo] section")
	}
	if !strings.Contains(content, `command = "xpo"`) {
		t.Fatal("TOML file missing command")
	}
	if !strings.Contains(content, `args = ["mcp"]`) {
		t.Fatal("TOML file missing args")
	}
}

func TestScenario_InitMCP_TOML_AlreadyConfigured(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}

	EnsureMCPConfigFor(spec)
	dataBefore, _ := os.ReadFile(".codex/config.toml")

	EnsureMCPConfigFor(spec)
	dataAfter, _ := os.ReadFile(".codex/config.toml")

	if string(dataBefore) != string(dataAfter) {
		t.Fatal("TOML config was modified on re-install")
	}
}

func TestScenario_InitMCP_TOML_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.MkdirAll(".codex", 0755)
	existing := "[some_other_section]\nkey = \"value\"\n"
	os.WriteFile(".codex/config.toml", []byte(existing), 0644)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}
	EnsureMCPConfigFor(spec)

	data, _ := os.ReadFile(".codex/config.toml")
	content := string(data)
	if !strings.Contains(content, "[some_other_section]") {
		t.Fatal("existing TOML content was lost")
	}
	if !strings.Contains(content, "[mcp_servers.xpo]") {
		t.Fatal("xpo section was not appended")
	}
}

func TestScenario_InitMCP_MultipleFormats(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	specs := []MCPConfigSpec{
		{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"},
		{File: ".cursor/mcp.json", ServerKey: "mcpServers", Format: "json", NeedsDir: true},
		{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true},
		{File: "opencode.json", ServerKey: "mcp", Format: "json", EntryStyle: "local-array"},
	}

	for _, spec := range specs {
		if err := EnsureMCPConfigFor(spec); err != nil {
			t.Fatalf("failed for %s: %v", spec.File, err)
		}
		status := DetectMCPConfigFor(spec)
		if !status.Exists || !status.HasExponential {
			t.Fatalf("detection failed for %s after install", spec.File)
		}
	}

	data, _ := os.ReadFile("opencode.json")
	var doc map[string]map[string]interface{}
	json.Unmarshal(data, &doc)
	xpo := doc["mcp"]["xpo"].(map[string]interface{})
	if xpo["type"] != "local" {
		t.Fatal("OpenCode entry should have type=local")
	}
	cmdArr, ok := xpo["command"].([]interface{})
	if !ok || len(cmdArr) != 2 {
		t.Fatal("OpenCode entry should have command as array")
	}
}

// --- xpo init skill scenarios ---

func TestScenario_InitSkill_Clean(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{
		Name:     "Claude Code",
		File:     "CLAUDE.md",
		SkillDir: ".claude/skills",
		Format:   "markdown",
	}

	err := AppendAgentInstructions(agent, "test")
	if err != nil {
		t.Fatalf("AppendAgentInstructions failed: %v", err)
	}

	content, _ := os.ReadFile("CLAUDE.md")
	s := string(content)
	if !strings.Contains(s, "## Exponential (xpo)") {
		t.Fatal("CLAUDE.md missing agent instructions")
	}
	if !strings.Contains(s, "`test`") {
		t.Fatal("CLAUDE.md missing prefix")
	}
	if !strings.Contains(s, "`test-a1b2c3`") {
		t.Fatal("CLAUDE.md missing example ID with separator")
	}
	// Should have managed block markers
	if !strings.Contains(s, "<!-- xpo:begin") {
		t.Fatal("CLAUDE.md should have managed block begin marker")
	}
	if !strings.Contains(s, "<!-- xpo:end -->") {
		t.Fatal("CLAUDE.md should have managed block end marker")
	}

	skillDir, err := WriteAgentSkill(agent, "")
	if err != nil {
		t.Fatalf("WriteAgentSkill failed: %v", err)
	}
	if skillDir == "" {
		t.Fatal("expected non-empty skill dir")
	}

	for _, file := range []string{"SKILL.md", "references/spec-guide.md", "references/mcp-tools.md"} {
		path := filepath.Join(skillDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("skill file missing: %s", path)
		}
	}

	status := DetectSkillInstall(agent)
	if !status.Local {
		t.Fatal("skill should be detected as locally installed")
	}
}

func TestScenario_InitSkill_ExistingFile_NoXpoSection(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	existingContent := "# My Project\n\nThis is my custom documentation.\n\n## Build Instructions\n\nRun `make build`.\n"
	os.WriteFile("CLAUDE.md", []byte(existingContent), 0644)

	agent := AgentConfig{Name: "Claude Code", File: "CLAUDE.md", SkillDir: ".claude/skills", Format: "markdown"}

	if HasAgentInstructions(existingContent) {
		t.Fatal("should not detect xpo section in vanilla CLAUDE.md")
	}

	err := AppendAgentInstructions(agent, "myproject")
	if err != nil {
		t.Fatalf("AppendAgentInstructions failed: %v", err)
	}

	content, _ := os.ReadFile("CLAUDE.md")
	s := string(content)

	if !strings.Contains(s, "# My Project") {
		t.Fatal("user's heading was lost")
	}
	if !strings.Contains(s, "This is my custom documentation.") {
		t.Fatal("user's content was lost")
	}
	if !strings.Contains(s, "## Build Instructions") {
		t.Fatal("user's build section was lost")
	}
	if !strings.Contains(s, "## Exponential (xpo)") {
		t.Fatal("agent instructions not appended")
	}
	if !strings.Contains(s, "`myproject`") {
		t.Fatal("prefix not in appended section")
	}
	if !strings.Contains(s, "<!-- xpo:begin") {
		t.Fatal("managed block markers not present")
	}
}

func TestScenario_InitSkill_ExistingXpoSection_Replaced(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// Legacy format: heading-based section without managed block markers
	existingContent := "# My Project\n\nCustom docs.\n\n# Agent Instructions\n\nOld xpo instructions with old-prefix.\n"
	os.WriteFile("CLAUDE.md", []byte(existingContent), 0644)

	agent := AgentConfig{Name: "Claude Code", File: "CLAUDE.md", SkillDir: ".claude/skills", Format: "markdown"}

	if !HasAgentInstructions(existingContent) {
		t.Fatal("should detect existing xpo section")
	}

	err := AppendAgentInstructions(agent, "newprefix")
	if err != nil {
		t.Fatalf("AppendAgentInstructions failed: %v", err)
	}

	content, _ := os.ReadFile("CLAUDE.md")
	s := string(content)

	if !strings.Contains(s, "# My Project") {
		t.Fatal("user's heading was lost")
	}
	if !strings.Contains(s, "Custom docs.") {
		t.Fatal("user's content was lost")
	}
	if strings.Contains(s, "Old xpo instructions") {
		t.Fatal("old xpo content was not replaced")
	}
	if strings.Contains(s, "old-prefix") {
		t.Fatal("old prefix still present")
	}
	if !strings.Contains(s, "`newprefix`") {
		t.Fatal("new prefix not in replaced section")
	}
	// Should now have managed block markers (migration)
	if !strings.Contains(s, "<!-- xpo:begin") {
		t.Fatal("legacy section should be migrated to managed block")
	}
}

func TestScenario_InitSkill_ManagedBlock_Updated(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "Claude Code", File: "CLAUDE.md", SkillDir: ".claude/skills", Format: "markdown"}

	// First write creates managed block
	AppendAgentInstructions(agent, "test")
	content1, _ := os.ReadFile("CLAUDE.md")
	block1 := FindManagedBlock(string(content1), "markdown")
	if block1 == nil {
		t.Fatal("expected managed block after first write")
	}

	// Second write updates managed block
	AppendAgentInstructions(agent, "test")
	content2, _ := os.ReadFile("CLAUDE.md")
	block2 := FindManagedBlock(string(content2), "markdown")
	if block2 == nil {
		t.Fatal("expected managed block after second write")
	}

	// Content should be the same (idempotent)
	if block1.Content != block2.Content {
		t.Fatal("managed block content should be identical on idempotent write")
	}
}

func TestScenario_InitSkill_ManagedBlock_EditedByHand(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "Claude Code", File: "CLAUDE.md", SkillDir: ".claude/skills", Format: "markdown"}

	// First write
	AppendAgentInstructions(agent, "test")

	// Simulate hand-edit: modify the managed block content
	content, _ := os.ReadFile("CLAUDE.md")
	modified := strings.Replace(string(content), "## Exponential (xpo)", "# My Custom Instructions", 1)
	os.WriteFile("CLAUDE.md", []byte(modified), 0644)

	// Verify the block is detected as edited
	modContent, _ := os.ReadFile("CLAUDE.md")
	block := FindManagedBlock(string(modContent), "markdown")
	if block == nil {
		t.Fatal("expected to find managed block")
	}
	if !BlockWasEdited(block) {
		t.Fatal("block should be detected as edited")
	}

	// WriteAgentInstructions without force should skip
	result, err := WriteAgentInstructions(agent, "test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "skipped" {
		t.Fatalf("expected skipped, got %s", result.Action)
	}
	if !result.WasEdited {
		t.Fatal("expected WasEdited=true")
	}

	// With force should overwrite
	result2, err := WriteAgentInstructions(agent, "test", true)
	if err != nil {
		t.Fatalf("force write failed: %v", err)
	}
	if result2.Action != "updated" {
		t.Fatalf("expected updated, got %s", result2.Action)
	}
}

func TestScenario_InitSkill_SharedAGENTSmd(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	codex := AgentConfig{Name: "Codex", File: "AGENTS.md", SkillDir: ".codex/skills", Format: "markdown"}
	opencode := AgentConfig{Name: "OpenCode", File: "AGENTS.md", SkillDir: ".opencode/skills", Format: "markdown"}

	AppendAgentInstructions(codex, "test")
	content1, _ := os.ReadFile("AGENTS.md")

	AppendAgentInstructions(opencode, "test")
	content2, _ := os.ReadFile("AGENTS.md")

	// Content should be identical (same managed block, same prefix)
	if string(content1) != string(content2) {
		t.Fatal("second write to shared AGENTS.md changed content")
	}
}

func TestScenario_InitSkill_SkillAlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "Test", SkillDir: ".test/skills"}

	WriteAgentSkill(agent, "")
	status := DetectSkillInstall(agent)
	if !status.Local {
		t.Fatal("skill should be detected after first install")
	}

	skillDir, err := WriteAgentSkill(agent, "")
	if err != nil {
		t.Fatalf("force re-install failed: %v", err)
	}

	skillContent, _ := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if !strings.Contains(string(skillContent), "xpo Development Workflow") {
		t.Fatal("SKILL.md content invalid after re-install")
	}
}

func TestScenario_InitSkill_GlobalInstall(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	globalBase := filepath.Join(dir, ".config", "xpo", "skills")
	agent := AgentConfig{Name: "Test", SkillDir: ".test/skills", GlobalSkillDir: ""}

	skillDir, err := WriteAgentSkill(agent, globalBase)
	if err != nil {
		t.Fatalf("global install failed: %v", err)
	}

	expectedDir := filepath.Join(globalBase, "xpo")
	if skillDir != expectedDir {
		t.Fatalf("expected skill dir %s, got %s", expectedDir, skillDir)
	}

	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatal("SKILL.md not created in global location")
	}

	if _, err := os.Stat(filepath.Join(".test", "skills", "xpo", "SKILL.md")); err == nil {
		t.Fatal("local skill should NOT be created during global install")
	}
}

func TestScenario_InitSkill_AgentWithoutSkillSupport(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "GitHub Copilot", File: ".github/copilot-instructions.md", Format: "markdown", NeedsDir: true}

	err := AppendAgentInstructions(agent, "test")
	if err != nil {
		t.Fatalf("AppendAgentInstructions failed: %v", err)
	}

	content, _ := os.ReadFile(".github/copilot-instructions.md")
	s := string(content)

	if !strings.Contains(s, "## Workflow") {
		t.Fatal("agent without skill support should get full docs with Workflow section")
	}
	if !strings.Contains(s, "## MCP Tools Reference") {
		t.Fatal("agent without skill support should get MCP tools reference inline")
	}

	skillDir, err := WriteAgentSkill(agent, "")
	if err != nil {
		t.Fatalf("WriteAgentSkill failed: %v", err)
	}
	if skillDir != "" {
		t.Fatal("expected empty skill dir for agent without SkillDir")
	}
}

// --- Full workflow: init → MCP → skill ---

func TestScenario_FullWorkflow(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// Step 1: xpo init
	res, err := InitProject(false, "test")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	if !res.Created {
		t.Fatal("expected .xpo created")
	}

	// Step 2: MCP config
	mcpSpec := MCPConfigSpec{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"}
	if err := EnsureMCPConfigFor(mcpSpec); err != nil {
		t.Fatalf("EnsureMCPConfigFor failed: %v", err)
	}
	status := DetectMCPConfigFor(mcpSpec)
	if !status.HasExponential {
		t.Fatal("MCP config missing xpo entry")
	}

	// Step 3: Agent instructions + skill
	agent := AgentConfig{
		Name:     "Claude Code",
		File:     "CLAUDE.md",
		SkillDir: ".claude/skills",
		Format:   "markdown",
	}

	if err := AppendAgentInstructions(agent, "test"); err != nil {
		t.Fatalf("AppendAgentInstructions failed: %v", err)
	}
	if _, err := WriteAgentSkill(agent, ""); err != nil {
		t.Fatalf("WriteAgentSkill failed: %v", err)
	}

	// Verify everything is in place
	if _, err := os.Stat(".xpo/config.yaml"); err != nil {
		t.Fatal("config.yaml missing")
	}
	if _, err := os.Stat(".mcp.json"); err != nil {
		t.Fatal(".mcp.json missing")
	}
	if _, err := os.Stat("CLAUDE.md"); err != nil {
		t.Fatal("CLAUDE.md missing")
	}
	if _, err := os.Stat(".claude/skills/xpo/SKILL.md"); err != nil {
		t.Fatal("skill SKILL.md missing")
	}

	// Verify CLAUDE.md has managed block
	claudeContent, _ := os.ReadFile("CLAUDE.md")
	if !strings.Contains(string(claudeContent), "<!-- xpo:begin") {
		t.Fatal("CLAUDE.md should have managed block markers")
	}

	// Verify re-running init doesn't break anything
	res2, err := InitProject(false, "test")
	if err != nil {
		t.Fatalf("re-init failed: %v", err)
	}
	if res2.Created {
		t.Fatal("re-init should not report Created")
	}

	// MCP config still intact
	status = DetectMCPConfigFor(mcpSpec)
	if !status.HasExponential {
		t.Fatal("MCP config lost after re-init")
	}

	// Skills still intact
	skillStatus := DetectSkillInstall(agent)
	if !skillStatus.Local {
		t.Fatal("skill lost after re-init")
	}
}

// --- Prefix handling ---

func TestDefaultPrefix_NoDash(t *testing.T) {
	prefix := DefaultPrefix()
	if strings.HasSuffix(prefix, "-") {
		t.Fatalf("DefaultPrefix should not end with dash, got %q", prefix)
	}
}
