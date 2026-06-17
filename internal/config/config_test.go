package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigVersion(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "issue-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	xpoDir := filepath.Join(tmpDir, ".xpo")
	if err := os.MkdirAll(xpoDir, 0755); err != nil {
		t.Fatalf("Failed to create .xpo dir: %v", err)
	}

	// Change CWD
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer os.Chdir(originalWd)

	// Case 1: No version (legacy/default)
	os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte("prefix: test-\n"), 0644)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Version != 0 {
		t.Errorf("Expected version 0 (default), got %d", cfg.Version)
	}

	// Case 2: With version
	os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte("prefix: test-\nversion: 2\n"), 0644)
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Version != 2 {
		t.Errorf("Expected version 2, got %d", cfg.Version)
	}
}
