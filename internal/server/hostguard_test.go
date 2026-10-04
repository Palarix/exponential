package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBindDefaultsToLoopback(t *testing.T) {
	s := NewServer(nil, 0, false, 0)
	l, err := s.Bind()
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	defer l.Close()
	ip := l.Addr().(*net.TCPAddr).IP
	if !ip.IsLoopback() {
		t.Fatalf("board bound %s, want a loopback address", ip)
	}
}

func TestHostGuard(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	cases := []struct {
		bind, host string
		want       int
	}{
		{DefaultHost, "localhost:8080", http.StatusOK},
		{DefaultHost, "LOCALHOST:5173", http.StatusOK},
		{DefaultHost, "127.0.0.1:8080", http.StatusOK},
		{DefaultHost, "[::1]:8080", http.StatusOK},
		{DefaultHost, "localhost", http.StatusOK},
		{DefaultHost, "evil.example:8080", http.StatusForbidden},
		{DefaultHost, "192.168.1.10:8080", http.StatusForbidden},
		{DefaultHost, "localhost.evil.example", http.StatusForbidden},
		{"192.168.1.10", "192.168.1.10:8080", http.StatusOK},
		{"192.168.1.10", "evil.example:8080", http.StatusForbidden},
		{"0.0.0.0", "evil.example:8080", http.StatusOK},
		{"::", "anything:8080", http.StatusOK},
	}
	for _, c := range cases {
		s := &Server{Host: c.bind}
		req := httptest.NewRequest("GET", "/api/issues", nil)
		req.Host = c.host
		w := httptest.NewRecorder()
		s.hostGuard(ok).ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("bind %q, Host %q: got %d, want %d", c.bind, c.host, w.Code, c.want)
		}
	}
}

func TestServerURL(t *testing.T) {
	cases := map[string]string{
		DefaultHost:    "http://localhost:8080",
		"0.0.0.0":      "http://localhost:8080",
		"192.168.1.10": "http://192.168.1.10:8080",
		"fd00::1":      "http://[fd00::1]:8080",
	}
	for host, want := range cases {
		s := &Server{Host: host, Port: 8080}
		if got := s.URL(); got != want {
			t.Errorf("URL() with host %q = %q, want %q", host, got, want)
		}
	}
}
