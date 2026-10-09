package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"html/template"
	"net/http"
	"strings"
)

// GenerateAccessToken returns a fresh 256-bit token, hex-encoded. The board
// requires it when bound to a non-loopback address.
func GenerateAccessToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// AccessURL is the URL to open in a browser: URL() plus the access token,
// when one is required.
func (s *Server) AccessURL() string {
	if s.AccessToken == "" {
		return s.URL()
	}
	return s.URL() + "/?token=" + s.AccessToken
}

// cookieName is unique per run, derived from the token without revealing
// it. Cookies are scoped by host, not port, and ports go to whichever board
// starts first, so neither a fixed name nor the port can keep concurrent
// boards on one host from overwriting each other's session.
func (s *Server) cookieName() string {
	sum := sha256.Sum256([]byte(s.AccessToken))
	return "exponential-" + hex.EncodeToString(sum[:])[:8]
}

func (s *Server) validToken(candidate string) bool {
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(s.AccessToken)) == 1
}

// setSessionCookie stores the token in an HttpOnly cookie. SameSite=Lax so
// following a link to the board from another site keeps the session, while
// cross-site writes (all non-GET) still go without it.
func (s *Server) setSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName(),
		Value:    s.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// tokenGuard requires the per-run access token, carried in a session cookie,
// on every request except /healthz and the /auth page. A GET with a valid
// ?token= is exchanged for the cookie and redirected with the token removed.
// Without a valid cookie, API and non-GET requests get 401 and page loads
// are redirected to /auth. A no-op when s.AccessToken is empty (loopback).
func (s *Server) tokenGuard(next http.Handler) http.Handler {
	if s.AccessToken == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz":
			next.ServeHTTP(w, r)
			return
		case r.URL.Path == "/auth" && r.Method == http.MethodGet:
			renderAuthPage(w, http.StatusOK, "")
			return
		case r.URL.Path == "/auth" && r.Method == http.MethodPost:
			s.handleAuthSubmit(w, r)
			return
		}

		if c, err := r.Cookie(s.cookieName()); err == nil && s.validToken(c.Value) {
			next.ServeHTTP(w, r)
			return
		}

		q := r.URL.Query()
		if r.Method == http.MethodGet && q.Has("token") && s.validToken(q.Get("token")) {
			s.setSessionCookie(w)
			q.Del("token")
			u := *r.URL
			u.RawQuery = q.Encode()
			http.Redirect(w, r, u.RequestURI(), http.StatusFound)
			return
		}

		if r.Method != http.MethodGet || r.URL.Path == "/mcp" || strings.HasPrefix(r.URL.Path, "/api/") {
			respondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		http.Redirect(w, r, "/auth", http.StatusFound)
	})
}

func (s *Server) handleAuthSubmit(w http.ResponseWriter, r *http.Request) {
	if !s.validToken(r.PostFormValue("token")) {
		renderAuthPage(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	s.setSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var authPage = template.Must(template.New("auth").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>xpo board · sign in</title>
<style>
  :root { color-scheme: light dark; }
  body { margin: 0; min-height: 100vh; display: grid; place-items: center;
         font: 14px/1.5 system-ui, sans-serif; background: Canvas; color: CanvasText; }
  form { width: min(360px, calc(100vw - 32px)); display: grid; gap: 12px; }
  h1 { font-size: 18px; margin: 0; }
  p { margin: 0; opacity: .75; }
  input, button { font: inherit; padding: 8px 10px; border-radius: 6px; }
  input { border: 1px solid GrayText; background: Field; color: FieldText; }
  button { border: 0; background: #2563eb; color: #fff; cursor: pointer; }
  .error { color: #dc2626; opacity: 1; }
</style>
</head>
<body>
<form method="post" action="/auth">
  <h1>xpo board</h1>
  <p>Paste the access token printed in the terminal that started <code>xpo board</code>.</p>
  {{if .}}<p class="error" role="alert">{{.}}</p>{{end}}
  <input type="password" name="token" aria-label="Access token" placeholder="Access token" autocomplete="off" autofocus required>
  <button type="submit">Sign in</button>
</form>
</body>
</html>
`))

func renderAuthPage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = authPage.Execute(w, message)
}
