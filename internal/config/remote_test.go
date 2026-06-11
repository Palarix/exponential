package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndGetServerCredential(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	err := SaveServerCredential("https://beats.example.com", "test-token")
	if err != nil {
		t.Fatalf("SaveServerCredential: %v", err)
	}

	cred, ok := GetServerCredential("https://beats.example.com")
	if !ok {
		t.Fatal("expected credential to be found")
	}
	if cred.Token != "test-token" {
		t.Errorf("Token: got %q, want test-token", cred.Token)
	}
	if cred.URL != "https://beats.example.com" {
		t.Errorf("URL: got %q", cred.URL)
	}

	// Verify file permissions
	info, _ := os.Stat(filepath.Join(dir, "beats", "credentials.yaml"))
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %o", info.Mode().Perm())
	}
}

func TestGetServerCredential_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	_, ok := GetServerCredential("https://unknown.example.com")
	if ok {
		t.Error("expected credential not to be found")
	}
}

func TestMultipleServers(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	SaveServerCredential("https://server-a.example.com", "token-a")
	SaveServerCredential("https://server-b.example.com:9090", "token-b")

	credA, ok := GetServerCredential("https://server-a.example.com")
	if !ok || credA.Token != "token-a" {
		t.Errorf("server-a: got %v, %v", credA, ok)
	}

	credB, ok := GetServerCredential("https://server-b.example.com:9090")
	if !ok || credB.Token != "token-b" {
		t.Errorf("server-b: got %v, %v", credB, ok)
	}
}

func TestRemoveServerCredential(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	SaveServerCredential("https://beats.example.com", "test-token")

	err := RemoveServerCredential("https://beats.example.com")
	if err != nil {
		t.Fatalf("RemoveServerCredential: %v", err)
	}

	_, ok := GetServerCredential("https://beats.example.com")
	if ok {
		t.Error("expected credential to be removed")
	}
}

func TestRemoveServerCredential_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	err := RemoveServerCredential("https://unknown.example.com")
	if err != nil {
		t.Fatalf("RemoveServerCredential on unknown: %v", err)
	}
}

func TestResolveRemote_FillsToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	SaveServerCredential("https://beats.example.com", "my-jwt")

	rc := ResolveRemote(RemoteConfig{URL: "https://beats.example.com"})
	if rc.Token != "my-jwt" {
		t.Errorf("expected token filled, got %q", rc.Token)
	}
}

func TestResolveRemote_EmptyURL(t *testing.T) {
	rc := ResolveRemote(RemoteConfig{})
	if rc.Token != "" {
		t.Errorf("expected empty token, got %q", rc.Token)
	}
}

func TestResolveRemote_ExistingTokenPreserved(t *testing.T) {
	rc := ResolveRemote(RemoteConfig{URL: "https://beats.example.com", Token: "existing"})
	if rc.Token != "existing" {
		t.Errorf("expected existing token preserved, got %q", rc.Token)
	}
}

func TestHostFromURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://beats.example.com", "beats.example.com"},
		{"http://localhost:8080", "localhost:8080"},
		{"https://beats.example.com:443/path", "beats.example.com:443"},
	}
	for _, tt := range tests {
		got := hostFromURL(tt.input)
		if got != tt.want {
			t.Errorf("hostFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
