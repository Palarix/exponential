package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	origOut := log.Writer()
	origFlags := log.Flags()
	log.SetFlags(0)
	log.SetOutput(&buf)
	defer func() {
		log.SetFlags(origFlags)
		log.SetOutput(origOut)
	}()
	fn()
	return buf.String()
}

func fakeJWT(sub string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA"}`))
	claims, _ := json.Marshal(map[string]string{"sub": sub})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	sig := base64.RawURLEncoding.EncodeToString([]byte("fakesig"))
	return header + "." + payload + "." + sig
}

func TestRequestLogger_Anonymous(t *testing.T) {
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	output := captureLog(t, func() {
		handler.ServeHTTP(rec, req)
	})

	parts := strings.Split(strings.TrimSpace(output), ",")
	if len(parts) != 5 {
		t.Fatalf("expected 5 fields, got %d: %q", len(parts), output)
	}
	if parts[0] != "GET" {
		t.Errorf("method: got %q, want GET", parts[0])
	}
	if parts[1] != "/healthz" {
		t.Errorf("path: got %q, want /healthz", parts[1])
	}
	if parts[2] != "200" {
		t.Errorf("status: got %q, want 200", parts[2])
	}
	if parts[4] != "anonymous" {
		t.Errorf("client: got %q, want anonymous", parts[4])
	}
}

func TestRequestLogger_Authenticated(t *testing.T) {
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest("POST", "/api/issues", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("Alice <alice@example.com>"))
	rec := httptest.NewRecorder()

	output := captureLog(t, func() {
		handler.ServeHTTP(rec, req)
	})

	parts := strings.Split(strings.TrimSpace(output), ",")
	if len(parts) != 5 {
		t.Fatalf("expected 5 fields, got %d: %q", len(parts), output)
	}
	if parts[2] != "201" {
		t.Errorf("status: got %q, want 201", parts[2])
	}
	if parts[4] != "alice" {
		t.Errorf("client: got %q, want alice", parts[4])
	}
}

func TestRequestLogger_DefaultStatus(t *testing.T) {
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	output := captureLog(t, func() {
		handler.ServeHTTP(rec, req)
	})

	parts := strings.Split(strings.TrimSpace(output), ",")
	if parts[2] != "200" {
		t.Errorf("status: got %q, want 200 (implicit)", parts[2])
	}
}

func TestClientFromRequest_NoHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := clientFromRequest(req); got != "anonymous" {
		t.Errorf("got %q, want anonymous", got)
	}
}

func TestClientFromRequest_ValidJWT(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("Bob <bob@example.com>"))
	if got := clientFromRequest(req); got != "bob" {
		t.Errorf("got %q, want bob", got)
	}
}

func TestClientFromRequest_MalformedToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	if got := clientFromRequest(req); got != "anonymous" {
		t.Errorf("got %q, want anonymous", got)
	}
}

func TestConfigureLogging(t *testing.T) {
	origOut := log.Writer()
	origFlags := log.Flags()
	defer func() {
		log.SetFlags(origFlags)
		log.SetOutput(origOut)
	}()

	ConfigureLogging()

	if log.Flags() != 0 {
		t.Errorf("expected flags 0, got %d", log.Flags())
	}
}
