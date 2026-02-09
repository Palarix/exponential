package beats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
)

func TestRunMigrations(t *testing.T) {
	// Setup env
	tmpDir, err := os.MkdirTemp("", "beats-migrate-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	beatsDir := filepath.Join(tmpDir, ".beats")
	if err := os.MkdirAll(beatsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beats dir: %v", err)
	}

	cwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(cwd)

	// Create config.yaml with version 0 (legacy)
	configPath := filepath.Join(beatsDir, "config.yaml")
	os.WriteFile(configPath, []byte("prefix: test-\nversion: 0\n"), 0644)

	// Create issue with ParentID (legacy format)
	// We need to write raw event because AddIssue might use new format logic
	// But AddIssue creates EventTypeCreate which uses CreatePayload.
	// CreatePayload still has ParentID.

	// Write raw event
	evt := model.Event{
		ID:   "test-1",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:    "Parent",
			ParentID: "", // No parent
		},
		CreatedAt: time.Now().UTC(),
	}
	storage.AppendEvent(evt)

	evt2 := model.Event{
		ID:   "test-2",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:    "Child",
			ParentID: "test-1", // Legacy parent reference
		},
		CreatedAt: time.Now().UTC(),
	}
	storage.AppendEvent(evt2)

	// Load Client
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	client := NewClient(cfg)

	// Run Migrations
	report, err := client.RunMigrations(MigrateOptions{DryRun: false})
	if err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify Report
	if len(report) == 0 {
		t.Errorf("Expected migration report items")
	}

	// Verify Config Update
	bytes, _ := os.ReadFile(configPath)
	if !strings.Contains(string(bytes), "version: 1") {
		t.Errorf("Config should be updated to version 1. Content: %s", string(bytes))
	}

	// Verify Data Update
	events, _ := storage.ReadEvents()
	lastEvt := events[len(events)-1]
	if lastEvt.Type != model.EventTypeUpdate {
		t.Errorf("Expected last event to be update")
	}
	// TODO: verify payload has dependencies, but that requires unmarshaling specific payload type
}

func TestRunMigrationsIdempotency(t *testing.T) {
	// Setup env
	tmpDir, _ := os.MkdirTemp("", "beats-migrate-idempotency-*")
	defer os.RemoveAll(tmpDir)
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	cwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(cwd)

	// Create config with version 1 (already migrated)
	configPath := filepath.Join(beatsDir, "config.yaml")
	os.WriteFile(configPath, []byte("prefix: test-\nversion: 1\n"), 0644)

	// Load Client
	cfg, _ := config.LoadConfig()
	client := NewClient(cfg)

	// Run Migrations
	report, err := client.RunMigrations(MigrateOptions{DryRun: false})
	if err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Should be empty report
	if len(report) > 0 {
		t.Errorf("Expected no migrations for version 1")
	}
}
