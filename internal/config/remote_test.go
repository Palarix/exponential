package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadRemoteConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.MkdirAll(".beats", 0755)

	rc := RemoteConfig{
		URL:   "https://beats.example.com",
		Token: "test-token-abc",
	}
	if err := SaveRemoteConfig(rc); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	loaded := LoadRemoteConfig()
	if loaded.URL != rc.URL {
		t.Errorf("URL: got %q, want %q", loaded.URL, rc.URL)
	}
	if loaded.Token != rc.Token {
		t.Errorf("Token: got %q, want %q", loaded.Token, rc.Token)
	}

	// Verify file permissions
	info, _ := os.Stat(".beats/remote.yaml")
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %o", info.Mode().Perm())
	}
}

func TestLoadRemoteConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	rc := LoadRemoteConfig()
	if rc.URL != "" {
		t.Errorf("expected empty URL, got %q", rc.URL)
	}
}

func TestClearRemoteConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.MkdirAll(".beats", 0755)
	SaveRemoteConfig(RemoteConfig{URL: "https://test.com", Token: "tok"})

	if err := ClearRemoteConfig(); err != nil {
		t.Fatalf("ClearRemoteConfig: %v", err)
	}

	rc := LoadRemoteConfig()
	if rc.URL != "" {
		t.Errorf("expected empty after clear, got %q", rc.URL)
	}
}

func TestClearRemoteConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if err := ClearRemoteConfig(); err != nil {
		t.Fatalf("ClearRemoteConfig on missing file: %v", err)
	}
}

func TestSaveRemoteConfig_EnsuresGitignore(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.MkdirAll(".beats", 0755)

	SaveRemoteConfig(RemoteConfig{URL: "https://test.com", Token: "tok"})

	data, err := os.ReadFile(filepath.Join(".beats", ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "remote.yaml") {
		t.Error("expected remote.yaml in .gitignore")
	}
	if !strings.Contains(content, "server.key") {
		t.Error("expected server.key in .gitignore")
	}

	// Save again — should not duplicate entries
	SaveRemoteConfig(RemoteConfig{URL: "https://test.com", Token: "tok2"})
	data, _ = os.ReadFile(filepath.Join(".beats", ".gitignore"))
	if strings.Count(string(data), "remote.yaml") != 1 {
		t.Error("expected exactly one remote.yaml entry in .gitignore")
	}
}
