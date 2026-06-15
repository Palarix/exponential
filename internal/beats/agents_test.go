package beats

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
	if status.Exists || status.HasBeats {
		t.Fatalf("expected no file, got Exists=%v HasBeats=%v", status.Exists, status.HasBeats)
	}
}

func TestDetectMCPConfig_ExistsWithoutBeats(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"other":{"command":"other"}}}`), 0644)

	status := DetectMCPConfig()
	if !status.Exists {
		t.Fatal("expected Exists=true")
	}
	if status.HasBeats {
		t.Fatal("expected HasBeats=false")
	}
}

func TestDetectMCPConfig_ExistsWithBeats(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"beats":{"command":"beats","args":["mcp"]}}}`), 0644)

	status := DetectMCPConfig()
	if !status.Exists || !status.HasBeats {
		t.Fatalf("expected Exists=true HasBeats=true, got %v %v", status.Exists, status.HasBeats)
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

	beats, ok := doc["mcpServers"]["beats"].(map[string]interface{})
	if !ok {
		t.Fatal("beats entry not found")
	}
	if beats["command"] != "beats" {
		t.Fatalf("expected command=beats, got %v", beats["command"])
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
	if _, ok := doc["mcpServers"]["beats"]; !ok {
		t.Fatal("beats entry not added")
	}
}
