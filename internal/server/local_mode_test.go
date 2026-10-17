package server

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func localMode(t *testing.T, h http.Handler) (int, map[string]any) {
	t.Helper()
	w := doJSON(t, h, "GET", "/api/local/mode", "")
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

func TestLocalMode_LocalBoard(t *testing.T) {
	s := setupTestServer(t)
	code, body := localMode(t, s.SetupRoutes())
	if _, ok := body["remote"]; code != http.StatusOK || ok {
		t.Fatalf("got %d %v, want 200 without remote", code, body)
	}
}

func TestLocalMode_ReportsGitRepo(t *testing.T) {
	s := setupTestServer(t)
	h := s.SetupRoutes()
	if _, body := localMode(t, h); body["git"] != false || body["root"] == "" || body["root"] == nil {
		t.Fatalf("without .git: %v, want git=false and a root", body)
	}
	os.Mkdir(".git", 0o755)
	if _, body := localMode(t, h); body["git"] != true {
		t.Fatalf("with .git: %v, want git=true", body)
	}
}
