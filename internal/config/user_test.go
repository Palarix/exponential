package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestInboxLastRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XPO_CONFIG_DIR", filepath.Join(dir, "xpo"))
	t.Chdir(dir)

	if got := GetInboxLastRead(); !got.IsZero() {
		t.Errorf("expected zero time for unset cursor, got %v", got)
	}

	want := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	if err := SetInboxLastRead(want); err != nil {
		t.Fatalf("SetInboxLastRead: %v", err)
	}

	if got := GetInboxLastRead(); !got.Equal(want) {
		t.Errorf("GetInboxLastRead = %v, want %v", got, want)
	}

	// Cursor must live under the projects map keyed by the absolute .xpo path.
	c := LoadUserConfig()
	absXpo, _ := filepath.Abs(".xpo")
	if c.Projects[absXpo].LastRead == "" {
		t.Errorf("expected last_read stored under projects[%q], got %+v", absXpo, c.Projects)
	}
}
