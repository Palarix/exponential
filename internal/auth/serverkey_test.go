package auth

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrGenerateServerKey_GeneratesNew(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.key")

	key, err := LoadOrGenerateServerKey(path)
	if err != nil {
		t.Fatalf("LoadOrGenerateServerKey: %v", err)
	}
	if len(key) != ed25519.PrivateKeySize {
		t.Errorf("expected %d byte key, got %d", ed25519.PrivateKeySize, len(key))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("key file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestLoadOrGenerateServerKey_LoadsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.key")

	key1, _ := LoadOrGenerateServerKey(path)
	key2, err := LoadOrGenerateServerKey(path)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if !key1.Equal(key2) {
		t.Error("expected same key on second load")
	}
}

func TestLoadOrGenerateServerKey_BadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.key")
	os.WriteFile(path, []byte("not a key"), 0600)

	_, err := LoadOrGenerateServerKey(path)
	if err == nil {
		t.Fatal("expected error for invalid key file")
	}
}
