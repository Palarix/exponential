package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobalAccessor(t *testing.T) {
	defer Reset()

	if got := Get(); got != nil {
		t.Errorf("expected nil before Set, got %+v", got)
	}

	cfg := &Config{DefaultBranch: "develop"}
	Set(cfg)

	got := Get()
	if got == nil {
		t.Fatal("expected non-nil after Set")
	}
	if got.DefaultBranch != "develop" {
		t.Errorf("expected develop, got %s", got.DefaultBranch)
	}

	Reset()
	if got := Get(); got != nil {
		t.Errorf("expected nil after Reset, got %+v", got)
	}
}

func TestLoadConfigDefaultBranch(t *testing.T) {
	tmpDir := t.TempDir()
	xpoDir := filepath.Join(tmpDir, ".xpo")
	if err := os.MkdirAll(xpoDir, 0755); err != nil {
		t.Fatalf("Failed to create .xpo dir: %v", err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte("prefix: test-\ndefault_branch: trunk\n"), 0644)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.DefaultBranch != "trunk" {
		t.Errorf("expected trunk, got %s", cfg.DefaultBranch)
	}
}

func TestLoadConfigDefaultBranchEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	xpoDir := filepath.Join(tmpDir, ".xpo")
	if err := os.MkdirAll(xpoDir, 0755); err != nil {
		t.Fatalf("Failed to create .xpo dir: %v", err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte("prefix: test-\n"), 0644)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.DefaultBranch != "" {
		t.Errorf("expected empty default_branch, got %s", cfg.DefaultBranch)
	}
}

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

func loadConfigWith(t *testing.T, yaml string) (warnings string) {
	t.Helper()
	tmpDir := t.TempDir()
	xpoDir := filepath.Join(tmpDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(originalWd) })

	var buf bytes.Buffer
	origOut := warnOut
	warnOut = &buf
	t.Cleanup(func() { warnOut = origOut })

	os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte(yaml), 0644)
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	return buf.String()
}

func TestLoadConfigWarnsOnWorktreesFalse(t *testing.T) {
	got := loadConfigWith(t, "prefix: test-\nworktrees: false\n")
	if !strings.Contains(got, "worktrees: false is no longer supported") {
		t.Errorf("expected a worktrees warning, got %q", got)
	}
}

func TestLoadConfigAcceptsLegacyWorktreesTrueSilently(t *testing.T) {
	if got := loadConfigWith(t, "prefix: test-\nworktrees: true\n"); got != "" {
		t.Errorf("worktrees: true should load silently, got %q", got)
	}
	if got := loadConfigWith(t, "prefix: test-\n"); got != "" {
		t.Errorf("a config without the key should load silently, got %q", got)
	}
}
