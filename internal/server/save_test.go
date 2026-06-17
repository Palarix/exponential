package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func TestHandleSave_NoPending(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]string{"message": ""})
	req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "no_changes" {
		t.Errorf("expected no_changes, got %v", resp["status"])
	}
}

func TestHandleSave_PersistsPending(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	srv.AddPendingEvent(model.Event{
		ID: "test-aaa", Type: model.EventTypeCreate,
		Payload: model.CreatePayload{Title: "Pending"},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})

	body, _ := json.Marshal(map[string]string{"message": "test save"})
	req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "saved" {
		t.Errorf("expected saved, got %v", resp["status"])
	}
	if resp["count"].(float64) != 1 {
		t.Errorf("expected count=1, got %v", resp["count"])
	}

	events, _ := storage.ReadEvents()
	found := false
	for _, e := range events {
		if e.ID == "test-aaa" {
			found = true
		}
	}
	if !found {
		t.Error("pending event not persisted to disk")
	}

	if srv.GetPendingCount() != 0 {
		t.Error("pending events should be cleared after save")
	}
}
