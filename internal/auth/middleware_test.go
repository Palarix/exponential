package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/palarix/beats/internal/config"
)

func TestRequireAuthHandler_ValidToken(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	}
	token, _ := SignToken(priv, claims)

	var gotUser UserIdentity
	var gotOK bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotOK = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAuthHandler(pub, inner)

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !gotOK {
		t.Fatal("expected user in context")
	}
	if gotUser.Name != "Alice" {
		t.Errorf("expected Alice, got %q", gotUser.Name)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	}
	token, _ := SignToken(priv, claims)

	var gotUser UserIdentity
	var gotOK bool
	handler := RequireAuth(pub, func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotOK = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !gotOK {
		t.Fatal("expected user in context")
	}
	if gotUser.Name != "Alice" {
		t.Errorf("expected Alice, got %q", gotUser.Name)
	}
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)

	handler := RequireAuth(pub, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest("GET", "/api/issues", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_InvalidScheme(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)

	handler := RequireAuth(pub, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(-time.Hour).Unix(),
		Iat: time.Now().Add(-2 * time.Hour).Unix(),
	}
	token, _ := SignToken(priv, claims)

	handler := RequireAuth(pub, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_TamperedToken(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	}
	token, _ := SignToken(priv, claims)
	parts := strings.SplitN(token, ".", 3)
	tampered := parts[0] + "." + "dGFtcGVyZWQ" + "." + parts[2]

	handler := RequireAuth(pub, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("Authorization", "Bearer "+tampered)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func testPerms() config.PermissionsConfig {
	return config.PermissionsConfig{
		Roles: map[string][]string{
			"admin":  {"*"},
			"viewer": {"issue.read"},
		},
		Users: map[string]string{
			"alice@example.com": "admin",
			"bob@example.com":   "viewer",
		},
	}
}

func TestRequireCapability_Allowed(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	token, _ := SignToken(priv, Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	})

	called := false
	handler := RequireCapability(pub, testPerms(), "issue.delete", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("DELETE", "/api/issues/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Error("handler should have been called for admin")
	}
}

func TestRequireCapability_Denied(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	token, _ := SignToken(priv, Claims{
		Sub: "Bob <bob@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	})

	handler := RequireCapability(pub, testPerms(), "issue.delete", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for viewer deleting")
	})

	req := httptest.NewRequest("DELETE", "/api/issues/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestRequireCapability_NoPermsConfigured(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	token, _ := SignToken(priv, Claims{
		Sub: "Anyone <anyone@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	})

	called := false
	handler := RequireCapability(pub, config.PermissionsConfig{}, "issue.delete", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("DELETE", "/api/issues/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Errorf("expected open access when no perms configured, got %d", rec.Code)
	}
}
