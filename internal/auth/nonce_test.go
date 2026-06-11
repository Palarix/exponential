package auth

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestNonceGenerate(t *testing.T) {
	ns := NewNonceStore(5 * time.Minute)
	nonce, err := ns.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if nonce == "" {
		t.Fatal("expected non-empty nonce")
	}
	decoded, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		t.Fatalf("nonce is not valid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(decoded))
	}
}

func TestNonceGenerate_Unique(t *testing.T) {
	ns := NewNonceStore(5 * time.Minute)
	a, _ := ns.Generate()
	b, _ := ns.Generate()
	if a == b {
		t.Error("expected unique nonces")
	}
}

func TestNonceValidate_Valid(t *testing.T) {
	ns := NewNonceStore(5 * time.Minute)
	nonce, _ := ns.Generate()
	if !ns.Validate(nonce) {
		t.Error("expected valid nonce")
	}
}

func TestNonceValidate_Unknown(t *testing.T) {
	ns := NewNonceStore(5 * time.Minute)
	if ns.Validate("not-a-real-nonce") {
		t.Error("expected unknown nonce to be invalid")
	}
}

func TestNonceValidate_SingleUse(t *testing.T) {
	ns := NewNonceStore(5 * time.Minute)
	nonce, _ := ns.Generate()
	if !ns.Validate(nonce) {
		t.Fatal("first validate should succeed")
	}
	if ns.Validate(nonce) {
		t.Error("second validate should fail (single-use)")
	}
}

func TestNonceValidate_Expired(t *testing.T) {
	ns := NewNonceStore(1 * time.Millisecond)
	nonce, _ := ns.Generate()
	time.Sleep(5 * time.Millisecond)
	if ns.Validate(nonce) {
		t.Error("expected expired nonce to be invalid")
	}
}
