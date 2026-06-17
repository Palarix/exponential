package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupTestServerWithConfig(t *testing.T) *Server {
	t.Helper()
	srv := setupTestServer(t)
	configPath := filepath.Join(".xpo", "config.yaml")
	os.WriteFile(configPath, []byte("prefix: test-\nversion: 2\nlabels:\n  bug: \"#ff0000\"\n"), 0644)
	return srv
}

func TestHandleAddLabel(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]string{"name": "feature", "color": "#00ff00"})
	req := httptest.NewRequest("POST", "/api/config/labels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if srv.Config.Labels["feature"] != "#00ff00" {
		t.Errorf("label not added to config: %v", srv.Config.Labels)
	}
}

func TestHandleAddLabel_MissingFields(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]string{"name": "", "color": ""})
	req := httptest.NewRequest("POST", "/api/config/labels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleDeleteLabel(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	srv.Config.Labels = map[string]string{"bug": "#ff0000", "feature": "#00ff00"}
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]string{"name": "bug"})
	req := httptest.NewRequest("DELETE", "/api/config/labels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, exists := srv.Config.Labels["bug"]; exists {
		t.Error("bug label should be removed from config")
	}
}

func TestHandleGetUser(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/user", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["name"] == "" {
		t.Error("expected non-empty name")
	}
}
