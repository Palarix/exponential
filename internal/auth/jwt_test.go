package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"
)

func generateSigningKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

func TestSignAndVerify_Roundtrip(t *testing.T) {
	pub, priv := generateSigningKey(t)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	}

	token, err := SignToken(priv, claims)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}

	got, err := VerifyToken(pub, token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}

	if got.Sub != claims.Sub {
		t.Errorf("Sub: got %q, want %q", got.Sub, claims.Sub)
	}
	if got.Exp != claims.Exp {
		t.Errorf("Exp: got %d, want %d", got.Exp, claims.Exp)
	}
}

func TestVerify_ExpiredToken(t *testing.T) {
	pub, priv := generateSigningKey(t)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(-time.Hour).Unix(),
		Iat: time.Now().Add(-2 * time.Hour).Unix(),
	}

	token, _ := SignToken(priv, claims)
	_, err := VerifyToken(pub, token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerify_BadSignature(t *testing.T) {
	pub, priv := generateSigningKey(t)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	}

	token, _ := SignToken(priv, claims)
	// Tamper with the payload section to change the claims
	parts := strings.SplitN(token, ".", 3)
	tampered := parts[0] + "." + "dGFtcGVyZWQ" + "." + parts[2]

	_, err := VerifyToken(pub, tampered)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	_, priv := generateSigningKey(t)
	otherPub, _ := generateSigningKey(t)

	claims := Claims{
		Sub: "Alice <alice@example.com>",
		Exp: time.Now().Add(time.Hour).Unix(),
		Iat: time.Now().Unix(),
	}

	token, _ := SignToken(priv, claims)
	_, err := VerifyToken(otherPub, token)
	if err == nil {
		t.Fatal("expected error for wrong key")
	}
}

func TestVerify_MalformedToken(t *testing.T) {
	pub, _ := generateSigningKey(t)

	cases := []string{
		"",
		"not.a.valid.token",
		"just-one-part",
		"two.parts",
	}

	for _, tc := range cases {
		_, err := VerifyToken(pub, tc)
		if err == nil {
			t.Errorf("expected error for %q", tc)
		}
	}
}
