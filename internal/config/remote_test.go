package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	info, _ := os.Stat(filepath.Join(dir, "beats", "user.yaml"))
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %o", info.Mode().Perm())
	}
}

func TestSaveServerCredential_PreservesLastRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	// Seed per-user state with a last_read value, then re-login (new token).
	c := LoadUserConfig()
	c.Servers["beats.example.com"] = ServerCredential{
		URL:      "https://beats.example.com",
		Token:    "old-token",
		LastRead: "2026-06-15T10:30:00Z",
	}
	if err := SaveUserConfig(c); err != nil {
		t.Fatalf("SaveUserConfig: %v", err)
	}

	if err := SaveServerCredential("https://beats.example.com", "new-token"); err != nil {
		t.Fatalf("SaveServerCredential: %v", err)
	}

	cred, ok := GetServerCredential("https://beats.example.com")
	if !ok {
		t.Fatal("expected credential to be found")
	}
	if cred.Token != "new-token" {
		t.Errorf("Token: got %q, want new-token", cred.Token)
	}
	if cred.LastRead != "2026-06-15T10:30:00Z" {
		t.Errorf("LastRead: got %q, want it preserved across re-login", cred.LastRead)
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

func TestReadRemoteURL_NoFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	got := ReadRemoteURL()
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestReadRemoteURL_WithRemote(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	os.WriteFile(filepath.Join(dir, ".beats", "config.yaml"), []byte("remote:\n  url: https://beats.example.com\n"), 0644)

	got := ReadRemoteURL()
	if got != "https://beats.example.com" {
		t.Errorf("expected https://beats.example.com, got %q", got)
	}
}

func TestReadRemoteURL_NoRemoteSection(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	os.WriteFile(filepath.Join(dir, ".beats", "config.yaml"), []byte("prefix: myapp-\n"), 0644)

	got := ReadRemoteURL()
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestIsLocalProjectConfig_True(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	os.WriteFile(filepath.Join(dir, ".beats", "config.yaml"), []byte("prefix: myapp-\nversion: 2\n"), 0644)

	if !IsLocalProjectConfig() {
		t.Error("expected true for config with prefix")
	}
}

func TestIsLocalProjectConfig_FalseVersionOnly(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	os.WriteFile(filepath.Join(dir, ".beats", "config.yaml"), []byte("version: 2\nremote:\n  url: https://beats.example.com\n"), 0644)

	if IsLocalProjectConfig() {
		t.Error("expected false for remote config that just has version")
	}
}

func TestIsLocalProjectConfig_FalseRemoteOnly(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	os.WriteFile(filepath.Join(dir, ".beats", "config.yaml"), []byte("remote:\n  url: https://beats.example.com\n"), 0644)

	if IsLocalProjectConfig() {
		t.Error("expected false for remote-only config")
	}
}

func TestIsLocalProjectConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if IsLocalProjectConfig() {
		t.Error("expected false when no config file")
	}
}

func TestSetRemoteURL_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := SetRemoteURL("https://beats.example.com"); err != nil {
		t.Fatalf("SetRemoteURL: %v", err)
	}

	got := ReadRemoteURL()
	if got != "https://beats.example.com" {
		t.Errorf("expected https://beats.example.com, got %q", got)
	}

	// Verify version field was written
	data, _ := os.ReadFile(filepath.Join(dir, ".beats", "config.yaml"))
	if !contains(string(data), "version: 2") {
		t.Errorf("expected version: 2 in config, got:\n%s", string(data))
	}
}

func TestSetRemoteURL_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	os.WriteFile(filepath.Join(dir, ".beats", "config.yaml"), []byte("some_key: some_value\n"), 0644)

	if err := SetRemoteURL("https://beats.example.com"); err != nil {
		t.Fatalf("SetRemoteURL: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".beats", "config.yaml"))
	content := string(data)
	if got := ReadRemoteURL(); got != "https://beats.example.com" {
		t.Errorf("remote URL: got %q", got)
	}
	if !contains(content, "some_key") {
		t.Error("existing content was not preserved")
	}
}

func TestSetRemoteURL_UpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	os.WriteFile(filepath.Join(dir, ".beats", "config.yaml"), []byte("remote:\n  url: https://old.example.com\n"), 0644)

	if err := SetRemoteURL("https://new.example.com"); err != nil {
		t.Fatalf("SetRemoteURL: %v", err)
	}

	got := ReadRemoteURL()
	if got != "https://new.example.com" {
		t.Errorf("expected https://new.example.com, got %q", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestInboxLastRead_Distributed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	url := "https://beats.example.com"

	// Unset cursor reads as zero time.
	if got := GetInboxLastRead(url); !got.IsZero() {
		t.Errorf("expected zero time for unset cursor, got %v", got)
	}

	want := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)
	if err := SetInboxLastRead(url, want); err != nil {
		t.Fatalf("SetInboxLastRead: %v", err)
	}

	if got := GetInboxLastRead(url); !got.Equal(want) {
		t.Errorf("GetInboxLastRead = %v, want %v", got, want)
	}

	// Cursor must live under the server entry alongside credentials.
	c := LoadUserConfig()
	if c.Servers[hostFromURL(url)].LastRead == "" {
		t.Error("expected last_read stored under servers entry")
	}
}

func TestInboxLastRead_DistributedPreservesToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))

	url := "https://beats.example.com"
	if err := SaveServerCredential(url, "tok"); err != nil {
		t.Fatalf("SaveServerCredential: %v", err)
	}
	if err := SetInboxLastRead(url, time.Now()); err != nil {
		t.Fatalf("SetInboxLastRead: %v", err)
	}

	cred, ok := GetServerCredential(url)
	if !ok || cred.Token != "tok" {
		t.Errorf("setting cursor wiped token: %+v ok=%v", cred, ok)
	}
}

func TestInboxLastRead_Local(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BEATS_CONFIG_DIR", filepath.Join(dir, "beats"))
	t.Chdir(dir)

	// Local mode: empty remote URL routes to the projects map.
	if got := GetInboxLastRead(""); !got.IsZero() {
		t.Errorf("expected zero time for unset local cursor, got %v", got)
	}

	want := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	if err := SetInboxLastRead("", want); err != nil {
		t.Fatalf("SetInboxLastRead: %v", err)
	}

	if got := GetInboxLastRead(""); !got.Equal(want) {
		t.Errorf("GetInboxLastRead = %v, want %v", got, want)
	}

	// Cursor must live under the projects map keyed by the absolute .beats path.
	c := LoadUserConfig()
	absBeats, _ := filepath.Abs(".beats")
	if c.Projects[absBeats].LastRead == "" {
		t.Errorf("expected last_read stored under projects[%q], got %+v", absBeats, c.Projects)
	}
	if len(c.Servers) != 0 {
		t.Errorf("local cursor should not touch servers map, got %+v", c.Servers)
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
