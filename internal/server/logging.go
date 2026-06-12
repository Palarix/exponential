package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type utcLogWriter struct{}

func (utcLogWriter) Write(p []byte) (int, error) {
	return fmt.Fprintf(os.Stderr, "%s %s", time.Now().UTC().Format(time.RFC3339), p)
}

// ConfigureLogging sets the log package to use RFC3339 UTC timestamps,
// matching the format used by RequestLogger.
func ConfigureLogging() {
	log.SetFlags(0)
	log.SetOutput(utcLogWriter{})
}

// RequestLogger wraps an http.Handler and logs each request as a
// comma-separated line:
//
//	timestamp,method,path,status,duration_ms,client
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Printf("%s,%s,%d,%d,%s",
			r.Method,
			r.URL.Path,
			rec.status,
			time.Since(start).Milliseconds(),
			clientFromRequest(r),
		)
	})
}

// clientFromRequest extracts the user identity from the JWT bearer token
// on the request. Returns "anonymous" if no valid token is present.
func clientFromRequest(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "anonymous"
	}
	parts := strings.SplitN(strings.TrimPrefix(h, "Bearer "), ".", 3)
	if len(parts) != 3 {
		return "anonymous"
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "anonymous"
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Sub == "" {
		return "anonymous"
	}
	// Extract username from "Name <user@host>" format
	if start := strings.Index(claims.Sub, "<"); start != -1 {
		if at := strings.Index(claims.Sub[start:], "@"); at != -1 {
			return claims.Sub[start+1 : start+at]
		}
	}
	return claims.Sub
}
