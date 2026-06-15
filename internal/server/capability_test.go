package server

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/palarix/beats/internal/auth"
	"github.com/palarix/beats/internal/config"
)

func setupCapServer(t *testing.T, perms config.PermissionsConfig) (*Server, ed25519.PrivateKey) {
	t.Helper()
	srv := setupTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	srv.SigningKey = priv
	srv.VerifyKey = pub
	srv.Config.Permissions = perms
	return srv, priv
}

func tokenFor(priv ed25519.PrivateKey, identity string) string {
	tok, _ := auth.SignToken(priv, auth.Claims{
		Sub: identity,
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	})
	return tok
}

func TestCapability_AdminCanWrite(t *testing.T) {
	perms := config.PermissionsConfig{
		Roles: map[string][]string{"admin": {"*"}},
		Users: map[string]string{"alice@test.com": "admin"},
	}
	srv, priv := setupCapServer(t, perms)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "", "type": "CREATE",
		"payload": map[string]interface{}{"title": "test"},
	})
	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(priv, "Alice <alice@test.com>"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin should be allowed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCapability_ViewerBlockedFromWrite(t *testing.T) {
	perms := config.PermissionsConfig{
		Roles: map[string][]string{
			"admin":  {"*"},
			"viewer": {"issue.read"},
		},
		Users: map[string]string{
			"alice@test.com": "admin",
			"bob@test.com":   "viewer",
		},
	}
	srv, priv := setupCapServer(t, perms)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "", "type": "CREATE",
		"payload": map[string]interface{}{"title": "blocked"},
	})
	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(priv, "Bob <bob@test.com>"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("viewer should be blocked from write, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCapability_ViewerCanRead(t *testing.T) {
	perms := config.PermissionsConfig{
		Roles: map[string][]string{"viewer": {"issue.read"}},
		Users: map[string]string{"bob@test.com": "viewer"},
	}
	srv, priv := setupCapServer(t, perms)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(priv, "Bob <bob@test.com>"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("viewer should read issues, got %d", w.Code)
	}
}

func TestCapability_NoPermsAllowsEveryone(t *testing.T) {
	srv, priv := setupCapServer(t, config.PermissionsConfig{})
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "", "type": "CREATE",
		"payload": map[string]interface{}{"title": "open"},
	})
	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(priv, "Anyone <any@test.com>"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("no perms should allow everyone, got %d", w.Code)
	}
}

func TestCapability_UnmappedUserDenied(t *testing.T) {
	perms := config.PermissionsConfig{
		Roles: map[string][]string{"admin": {"*"}},
		Users: map[string]string{"alice@test.com": "admin"},
	}
	srv, priv := setupCapServer(t, perms)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(priv, "Unknown <unknown@test.com>"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("unmapped user should be denied, got %d", w.Code)
	}
}
