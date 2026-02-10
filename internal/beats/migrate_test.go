package beats

import (
	"testing"
)

func TestRunMigrations_AlreadyUpToDate(t *testing.T) {
	version, err := RunMigrations(2)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if version != 2 {
		t.Errorf("Expected version 2, got %d", version)
	}
}

func TestRunMigrations_OldVersion(t *testing.T) {
	_, err := RunMigrations(1)
	if err == nil {
		t.Error("Expected error for old version")
	}
}

func TestRunMigrations_V0(t *testing.T) {
	_, err := RunMigrations(0)
	if err == nil {
		t.Error("Expected error for v0")
	}
}
