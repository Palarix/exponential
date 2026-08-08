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
