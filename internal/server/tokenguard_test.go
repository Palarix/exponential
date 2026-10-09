package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

const testToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// testCookie is the session cookie name for testToken: exponential- plus the
// first 8 hex chars of sha256(token).
var testCookie = func() string {
	sum := sha256.Sum256([]byte(testToken))
	return "exponential-" + hex.EncodeToString(sum[:])[:8]
}()

func guarded(token string) http.Handler {
	s := &Server{AccessToken: token}
	return s.tokenGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func withCookie(req *http.Request, value string) *http.Request {
	req.AddCookie(&http.Cookie{Name: testCookie, Value: value})
	return req
}

func TestGenerateAccessToken(t *testing.T) {
	a, err := GenerateAccessToken()
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	b, _ := GenerateAccessToken()
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(a) {
		t.Fatalf("token %q is not 64 hex chars", a)
	}
	if a == b {
		t.Fatal("two generated tokens are equal")
	}
}

func TestTokenGuardDisabledWithoutToken(t *testing.T) {
	for _, path := range []string{"/", "/api/issues", "/board"} {
		w := httptest.NewRecorder()
		guarded("").ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s without token configured: got %d, want 200", path, w.Code)
		}
	}
}

func TestTokenGuardWithoutCookie(t *testing.T) {
	cases := []struct {
		method, path string
		want         int
		location     string
	}{
		{"GET", "/healthz", http.StatusOK, ""},
		{"GET", "/api/issues", http.StatusUnauthorized, ""},
		{"GET", "/api/events", http.StatusUnauthorized, ""},
		{"POST", "/api/save", http.StatusUnauthorized, ""},
		{"POST", "/mcp", http.StatusUnauthorized, ""},
		{"GET", "/", http.StatusFound, "/auth"},
		{"GET", "/board/xpo-1", http.StatusFound, "/auth"},
		{"GET", "/assets/index.js", http.StatusFound, "/auth"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		guarded(testToken).ServeHTTP(w, httptest.NewRequest(c.method, c.path, nil))
		if w.Code != c.want {
			t.Errorf("%s %s: got %d, want %d", c.method, c.path, w.Code, c.want)
		}
		if loc := w.Header().Get("Location"); loc != c.location {
			t.Errorf("%s %s: Location %q, want %q", c.method, c.path, loc, c.location)
		}
		if c.want == http.StatusUnauthorized && !strings.Contains(w.Body.String(), `"unauthorized"`) {
			t.Errorf("%s %s: body %q, want JSON unauthorized error", c.method, c.path, w.Body.String())
		}
	}
}

func TestTokenGuardCookie(t *testing.T) {
	w := httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, withCookie(httptest.NewRequest("GET", "/api/issues", nil), testToken))
	if w.Code != http.StatusOK {
		t.Errorf("valid cookie: got %d, want 200", w.Code)
	}

	w = httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, withCookie(httptest.NewRequest("GET", "/api/issues", nil), "wrong"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong cookie: got %d, want 401", w.Code)
	}

	// The token under another run's cookie name doesn't count.
	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.AddCookie(&http.Cookie{Name: "exponential-00000000", Value: testToken})
	w = httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("other port's cookie: got %d, want 401", w.Code)
	}
}

func sessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == testCookie {
			return c
		}
	}
	return nil
}

func TestTokenGuardQueryExchange(t *testing.T) {
	w := httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, httptest.NewRequest("GET", "/board?view=list&token="+testToken, nil))
	if w.Code != http.StatusFound {
		t.Fatalf("exchange: got %d, want 302", w.Code)
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("bad Location: %v", err)
	}
	if loc.Path != "/board" || loc.Query().Get("view") != "list" || loc.Query().Has("token") {
		t.Errorf("Location %q: want /board?view=list without token", loc)
	}
	c := sessionCookie(t, w)
	if c == nil {
		t.Fatal("exchange set no session cookie")
	}
	if c.Value != testToken || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Errorf("cookie %+v: want value=token, HttpOnly, SameSite=Lax, Path=/", c)
	}

	w = httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, httptest.NewRequest("GET", "/?token=wrong", nil))
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/auth" {
		t.Errorf("wrong query token: got %d → %q, want 302 → /auth", w.Code, w.Header().Get("Location"))
	}
	if sessionCookie(t, w) != nil {
		t.Error("wrong query token set a session cookie")
	}
}

func TestAuthPage(t *testing.T) {
	w := httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, httptest.NewRequest("GET", "/auth", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /auth: got %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("GET /auth Content-Type %q, want text/html", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, `<form method="post" action="/auth"`) || !strings.Contains(body, `name="token"`) {
		t.Errorf("GET /auth body lacks the token form:\n%s", body)
	}
}

func postAuth(token string) *http.Request {
	req := httptest.NewRequest("POST", "/auth", strings.NewReader(url.Values{"token": {token}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestAuthSubmit(t *testing.T) {
	w := httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, postAuth(testToken))
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/" {
		t.Errorf("valid token: got %d → %q, want 303 → /", w.Code, w.Header().Get("Location"))
	}
	if c := sessionCookie(t, w); c == nil || c.Value != testToken {
		t.Errorf("valid token: cookie %+v, want session cookie", c)
	}

	w = httptest.NewRecorder()
	guarded(testToken).ServeHTTP(w, postAuth("wrong"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Invalid token") {
		t.Error("wrong token: page doesn't say Invalid token")
	}
	if sessionCookie(t, w) != nil {
		t.Error("wrong token set a session cookie")
	}
}

func TestAccessURL(t *testing.T) {
	s := &Server{Host: DefaultHost, Port: 8080}
	if got := s.AccessURL(); got != "http://localhost:8080" {
		t.Errorf("AccessURL without token = %q, want plain URL", got)
	}
	s = &Server{Host: "192.168.1.10", Port: 8080, AccessToken: testToken}
	if got, want := s.AccessURL(), "http://192.168.1.10:8080/?token="+testToken; got != want {
		t.Errorf("AccessURL = %q, want %q", got, want)
	}
}
