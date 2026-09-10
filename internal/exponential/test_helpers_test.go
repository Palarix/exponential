package exponential

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func setupLocalTransport(t *testing.T) *LocalTransport {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)

	exec.Command("git", "init", "-b", "main", dir).Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	storage.ResetRefStore()
	if err := storage.InitRefStore(); err != nil {
		t.Fatalf("InitRefStore: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(oldWd)
		storage.ResetHubRoot()
		storage.ResetRefStore()
	})

	return &LocalTransport{
		Config: &config.Config{
			Prefix:      "test-",
			Automations: config.Automations{},
		},
		UserOverride: "Tester <test@example.com>",
	}
}

func readAllIssues(t *testing.T) map[string]*model.Issue {
	t.Helper()
	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	return ProjectIssues(events)
}

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }
