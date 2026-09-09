package exponential

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectMCPConfig_Missing(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	status := DetectMCPConfig()
	if status.Exists || status.HasExponential {
		t.Fatalf("expected no file, got Exists=%v HasExponential=%v", status.Exists, status.HasExponential)
	}
}

func TestDetectMCPConfig_ExistsWithoutXpo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"other":{"command":"other"}}}`), 0644)

	status := DetectMCPConfig()
	if !status.Exists {
		t.Fatal("expected Exists=true")
	}
	if status.HasExponential {
		t.Fatal("expected HasExponential=false")
	}
}

func TestDetectMCPConfig_ExistsWithXpo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"xpo":{"command":"xpo","args":["mcp"]}}}`), 0644)

	status := DetectMCPConfig()
	if !status.Exists || !status.HasExponential {
		t.Fatalf("expected Exists=true HasExponential=true, got %v %v", status.Exists, status.HasExponential)
	}
}

func TestEnsureMCPConfig_CreatesNew(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	if err := EnsureMCPConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	var doc map[string]map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	xpo, ok := doc["mcpServers"]["xpo"].(map[string]interface{})
	if !ok {
		t.Fatal("xpo entry not found")
	}
	if xpo["command"] != "xpo" {
		t.Fatalf("expected command=xpo, got %v", xpo["command"])
	}
}

func TestHasAgentInstructions_NewHeading(t *testing.T) {
	content := "# Some project docs\n\n# Agent Instructions\n\nSome instructions here."
	if !HasAgentInstructions(content) {
		t.Fatal("expected true for new heading")
	}
}

func TestHasAgentInstructions_LegacyHeading(t *testing.T) {
	content := "# Some project docs\n\n# Exponential Agent Instructions\n\nSome instructions here."
	if !HasAgentInstructions(content) {
		t.Fatal("expected true for legacy heading")
	}
}

func TestHasAgentInstructions_NoHeading(t *testing.T) {
	content := "# Some project docs\n\nNothing related to agents."
	if HasAgentInstructions(content) {
		t.Fatal("expected false when no heading present")
	}
}

func TestAppendAgentInstructions_CreatesNew(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "Test Agent", File: "TEST.md", Format: "markdown"}
	if err := AppendAgentInstructions(agent, "test-"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile("TEST.md")
	s := string(content)
	if !strings.Contains(s, "# Agent Instructions") {
		t.Fatal("expected new heading in created file")
	}
	if !strings.Contains(s, "`test-`") {
		t.Fatal("expected prefix in created file")
	}
}

func TestAppendAgentInstructions_ReplacesLegacy(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	oldContent := "# My Project\n\nCustom docs here.\n\n# Exponential Agent Instructions\n\nOld stale instructions without skill rule."
	os.WriteFile("TEST.md", []byte(oldContent), 0644)

	agent := AgentConfig{Name: "Test Agent", File: "TEST.md", Format: "markdown"}
	if err := AppendAgentInstructions(agent, "test-"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile("TEST.md")
	s := string(content)
	if !strings.Contains(s, "# My Project") {
		t.Fatal("expected user content to be preserved")
	}
	if !strings.Contains(s, "Custom docs here.") {
		t.Fatal("expected user content to be preserved")
	}
	if strings.Contains(s, "# Exponential Agent Instructions") {
		t.Fatal("expected legacy heading to be replaced")
	}
	if !strings.Contains(s, "# Agent Instructions") {
		t.Fatal("expected new heading after replacement")
	}
	if !strings.Contains(s, "xpo-workflow") {
		t.Fatal("expected xpo-workflow skill rule in updated instructions")
	}
	if strings.Contains(s, "Old stale instructions") {
		t.Fatal("expected old instructions to be removed")
	}
}

func TestAppendAgentInstructions_ReplacesCurrentHeading(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	oldContent := "# My Project\n\nDocs.\n\n# Agent Instructions\n\nOutdated content with old-prefix-."
	os.WriteFile("TEST.md", []byte(oldContent), 0644)

	agent := AgentConfig{Name: "Test Agent", File: "TEST.md", Format: "markdown"}
	if err := AppendAgentInstructions(agent, "new-"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile("TEST.md")
	s := string(content)
	if !strings.Contains(s, "# My Project") {
		t.Fatal("expected user content to be preserved")
	}
	if !strings.Contains(s, "`new-`") {
		t.Fatal("expected updated prefix")
	}
	if strings.Contains(s, "old-prefix-") {
		t.Fatal("expected old prefix to be gone")
	}
}

func TestAppendAgentInstructions_AppendsToFileWithoutSection(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	existingContent := "# My Project\n\nSome existing content.\n"
	os.WriteFile("TEST.md", []byte(existingContent), 0644)

	agent := AgentConfig{Name: "Test Agent", File: "TEST.md", Format: "markdown"}
	if err := AppendAgentInstructions(agent, "test-"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile("TEST.md")
	s := string(content)
	if !strings.Contains(s, "# My Project") {
		t.Fatal("expected existing content to be preserved")
	}
	if !strings.Contains(s, "# Agent Instructions") {
		t.Fatal("expected agent instructions to be appended")
	}
}

func TestGenerateAgentStub_UsesNewHeading(t *testing.T) {
	stub := GenerateAgentStub("test-")
	if !strings.HasPrefix(stub, "# Agent Instructions") {
		t.Fatalf("expected stub to start with '# Agent Instructions', got: %s", stub[:50])
	}
	if strings.Contains(stub, "Exponential Agent Instructions") {
		t.Fatal("expected no legacy heading in stub")
	}
}

func TestEnsureMCPConfig_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"other":{"command":"other-tool","args":["serve"]}}}`), 0644)

	if err := EnsureMCPConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	var doc map[string]map[string]interface{}
	json.Unmarshal(data, &doc)

	if _, ok := doc["mcpServers"]["other"]; !ok {
		t.Fatal("existing 'other' entry was lost")
	}
	if _, ok := doc["mcpServers"]["xpo"]; !ok {
		t.Fatal("xpo entry not added")
	}
}

// --- New tests for MCPConfigSpec-based functions ---

func TestDetectMCPConfigFor_JSON_CustomKey(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile("opencode.json", []byte(`{"mcp":{"xpo":{"command":"xpo","args":["mcp"]}}}`), 0644)

	spec := MCPConfigSpec{File: "opencode.json", ServerKey: "mcp", Format: "json"}
	status := DetectMCPConfigFor(spec)
	if !status.Exists || !status.HasExponential {
		t.Fatalf("expected Exists=true HasExponential=true, got %v %v", status.Exists, status.HasExponential)
	}
}

func TestDetectMCPConfigFor_JSON_NoXpo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile("opencode.json", []byte(`{"mcp":{"other":{"command":"other"}}}`), 0644)

	spec := MCPConfigSpec{File: "opencode.json", ServerKey: "mcp", Format: "json"}
	status := DetectMCPConfigFor(spec)
	if !status.Exists {
		t.Fatal("expected Exists=true")
	}
	if status.HasExponential {
		t.Fatal("expected HasExponential=false")
	}
}

func TestDetectMCPConfigFor_TOML_WithXpo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.MkdirAll(".codex", 0755)
	os.WriteFile(".codex/config.toml", []byte("[mcp_servers.xpo]\ncommand = \"xpo\"\nargs = [\"mcp\"]\n"), 0644)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}
	status := DetectMCPConfigFor(spec)
	if !status.Exists || !status.HasExponential {
		t.Fatalf("expected Exists=true HasExponential=true, got %v %v", status.Exists, status.HasExponential)
	}
}

func TestDetectMCPConfigFor_TOML_WithoutXpo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.MkdirAll(".codex", 0755)
	os.WriteFile(".codex/config.toml", []byte("[mcp_servers.other]\ncommand = \"other\"\n"), 0644)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}
	status := DetectMCPConfigFor(spec)
	if !status.Exists {
		t.Fatal("expected Exists=true")
	}
	if status.HasExponential {
		t.Fatal("expected HasExponential=false")
	}
}

func TestDetectMCPConfigFor_EmptySpec(t *testing.T) {
	status := DetectMCPConfigFor(MCPConfigSpec{})
	if status.Exists || status.HasExponential {
		t.Fatal("empty spec should return empty status")
	}
}

func TestEnsureMCPConfigFor_JSON_CustomKey(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: "test.json", ServerKey: "mcp", Format: "json"}
	if err := EnsureMCPConfigFor(spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile("test.json")
	var doc map[string]map[string]interface{}
	json.Unmarshal(data, &doc)

	xpo, ok := doc["mcp"]["xpo"].(map[string]interface{})
	if !ok {
		t.Fatal("xpo entry not found under 'mcp' key")
	}
	if xpo["command"] != "xpo" {
		t.Fatalf("expected command=xpo, got %v", xpo["command"])
	}
}

func TestEnsureMCPConfigFor_JSON_LocalArrayStyle(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: "opencode.json", ServerKey: "mcp", Format: "json", EntryStyle: "local-array"}
	if err := EnsureMCPConfigFor(spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile("opencode.json")
	var doc map[string]map[string]interface{}
	json.Unmarshal(data, &doc)

	xpo, ok := doc["mcp"]["xpo"].(map[string]interface{})
	if !ok {
		t.Fatal("xpo entry not found under 'mcp' key")
	}
	if xpo["type"] != "local" {
		t.Fatalf("expected type=local, got %v", xpo["type"])
	}
	cmdArr, ok := xpo["command"].([]interface{})
	if !ok {
		t.Fatal("expected command to be an array")
	}
	if len(cmdArr) != 2 || cmdArr[0] != "xpo" || cmdArr[1] != "mcp" {
		t.Fatalf("expected command=[xpo, mcp], got %v", cmdArr)
	}
	if _, hasArgs := xpo["args"]; hasArgs {
		t.Fatal("local-array style should not have separate 'args' field")
	}
}

func TestEnsureMCPConfigFor_TOML_CreatesNew(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}
	if err := EnsureMCPConfigFor(spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(".codex/config.toml")
	content := string(data)
	if !strings.Contains(content, "[mcp_servers.xpo]") {
		t.Fatal("expected [mcp_servers.xpo] section")
	}
	if !strings.Contains(content, `command = "xpo"`) {
		t.Fatal("expected command = xpo")
	}
}

func TestEnsureMCPConfigFor_TOML_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.MkdirAll(".codex", 0755)
	os.WriteFile(".codex/config.toml", []byte("[mcp_servers.other]\ncommand = \"other\"\n"), 0644)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}
	if err := EnsureMCPConfigFor(spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(".codex/config.toml")
	content := string(data)
	if !strings.Contains(content, "[mcp_servers.other]") {
		t.Fatal("existing section was lost")
	}
	if !strings.Contains(content, "[mcp_servers.xpo]") {
		t.Fatal("xpo section not appended")
	}
}

func TestEnsureMCPConfigFor_TOML_NoopIfExists(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	original := "[mcp_servers.xpo]\ncommand = \"xpo\"\nargs = [\"mcp\"]\n"
	os.MkdirAll(".codex", 0755)
	os.WriteFile(".codex/config.toml", []byte(original), 0644)

	spec := MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}
	if err := EnsureMCPConfigFor(spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(".codex/config.toml")
	if string(data) != original {
		t.Fatal("file was modified when xpo section already exists")
	}
}

func TestEnsureMCPConfigFor_JSON_NeedsDir(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	spec := MCPConfigSpec{File: ".cursor/mcp.json", ServerKey: "mcpServers", Format: "json", NeedsDir: true}
	if err := EnsureMCPConfigFor(spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(".cursor/mcp.json")
	var doc map[string]map[string]interface{}
	json.Unmarshal(data, &doc)

	if _, ok := doc["mcpServers"]["xpo"]; !ok {
		t.Fatal("xpo entry not found")
	}
}

func TestLookupAgent_ByName(t *testing.T) {
	agent, ok := LookupAgent("Claude Code")
	if !ok {
		t.Fatal("expected to find Claude Code")
	}
	if agent.Binary != "claude" {
		t.Fatalf("expected binary=claude, got %s", agent.Binary)
	}
}

func TestLookupAgent_ByBinary(t *testing.T) {
	agent, ok := LookupAgent("codex")
	if !ok {
		t.Fatal("expected to find Codex by binary")
	}
	if agent.Name != "Codex" {
		t.Fatalf("expected name=Codex, got %s", agent.Name)
	}
}

func TestLookupAgent_CaseInsensitive(t *testing.T) {
	agent, ok := LookupAgent("CURSOR")
	if !ok {
		t.Fatal("expected to find Cursor case-insensitively")
	}
	if agent.Name != "Cursor" {
		t.Fatalf("expected name=Cursor, got %s", agent.Name)
	}
}

func TestLookupAgent_NotFound(t *testing.T) {
	_, ok := LookupAgent("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestMCPConfigSpec_HasMCPConfig(t *testing.T) {
	empty := MCPConfigSpec{}
	if empty.HasMCPConfig() {
		t.Fatal("empty spec should not have MCP config")
	}

	spec := MCPConfigSpec{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"}
	if !spec.HasMCPConfig() {
		t.Fatal("spec with file should have MCP config")
	}
}

func TestWriteAgentSkill_LocalInstall(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "Test", SkillDir: ".test/skills"}
	skillDir, err := WriteAgentSkill(agent, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skillDir != ".test/skills/xpo-workflow" {
		t.Fatalf("unexpected skill dir: %s", skillDir)
	}

	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatal("SKILL.md not created")
	}
	if _, err := os.Stat(filepath.Join(skillDir, "references", "spec-guide.md")); err != nil {
		t.Fatal("spec-guide.md not created")
	}
}

func TestWriteAgentSkill_NoSkillDir(t *testing.T) {
	agent := AgentConfig{Name: "Test"}
	skillDir, err := WriteAgentSkill(agent, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skillDir != "" {
		t.Fatalf("expected empty skill dir for agent without SkillDir, got %s", skillDir)
	}
}

func TestWriteAgentSkill_GlobalInstall(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	globalBase := filepath.Join(dir, "global-skills")
	agent := AgentConfig{Name: "Test", SkillDir: ".test/skills", GlobalSkillDir: ""}
	skillDir, err := WriteAgentSkill(agent, globalBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skillDir != filepath.Join(globalBase, "xpo-workflow") {
		t.Fatalf("unexpected skill dir: %s", skillDir)
	}

	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatal("SKILL.md not created in global location")
	}
}

func TestDetectSkillInstall_Local(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "Test", SkillDir: ".test/skills"}
	WriteAgentSkill(agent, "")

	status := DetectSkillInstall(agent)
	if !status.Local {
		t.Fatal("expected Local=true")
	}
	if status.Global {
		t.Fatal("expected Global=false")
	}
}

func TestDetectSkillInstall_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	agent := AgentConfig{Name: "Test", SkillDir: ".test/skills"}
	status := DetectSkillInstall(agent)
	if status.Local || status.Global || status.BrokenSymlink {
		t.Fatal("expected all false for uninstalled skill")
	}
}

func TestAgentRegistryNames(t *testing.T) {
	names := AgentRegistryNames()
	if len(names) != len(AgentRegistry) {
		t.Fatalf("expected %d names, got %d", len(AgentRegistry), len(names))
	}
	if names[0] != "Generic Agent" {
		t.Fatalf("expected first name to be Generic Agent, got %s", names[0])
	}
}

func TestAgentRegistry_TrimmedToSixHarnesses(t *testing.T) {
	if len(AgentRegistry) != 6 {
		t.Fatalf("expected 6 harnesses in registry, got %d", len(AgentRegistry))
	}

	expected := map[string]bool{
		"Generic Agent":  true,
		"Claude Code":    true,
		"GitHub Copilot": true,
		"Cursor":         true,
		"Codex":          true,
		"OpenCode":       true,
	}
	for _, a := range AgentRegistry {
		if !expected[a.Name] {
			t.Fatalf("unexpected agent in registry: %s", a.Name)
		}
	}
}
