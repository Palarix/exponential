package exponential

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func setupClient(t *testing.T) *Client {
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

	cfg := &config.Config{
		Prefix:      "test-",
		Automations: config.Automations{},
	}
	c := NewClient(cfg)
	c.UserOverride = "Tester <test@example.com>"
	return c
}

func TestClient_AddAndGetIssue(t *testing.T) {
	c := setupClient(t)
	issue, err := c.AddIssue(model.CreatePayload{Title: "via client"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.GetIssue(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "via client" {
		t.Errorf("title = %q", got.Title)
	}
}

func TestClient_ListIssues(t *testing.T) {
	c := setupClient(t)
	c.AddIssue(model.CreatePayload{Title: "A"})
	c.AddIssue(model.CreatePayload{Title: "B", Status: "PLANNED"})

	issues, err := c.ListIssues(FilterOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestClient_ListIssues_Filtered(t *testing.T) {
	c := setupClient(t)
	c.AddIssue(model.CreatePayload{Title: "A"})
	c.AddIssue(model.CreatePayload{Title: "B", Status: "PLANNED"})

	issues, err := c.ListIssues(FilterOptions{Statuses: []string{"PLANNED"}, All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 PLANNED issue, got %d", len(issues))
	}
}

func TestClient_UpdateIssue(t *testing.T) {
	c := setupClient(t)
	issue, _ := c.AddIssue(model.CreatePayload{Title: "update me"})

	_, err := c.UpdateIssue(issue.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err != nil {
		t.Fatal(err)
	}

	got, _ := c.GetIssue(issue.ID)
	if got.Status != model.StatusDoing {
		t.Errorf("status = %s", got.Status)
	}
}

func TestClient_AddComment(t *testing.T) {
	c := setupClient(t)
	issue, _ := c.AddIssue(model.CreatePayload{Title: "comment me"})

	err := c.AddComment(issue.ID, "hello")
	if err != nil {
		t.Fatal(err)
	}

	got, _ := c.GetIssue(issue.ID)
	if len(got.Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(got.Comments))
	}
}

func TestClient_DeleteIssue_NoCascade(t *testing.T) {
	c := setupClient(t)
	issue, _ := c.AddIssue(model.CreatePayload{Title: "delete me"})

	err := c.DeleteIssue(issue.ID, "test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.GetIssue(issue.ID)
	if err == nil {
		t.Error("deleted issue should not be found")
	}
}

func TestClient_DeleteIssue_WithCascade(t *testing.T) {
	c := setupClient(t)
	parent, _ := c.AddIssue(model.CreatePayload{Title: "parent"})
	child, _ := c.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID})

	err := c.DeleteIssue(parent.ID, "cascade", true)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.GetIssue(child.ID)
	if err == nil {
		t.Error("cascade-deleted child should not be found")
	}
}

func TestClient_GetUser(t *testing.T) {
	c := setupClient(t)
	if got := c.GetUser(); got != "Tester <test@example.com>" {
		t.Errorf("GetUser() = %q", got)
	}
}

func TestClient_GetInbox(t *testing.T) {
	c := setupClient(t)
	c.AddIssue(model.CreatePayload{Title: "inbox issue"})

	items, err := c.GetInbox(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	// All events are by the same user so inbox filters them out
	if len(items) != 0 {
		t.Errorf("expected 0 items (own actions filtered), got %d", len(items))
	}
}

func TestClient_FindIssue(t *testing.T) {
	c := setupClient(t)
	parent, _ := c.AddIssue(model.CreatePayload{Title: "parent"})
	c.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID})

	issue, children, _, err := c.FindIssue(parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if issue.Title != "parent" {
		t.Errorf("title = %q", issue.Title)
	}
	if len(children) != 1 {
		t.Errorf("expected 1 child, got %d", len(children))
	}
}

func TestClient_SyncLocal_PropagatesCollapse(t *testing.T) {
	c := setupClient(t)
	c.Collapse = true
	c.syncLocal()
	if !c.local.Collapse {
		t.Error("Collapse not propagated to local transport")
	}
}
