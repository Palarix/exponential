package auth

import (
	"crypto/ed25519"
	"fmt"
	"net/http"
	"strings"

	"github.com/palarix/beats/internal/config"
)

// RequireAuthHandler wraps an http.Handler with Bearer JWT verification.
func RequireAuthHandler(pubKey ed25519.PublicKey, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RequireAuth(pubKey, func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})(w, r)
	})
}

// RequireAuth returns middleware that verifies a Bearer JWT token and
// injects the authenticated UserIdentity into the request context.
func RequireAuth(pubKey ed25519.PublicKey, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"invalid authorization scheme"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := VerifyToken(pubKey, tokenStr)
		if err != nil {
			msg := "invalid or expired token"
			if err.Error() == "token expired" {
				msg = "token expired"
			}
			http.Error(w, `{"error":"`+msg+`"}`, http.StatusUnauthorized)
			return
		}

		user := ParseIdentity(claims.Sub)
		ctx := WithUser(r.Context(), user)
		next(w, r.WithContext(ctx))
	}
}

// RequireCapability wraps a handler with both JWT auth and a capability
// check against the permissions config. When permissions are not configured
// it falls through to RequireAuth only.
func RequireCapability(pubKey ed25519.PublicKey, perms config.PermissionsConfig, capability string, next http.HandlerFunc) http.HandlerFunc {
	return RequireAuth(pubKey, func(w http.ResponseWriter, r *http.Request) {
		if perms.Enabled() {
			user, _ := UserFromContext(r.Context())
			if !perms.HasCapability(user.Email, capability) {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden: requires %s capability"}`, capability), http.StatusForbidden)
				return
			}
		}
		next(w, r)
	})
}
