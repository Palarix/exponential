package server

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

// DefaultHost is the address the board binds to unless overridden with --host.
// The board is unauthenticated, so it must not be reachable from the network
// by default.
const DefaultHost = "127.0.0.1"

// IsLoopbackHost reports whether host names the local machine only.
func IsLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// isWildcardHost reports whether host binds all interfaces.
func isWildcardHost(host string) bool {
	if host == "" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}

// URL returns the address users should open in a browser.
func (s *Server) URL() string {
	host := s.Host
	if IsLoopbackHost(host) || isWildcardHost(host) {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(s.Port))
}

// hostGuard rejects requests whose Host header doesn't name this server,
// which blocks DNS-rebinding attacks from web pages in the user's browser.
// Loopback names are always allowed, plus s.Host when it is a concrete
// address. Binding a wildcard address disables the guard: the user asked
// for network exposure and clients may use any name for this machine.
func (s *Server) hostGuard(next http.Handler) http.Handler {
	if isWildcardHost(s.Host) {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
		if !IsLoopbackHost(host) && !strings.EqualFold(host, s.Host) {
			respondError(w, http.StatusForbidden, "host not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}
