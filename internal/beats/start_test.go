package beats

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

func setupStartTestEnv(t *testing.T) (*Client, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)

	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
	}

	client := NewClient(cfg)
	cleanup := func() { os.Chdir(origDir) }
	return client, cleanup
}

func createTestIssue(t *testing.T, id, title, status string) {
	t.Helper()
	evt := model.Event{
		ID:   id,
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  title,
			Status: status,
		},
	}
	if err := storage.AppendEvent(evt); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
}

func TestStartWork_NonGitRepo(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "PLANNED")

	branch, msgs, err := client.StartWork("test-abc123")
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty branch in non-git repo, got %q", branch)
	}
	if len(msgs) == 0 {
		t.Error("expected update messages")
	}

	issue, err := client.GetIssue("test-abc123")
	if err != nil {
		t.Fatalf("GetIssue() error: %v", err)
	}
	if issue.Status != model.StatusDoing {
		t.Errorf("expected DOING, got %s", issue.Status)
	}
}

func TestStartWork_DoneIssue(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "DONE")

	_, _, err := client.StartWork("test-abc123")
	if err == nil {
		t.Fatal("expected error for DONE issue")
	}
}

func TestStartWork_BlockedIssue(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "BLOCKED")

	_, _, err := client.StartWork("test-abc123")
	if err == nil {
		t.Fatal("expected error for BLOCKED issue")
	}
}

func TestStartWork_GitRepo_CreatesBranch(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	// Initialize git repo with "main" as the default branch
	cwd, _ := os.Getwd()
	runGit(t, cwd, "init", "-b", "main")
	runGit(t, cwd, "config", "user.email", "test@test.com")
	runGit(t, cwd, "config", "user.name", "Test")
	os.WriteFile("dummy.txt", []byte("init"), 0644)
	runGit(t, cwd, "add", ".")
	runGit(t, cwd, "commit", "-m", "init")

	createTestIssue(t, "test-abc123", "Fix Login Flow", "PLANNED")

	branch, msgs, err := client.StartWork("test-abc123")
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}
	if branch != "test-abc123/fix-login-flow" {
		t.Errorf("expected branch 'test-abc123/fix-login-flow', got %q", branch)
	}

	foundBranchMsg := false
	for _, msg := range msgs {
		if msg == "Created and switched to branch 'test-abc123/fix-login-flow'" {
			foundBranchMsg = true
		}
	}
	if !foundBranchMsg {
		t.Errorf("expected branch creation message in %v", msgs)
	}

	// Verify we're on the new branch
	out, _ := exec.Command("git", "branch", "--show-current").Output()
	currentBranch := string(out)
	if currentBranch[:len(currentBranch)-1] != "test-abc123/fix-login-flow" {
		t.Errorf("expected to be on branch test-abc123/fix-login-flow, got %q", currentBranch)
	}
}

func TestStartWork_GitRepo_ExistingBranch(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	cwd, _ := os.Getwd()
	runGit(t, cwd, "init", "-b", "main")
	runGit(t, cwd, "config", "user.email", "test@test.com")
	runGit(t, cwd, "config", "user.name", "Test")
	os.WriteFile("dummy.txt", []byte("init"), 0644)
	runGit(t, cwd, "add", ".")
	runGit(t, cwd, "commit", "-m", "init")

	// Pre-create the branch
	runGit(t, cwd, "branch", "test-abc123/fix-login-flow")

	createTestIssue(t, "test-abc123", "Fix Login Flow", "PLANNED")

	branch, msgs, err := client.StartWork("test-abc123")
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}
	if branch != "test-abc123/fix-login-flow" {
		t.Errorf("expected branch 'test-abc123/fix-login-flow', got %q", branch)
	}

	foundSwitchMsg := false
	for _, msg := range msgs {
		if msg == "Switched to existing branch 'test-abc123/fix-login-flow'" {
			foundSwitchMsg = true
		}
	}
	if !foundSwitchMsg {
		t.Errorf("expected switch message in %v", msgs)
	}
}
