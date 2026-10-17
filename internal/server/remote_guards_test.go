package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	if code != http.StatusOK || body["remote"] != false {
		t.Fatalf("got %d %v, want 200 remote=false", code, body)
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

func TestLocalMode_ProxyBoardAnswersLocally(t *testing.T) {
	var proxied bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		w.WriteHeader(http.StatusTeapot)
	}))
	defer backend.Close()

	s := setupTestServer(t)
	s.ProxyURL = backend.URL
	code, body := localMode(t, s.SetupRoutes())
	if code != http.StatusOK || body["remote"] != true || proxied {
		t.Fatalf("got %d %v proxied=%v, want 200 remote=true answered locally", code, body, proxied)
	}
}

func TestLocalMode_NotOnHeadlessServer(t *testing.T) {
	s := setupTestServer(t)
	s.Headless = true
	if code, _ := localMode(t, s.SetupRoutes()); code == http.StatusOK {
		t.Fatalf("xpo serve must not answer /api/local/mode: %d", code)
	}
}

func TestHeadless_RefusesStartAndMerge(t *testing.T) {
	s := setupTestServer(t)
	s.Headless = true
	h := s.SetupRoutes()
	for _, path := range []string{"/api/issues/test-abc123/start", "/api/issues/test-abc123/merge"} {
		w := doJSON(t, h, "POST", path, `{}`)
		if w.Code != http.StatusNotImplemented {
			t.Errorf("%s: got %d, want 501", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), "your machine") {
			t.Errorf("%s: body %q does not explain why", path, w.Body.String())
		}
	}
}

func TestLocalBoard_StillStartsWork(t *testing.T) {
	s := setupTestServer(t)
	h := s.SetupRoutes()
	w := doJSON(t, h, "POST", "/api/issues/test-nope00/start", "")
	if w.Code == http.StatusNotImplemented {
		t.Fatalf("a local board must not refuse start: %d %s", w.Code, w.Body.String())
	}
}
