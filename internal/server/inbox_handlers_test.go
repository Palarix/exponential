package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleInboxStatus(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/inbox/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		LastRead string `json:"last_read"`
		Unread   int    `json:"unread"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Unread < 0 {
		t.Errorf("unread should be ≥0, got %d", resp.Unread)
	}
}

func TestHandleInboxRead(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("POST", "/api/inbox/read", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		LastRead string `json:"last_read"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.LastRead == "" {
		t.Error("expected non-empty last_read timestamp")
	}
}

func TestHandleInboxRead_ThenStatus(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	// Mark as read
	req := httptest.NewRequest("POST", "/api/inbox/read", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Now check status — unread should be 0
	req = httptest.NewRequest("GET", "/api/inbox/status", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp struct {
		Unread int `json:"unread"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Unread != 0 {
		t.Errorf("after marking read, unread = %d, want 0", resp.Unread)
	}
}

func TestHandleUpdateLabel(t *testing.T) {
	srv := setupTestServerWithConfig(t)
	srv.Config.Labels = map[string]string{"bug": "#ff0000"}
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]string{"old_name": "bug", "new_name": "defect", "color": "#ee0000"})
	req := httptest.NewRequest("PUT", "/api/config/labels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, exists := srv.Config.Labels["bug"]; exists {
		t.Error("old label should be removed")
	}
	if srv.Config.Labels["defect"] != "#ee0000" {
		t.Errorf("new label not set: %v", srv.Config.Labels)
	}
}

func TestHandleGetCycles_Disabled(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/cycles", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Enabled bool `json:"enabled"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Enabled {
		t.Error("cycles should be disabled by default")
	}
}
