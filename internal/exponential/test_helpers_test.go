package exponential

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func setupLocalTransport(t *testing.T) *LocalTransport {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	return &LocalTransport{
		Config: &config.Config{
			Prefix: "test-",
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
