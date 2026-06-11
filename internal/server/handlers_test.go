package server

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/palarix/beats/internal/auth"
	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"golang.org/x/crypto/ssh"
)

var testIDCounter atomic.Int64

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644)
	os.WriteFile(filepath.Join(beatsDir, "config.toml"), []byte{}, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		Version:          2,
	}

	return NewServer(cfg, 0, false, 0)
}

func seedIssue(t *testing.T, title string) string {
	t.Helper()
	id := fmt.Sprintf("test-%06x", testIDCounter.Add(1))
	evt := model.Event{
		ID:        id,
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: title, Labels: []string{"feature"}},
		CreatedBy: "Test User <test@test.com>",
		CreatedAt: time.Now(),
	}
	if err := storage.AppendEvent(evt); err != nil {
		t.Fatalf("seedIssue: %v", err)
	}
	return evt.ID
}

func TestHandleGetIssues_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var issues []IssueResponse
	json.NewDecoder(w.Body).Decode(&issues)
	if issues != nil && len(issues) != 0 {
		t.Errorf("expected empty list, got %d issues", len(issues))
	}
}

func TestHandleGetIssues_WithData(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "First Issue")
	seedIssue(t, "Second Issue")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var issues []IssueResponse
	json.NewDecoder(w.Body).Decode(&issues)
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestHandleGetIssue_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/test-nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetIssue_Found(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "My Issue")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/"+id, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var issue IssueResponse
	json.NewDecoder(w.Body).Decode(&issue)
	if issue.Title != "My Issue" {
		t.Errorf("expected title 'My Issue', got %q", issue.Title)
	}
}

func TestHandleDraft_Create(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "",
		"type":     "CREATE",
		"payload":  map[string]interface{}{"title": "New Issue", "labels": []string{"bug"}},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
	if resp["issue_id"] == "" {
		t.Error("expected non-empty issue_id")
	}
}

func TestHandleDraft_BadJSON(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDraft_UnknownType(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "test-123456",
		"type":     "INVALID",
		"payload":  map[string]interface{}{},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetPending_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/pending", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PendingResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.HasPending {
		t.Error("expected no pending events")
	}
}

func TestHandleDiscardPending(t *testing.T) {
	srv := setupTestServer(t)
	srv.AddPendingEvent(model.Event{ID: "test-aaaaaa", Type: model.EventTypeCreate})
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("DELETE", "/api/pending", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if srv.GetPendingCount() != 0 {
		t.Error("expected pending to be cleared")
	}
}

func TestHandleGetConfig(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["prefix"] != "test-" {
		t.Errorf("expected prefix 'test-', got %q", resp["prefix"])
	}
}

func TestHandleGetIssueHistory_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/test-000000/history", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleGetIssueHistory_WithEvents(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "History Test")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/"+id+"/history", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var history []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&history)
	if len(history) != 1 {
		t.Errorf("expected 1 history event, got %d", len(history))
	}
}

func TestHandleActivity_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/activity", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleMetrics(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func setupAuthServer(t *testing.T) (*Server, ssh.Signer) {
	t.Helper()
	s := setupTestServer(t)

	// Generate server signing key
	serverKey, err := auth.LoadOrGenerateServerKey(filepath.Join(".beats", "server.key"))
	if err != nil {
		t.Fatalf("server key: %v", err)
	}
	s.SigningKey = serverKey
	s.VerifyKey = serverKey.Public().(ed25519.PublicKey)
	s.NonceStore = auth.NewNonceStore(5 * time.Minute)

	// Generate a user SSH key
	_, userPriv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(userPriv)
	sshPub, _ := ssh.NewPublicKey(userPriv.Public())

	// Write authorized_keys
	line := string(ssh.MarshalAuthorizedKey(sshPub))
	line = line[:len(line)-1] + " Alice <alice@example.com>\n"
	ak, _ := auth.LoadAuthorizedKeys(writeAuthKeys(t, line))
	s.AuthorizedKeys = ak

	return s, signer
}

func writeAuthKeys(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(".beats", "authorized_keys")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write authorized_keys: %v", err)
	}
	return path
}

func TestAuthFlow_ChallengeVerifyAndAccess(t *testing.T) {
	s, signer := setupAuthServer(t)
	mux := s.SetupRoutes()

	// 1. Get challenge nonce
	req := httptest.NewRequest("POST", "/auth/challenge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("challenge: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var challengeResp struct {
		Nonce string `json:"nonce"`
	}
	json.NewDecoder(w.Body).Decode(&challengeResp)
	if challengeResp.Nonce == "" {
		t.Fatal("expected non-empty nonce")
	}

	// 2. Sign the nonce and verify
	sig, _ := signer.Sign(rand.Reader, []byte(challengeResp.Nonce))
	pubKeyBytes := signer.PublicKey().Marshal()

	verifyBody, _ := json.Marshal(map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(pubKeyBytes),
		"signature":  base64.StdEncoding.EncodeToString(ssh.Marshal(sig)),
		"nonce":      challengeResp.Nonce,
	})

	req = httptest.NewRequest("POST", "/auth/verify", bytes.NewReader(verifyBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var verifyResp struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	json.NewDecoder(w.Body).Decode(&verifyResp)
	if verifyResp.Token == "" {
		t.Fatal("expected non-empty token")
	}

	// 3. Use token to access a protected endpoint
	req = httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("Authorization", "Bearer "+verifyResp.Token)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("authenticated GET: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_ProtectedEndpointWithoutToken(t *testing.T) {
	s, _ := setupAuthServer(t)
	mux := s.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_HealthzAlwaysPublic(t *testing.T) {
	s, _ := setupAuthServer(t)
	mux := s.SetupRoutes()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthz: expected 200, got %d", w.Code)
	}
}

func TestHeadless_NoStaticAssets(t *testing.T) {
	s := setupTestServer(t)
	s.Headless = true
	mux := s.SetupRoutes()

	// API still works
	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("API: expected 200, got %d", w.Code)
	}

	// Root path returns 404 (no static assets)
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Root: expected 404 in headless mode, got %d", w.Code)
	}
}

func TestMCPHandler_Mounted(t *testing.T) {
	s := setupTestServer(t)
	called := false
	s.MCPHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mux := s.SetupRoutes()

	req := httptest.NewRequest("POST", "/mcp", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if !called {
		t.Error("expected MCP handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMCPHandler_AuthProtected(t *testing.T) {
	s, _ := setupAuthServer(t)
	s.MCPHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux := s.SetupRoutes()

	// Without token: 401
	req := httptest.NewRequest("POST", "/mcp", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}
}

func TestProxyMode_ForwardsAPIWithToken(t *testing.T) {
	var gotAuth string
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"test-123","title":"proxied"}]`))
	}))
	defer backend.Close()

	s := setupTestServer(t)
	s.ProxyURL = backend.URL
	s.ProxyToken = "my-jwt-token"
	mux := s.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotAuth != "Bearer my-jwt-token" {
		t.Errorf("expected bearer token forwarded, got %q", gotAuth)
	}
	if gotPath != "/api/issues" {
		t.Errorf("expected /api/issues forwarded, got %q", gotPath)
	}
	if !strings.Contains(w.Body.String(), "proxied") {
		t.Error("expected proxied response body")
	}
}

func TestProxyMode_BackendUnreachable(t *testing.T) {
	s := setupTestServer(t)
	s.ProxyURL = "http://127.0.0.1:1" // nothing listening
	s.ProxyToken = "tok"
	mux := s.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "remote server unreachable") {
		t.Errorf("expected error message, got %s", w.Body.String())
	}
}

func TestProxyMode_HealthzStillLocal(t *testing.T) {
	s := setupTestServer(t)
	s.ProxyURL = "http://127.0.0.1:1" // nothing listening
	mux := s.SetupRoutes()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProxyMode_AuthEndpointsForwarded(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"nonce":"abc123"}`))
	}))
	defer backend.Close()

	s := setupTestServer(t)
	s.ProxyURL = backend.URL
	mux := s.SetupRoutes()

	req := httptest.NewRequest("POST", "/auth/challenge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotPath != "/auth/challenge" {
		t.Errorf("expected /auth/challenge forwarded, got %q", gotPath)
	}
}

func TestAuth_NonceReplayPrevented(t *testing.T) {
	s, signer := setupAuthServer(t)
	mux := s.SetupRoutes()

	// Get nonce
	req := httptest.NewRequest("POST", "/auth/challenge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp struct{ Nonce string }
	json.NewDecoder(w.Body).Decode(&resp)

	// First verify succeeds
	sig, _ := signer.Sign(rand.Reader, []byte(resp.Nonce))
	body, _ := json.Marshal(map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal()),
		"signature":  base64.StdEncoding.EncodeToString(ssh.Marshal(sig)),
		"nonce":      resp.Nonce,
	})

	req = httptest.NewRequest("POST", "/auth/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first verify: expected 200, got %d", w.Code)
	}

	// Second verify with same nonce fails (replay)
	req = httptest.NewRequest("POST", "/auth/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("replay: expected 401, got %d", w.Code)
	}
}
