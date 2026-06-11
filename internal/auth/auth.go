package auth

import (
	"context"
	"strings"
)

// UserIdentity represents an authenticated user extracted from the
// authorized_keys comment field.
type UserIdentity struct {
	Name  string
	Email string
	Raw   string // "Name <email>" — matches the CreatedBy format used in events
}

type contextKey struct{}

var userKey = contextKey{}

// WithUser returns a new context carrying the given user identity.
func WithUser(ctx context.Context, u UserIdentity) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext extracts the user identity from the context.
func UserFromContext(ctx context.Context) (UserIdentity, bool) {
	u, ok := ctx.Value(userKey).(UserIdentity)
	return u, ok
}

// ParseIdentity parses a "Name <email>" string into a UserIdentity.
// If the string doesn't match that format, the entire string is used as Name.
func ParseIdentity(raw string) UserIdentity {
	raw = strings.TrimSpace(raw)
	if idx := strings.LastIndex(raw, " <"); idx != -1 && strings.HasSuffix(raw, ">") {
		name := raw[:idx]
		email := raw[idx+2 : len(raw)-1]
		return UserIdentity{Name: name, Email: email, Raw: raw}
	}
	if strings.Contains(raw, "@") {
		return UserIdentity{Name: raw, Email: raw, Raw: raw + " <" + raw + ">"}
	}
	return UserIdentity{Name: raw, Raw: raw}
}
