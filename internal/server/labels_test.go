package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func labelRequest(t *testing.T, srv *Server, method string, payload map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(method, "/api/config/labels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.SetupRoutes().ServeHTTP(w, req)
	return w
}

func TestHandleUpdateLabel_RejectsBuiltinChanges(t *testing.T) {
	cases := map[string]map[string]string{
		"rename":            {"old_name": "bug", "new_name": "defect", "color": "#ff0000"},
		"case-only rename":  {"old_name": "bug", "new_name": "Bug", "color": "#ff0000"},
		"color-only change": {"old_name": "bug", "new_name": "bug", "color": "#00ff00"},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			srv := setupTestServerWithConfig(t)
			srv.Config.DefaultLabels = []string{"bug", "feature"}
			srv.Config.Labels = map[string]string{"bug": "#ff0000"}

			w := labelRequest(t, srv, "PUT", payload)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
			if srv.Config.Labels["bug"] != "#ff0000" {
				t.Errorf("built-in label changed: %v", srv.Config.Labels)
			}
			data, _ := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
			if !strings.Contains(string(data), "bug: \"#ff0000\"") {
				t.Errorf("config changed: %s", data)
			}
		})
	}
}

func TestHandleUpdateLabel_RejectsRenameOntoBuiltin(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	srv.Config.DefaultLabels = []string{"Bug"}

	w := labelRequest(t, srv, "PUT", map[string]string{"old_name": "ux", "new_name": "bug", "color": "#00ff00"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAddLabel_RejectsBuiltin(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	srv.Config.DefaultLabels = []string{"bug"}

	w := labelRequest(t, srv, "POST", map[string]string{"name": "Bug", "color": "#00ff00"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleUpdateLabel_AllowsCustomRename(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	srv.Config.DefaultLabels = []string{"feature"}

	w := labelRequest(t, srv, "PUT", map[string]string{"old_name": "bug", "new_name": "defect", "color": "#00ff00"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if srv.Config.Labels["defect"] != "#00ff00" {
		t.Errorf("label not renamed: %v", srv.Config.Labels)
	}
}

func TestHandleDeleteLabel_RejectsBuiltin(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	srv.Config.DefaultLabels = []string{"Bug"}

	w := labelRequest(t, srv, "DELETE", map[string]string{"name": "bug"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
