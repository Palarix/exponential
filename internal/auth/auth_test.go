package auth

import (
	"context"
	"testing"
)

func TestParseIdentity_NameEmail(t *testing.T) {
	id := ParseIdentity("Alice <alice@example.com>")
	if id.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", id.Name)
	}
	if id.Email != "alice@example.com" {
		t.Errorf("Email: got %q, want alice@example.com", id.Email)
	}
	if id.Raw != "Alice <alice@example.com>" {
		t.Errorf("Raw: got %q", id.Raw)
	}
}

func TestParseIdentity_EmailOnly(t *testing.T) {
	id := ParseIdentity("alice@example.com")
	if id.Email != "alice@example.com" {
		t.Errorf("Email: got %q", id.Email)
	}
}

func TestParseIdentity_NameOnly(t *testing.T) {
	id := ParseIdentity("alice")
	if id.Name != "alice" {
		t.Errorf("Name: got %q", id.Name)
	}
	if id.Email != "" {
		t.Errorf("Email: got %q, want empty", id.Email)
	}
}

func TestWithUser_UserFromContext(t *testing.T) {
	user := UserIdentity{Name: "Alice", Email: "alice@example.com", Raw: "Alice <alice@example.com>"}
	ctx := WithUser(context.Background(), user)
	got, ok := UserFromContext(ctx)
	if !ok {
		t.Fatal("expected user in context")
	}
	if got.Raw != user.Raw {
		t.Errorf("got %q, want %q", got.Raw, user.Raw)
	}
}

func TestUserFromContext_Missing(t *testing.T) {
	_, ok := UserFromContext(context.Background())
	if ok {
		t.Fatal("expected no user in empty context")
	}
}
