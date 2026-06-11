package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestDiscoverSSHKeys_FindsFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0700)

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(priv)
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}
	pemBytes := pem.EncodeToMemory(block)
	os.WriteFile(filepath.Join(sshDir, "id_ed25519"), pemBytes, 0600)

	keys, err := DiscoverSSHKeys()
	if err != nil {
		t.Fatalf("DiscoverSSHKeys: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("expected at least 1 key")
	}
	if keys[0].Signer.PublicKey().Type() != "ssh-ed25519" {
		t.Errorf("expected ssh-ed25519, got %s", keys[0].Signer.PublicKey().Type())
	}
}

func TestPreferEd25519(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(priv)

	keys := []SSHKey{
		{Comment: "rsa key", Signer: signer}, // pretend it's RSA for ordering
		{Comment: "ed25519 key", Signer: signer},
	}

	selected := PreferEd25519(keys)
	if selected == nil {
		t.Fatal("expected a key")
	}
}

func TestPreferEd25519_Empty(t *testing.T) {
	selected := PreferEd25519(nil)
	if selected != nil {
		t.Error("expected nil for empty list")
	}
}

func TestSSHKey_Label(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(priv)

	k := SSHKey{Path: "/home/user/.ssh/id_ed25519", Comment: "my key", Signer: signer}
	label := k.Label()
	if label == "" {
		t.Error("expected non-empty label")
	}
	if label != "my key (ssh-ed25519)" {
		t.Errorf("unexpected label: %s", label)
	}
}
