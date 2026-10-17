package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func builtStatic() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>xpo</title>")},
		"favicon.svg":   {Data: []byte("<svg/>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
}

// With the frontend built, unknown API paths must still 404 instead of
// getting the app's index.html.
func TestSPA_UnknownBackendPathsAreNotFound(t *testing.T) {
	h := spaHandler(builtStatic())
	for _, path := range []string{"/api", "/api/nope", "/api/local/chats/x/stream", "/auth/nope"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, w.Code)
		}
	}
}

func TestSPA_ServesAppRoutesAndFiles(t *testing.T) {
	h := spaHandler(builtStatic())
	for path, want := range map[string]string{
		"/":              "<!doctype html><title>xpo</title>",
		"/board":         "<!doctype html><title>xpo</title>",
		"/apiary":        "<!doctype html><title>xpo</title>",
		"/favicon.svg":   "<svg/>",
		"/assets/app.js": "console.log(1)",
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK || w.Body.String() != want {
			t.Errorf("GET %s = %d %q, want 200 %q", path, w.Code, w.Body.String(), want)
		}
	}
}
