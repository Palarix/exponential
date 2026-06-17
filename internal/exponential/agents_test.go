package exponential

import (
	"encoding/json"
	"os"
	"path/filepath"
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
