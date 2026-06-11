package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func generateTestKey(t *testing.T) (ssh.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("new public key: %v", err)
	}
	return sshPub, priv
}

func writeAuthorizedKeys(t *testing.T, dir string, entries ...string) string {
	t.Helper()
	path := filepath.Join(dir, "authorized_keys")
	content := ""
	for _, e := range entries {
		content += e + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadAuthorizedKeys_ValidFile(t *testing.T) {
	pub, _ := generateTestKey(t)
	line := string(ssh.MarshalAuthorizedKey(pub))
	line = line[:len(line)-1] + " Alice <alice@example.com>\n"

	dir := t.TempDir()
	path := writeAuthorizedKeys(t, dir, line)

	ak, err := LoadAuthorizedKeys(path)
	if err != nil {
		t.Fatalf("LoadAuthorizedKeys: %v", err)
	}
	if ak.Len() != 1 {
		t.Fatalf("expected 1 entry, got %d", ak.Len())
	}
}

func TestLoadAuthorizedKeys_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeAuthorizedKeys(t, dir)

	ak, err := LoadAuthorizedKeys(path)
	if err != nil {
		t.Fatalf("LoadAuthorizedKeys: %v", err)
	}
	if ak.Len() != 0 {
		t.Errorf("expected 0 entries, got %d", ak.Len())
	}
}

func TestLoadAuthorizedKeys_MissingFile(t *testing.T) {
	_, err := LoadAuthorizedKeys("/nonexistent/authorized_keys")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLookup_Match(t *testing.T) {
	pub, _ := generateTestKey(t)
	line := string(ssh.MarshalAuthorizedKey(pub))
	line = line[:len(line)-1] + " Alice <alice@example.com>\n"

	dir := t.TempDir()
	path := writeAuthorizedKeys(t, dir, line)

	ak, _ := LoadAuthorizedKeys(path)
	identity, ok := ak.Lookup(pub)
	if !ok {
		t.Fatal("expected key to be found")
	}
	if identity.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", identity.Name)
	}
	if identity.Email != "alice@example.com" {
		t.Errorf("Email: got %q", identity.Email)
	}
}

func TestLookup_NoMatch(t *testing.T) {
	pub1, _ := generateTestKey(t)
	pub2, _ := generateTestKey(t)

	line := string(ssh.MarshalAuthorizedKey(pub1))
	line = line[:len(line)-1] + " Alice <alice@example.com>\n"

	dir := t.TempDir()
	path := writeAuthorizedKeys(t, dir, line)

	ak, _ := LoadAuthorizedKeys(path)
	_, ok := ak.Lookup(pub2)
	if ok {
		t.Error("expected key not to be found")
	}
}

func TestLookup_MultipleKeys(t *testing.T) {
	pub1, _ := generateTestKey(t)
	pub2, _ := generateTestKey(t)

	line1 := string(ssh.MarshalAuthorizedKey(pub1))
	line1 = line1[:len(line1)-1] + " Alice <alice@example.com>\n"
	line2 := string(ssh.MarshalAuthorizedKey(pub2))
	line2 = line2[:len(line2)-1] + " Bob <bob@example.com>\n"

	dir := t.TempDir()
	path := writeAuthorizedKeys(t, dir, line1, line2)

	ak, _ := LoadAuthorizedKeys(path)
	if ak.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", ak.Len())
	}

	id1, _ := ak.Lookup(pub1)
	if id1.Name != "Alice" {
		t.Errorf("expected Alice, got %q", id1.Name)
	}

	id2, _ := ak.Lookup(pub2)
	if id2.Name != "Bob" {
		t.Errorf("expected Bob, got %q", id2.Name)
	}
}
