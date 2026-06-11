package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
	tampered := token[:len(token)-1] + "X"

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
