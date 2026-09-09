package exponential

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/storage"
)

func setupFreshRepo(t *testing.T, branchName string) string {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	runGit(t, dir, "init", "-b", branchName)
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	config.Reset()
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
		config.Reset()
	})
	return dir
}

func readConfigField(t *testing.T, dir, field string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".xpo", "config.yaml"))
	if err != nil {
		t.Fatalf("failed to read config.yaml: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, field+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
		}
	}
	return ""
}

func TestInitProject_DefaultBranch_Main(t *testing.T) {
	dir := setupFreshRepo(t, "main")
	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	got := readConfigField(t, dir, "default_branch")
	if got != "main" {
		t.Errorf("expected default_branch=main, got %q", got)
	}
}

func TestInitProject_DefaultBranch_Master(t *testing.T) {
	dir := setupFreshRepo(t, "master")
	runGit(t, dir, "config", "init.defaultBranch", "master")
	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	got := readConfigField(t, dir, "default_branch")
	if got != "master" {
		t.Errorf("expected default_branch=master, got %q", got)
	}
}

func TestInitProject_DefaultBranch_Develop(t *testing.T) {
	dir := setupFreshRepo(t, "develop")
	runGit(t, dir, "config", "init.defaultBranch", "develop")
	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	got := readConfigField(t, dir, "default_branch")
	if got != "develop" {
		t.Errorf("expected default_branch=develop, got %q", got)
	}
}

func TestInitProject_DefaultBranch_Trunk(t *testing.T) {
	dir := setupFreshRepo(t, "trunk")
	runGit(t, dir, "config", "init.defaultBranch", "trunk")
	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	got := readConfigField(t, dir, "default_branch")
	if got != "trunk" {
		t.Errorf("expected default_branch=trunk, got %q", got)
	}
}

func TestInitProject_DefaultBranch_WithRemote(t *testing.T) {
	remoteDir := t.TempDir()
	remoteDir, _ = filepath.EvalSymlinks(remoteDir)
	runGit(t, remoteDir, "init", "--bare", "-b", "release")

	dir := setupFreshRepo(t, "release")
	runGit(t, dir, "remote", "add", "origin", remoteDir)
	runGit(t, dir, "push", "-u", "origin", "release")
	runGit(t, dir, "remote", "set-head", "origin", "release")

	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	got := readConfigField(t, dir, "default_branch")
	if got != "release" {
		t.Errorf("expected default_branch=release, got %q", got)
	}
}

func TestInitProject_DefaultBranch_ConfigOverridePreserved(t *testing.T) {
	dir := setupFreshRepo(t, "main")

	// First init writes default_branch=main
	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("first InitProject failed: %v", err)
	}

	// Manually change default_branch in config to "custom"
	cfgPath := filepath.Join(dir, ".xpo", "config.yaml")
	data, _ := os.ReadFile(cfgPath)
	updated := strings.Replace(string(data), "default_branch: main", "default_branch: custom", 1)
	os.WriteFile(cfgPath, []byte(updated), 0644)

	// Load config and set globally so DefaultBranch() returns the config value
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	config.Set(cfg)

	got := DefaultBranch()
	if got != "custom" {
		t.Errorf("expected DefaultBranch()=custom after config edit, got %q", got)
	}
}

func TestInitProject_DefaultBranch_NoGitRepo(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	config.Reset()
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
		config.Reset()
	})

	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}
	got := readConfigField(t, dir, "default_branch")
	if got != "main" {
		t.Errorf("expected default_branch=main (fallback), got %q", got)
	}
}

func TestInitProject_DefaultBranch_Force(t *testing.T) {
	dir := setupFreshRepo(t, "main")

	// First init
	_, err := InitProject(false, "test-")
	if err != nil {
		t.Fatalf("first InitProject failed: %v", err)
	}

	// Switch to a different branch, set it as init.defaultBranch
	runGit(t, dir, "checkout", "-b", "v2")
	runGit(t, dir, "config", "init.defaultBranch", "v2")
	// Delete the main branch so probe finds v2 via init.defaultBranch
	cmd := exec.Command("git", "-C", dir, "branch", "-D", "main")
	cmd.Run()
	storage.ResetHubRoot()

	// Force re-init should detect the new default
	_, err = InitProject(true, "test-")
	if err != nil {
		t.Fatalf("force InitProject failed: %v", err)
	}
	got := readConfigField(t, dir, "default_branch")
	if got != "v2" {
		t.Errorf("expected default_branch=v2 after force re-init, got %q", got)
	}
}
