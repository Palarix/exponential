package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/palarix/beats/internal/auth"
)

// setupRoutes configures the HTTP routes for the server.
func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// handle conditionally wraps routes with auth middleware when auth is enabled.
	handle := func(pattern string, handler http.HandlerFunc) {
		if s.SigningKey != nil {
			mux.HandleFunc(pattern, auth.RequireAuth(s.VerifyKey, handler))
		} else {
			mux.HandleFunc(pattern, handler)
		}
	}

	// Auth endpoints (public, only registered when auth is enabled)
	if s.SigningKey != nil {
		mux.HandleFunc("POST /auth/challenge", s.handleAuthChallenge)
		mux.HandleFunc("POST /auth/verify", s.handleAuthVerify)
	}

	// Health check (always public)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// API routes
	handle("GET /api/issues", s.handleGetIssues)
	handle("GET /api/issues/{id}", s.handleGetIssue)
	handle("POST /api/draft", s.handleDraft)
	handle("GET /api/pending", s.handleGetPending)
	handle("POST /api/save", s.handleSave)
	handle("DELETE /api/pending", s.handleDiscardPending)
	handle("GET /api/config", s.handleGetConfig)
	handle("POST /api/config/labels", s.handleAddLabel)
	handle("PUT /api/config/labels", s.handleUpdateLabel)
	handle("DELETE /api/config/labels", s.handleDeleteLabel)
	handle("GET /api/issues/{id}/history", s.handleGetIssueHistory)
	handle("GET /api/issues/{id}/commits", s.handleGetIssueCommits)
	handle("GET /api/issues/{id}/files", s.handleGetIssueFiles)
	handle("GET /api/issues/{id}/diff", s.handleGetIssueDiff)
	handle("GET /api/issues/{id}/commits/{sha}/diff", s.handleGetCommitDiff)
	handle("GET /api/issues/{id}/mergeability", s.handleMergeability)
	handle("POST /api/issues/{id}/merge", s.handleMergeIssue)
	handle("POST /api/issues/{id}/start", s.handleStartWork)
	handle("GET /api/instances", s.handleListInstances)
	handle("GET /api/metrics", s.handleMetrics)
	handle("GET /api/activity", s.handleActivity)
	handle("GET /api/cycles", s.handleGetCycles)
	handle("GET /api/cycles/{id}/progress", s.handleCycleProgress)
	handle("GET /api/user", s.handleGetUser)

	// Development mode: proxy all static requests to Vite dev server
	if s.DevMode {
		viteURL, err := url.Parse(fmt.Sprintf("http://localhost:%d", s.DevPort))
		if err != nil {
			log.Fatalf("Failed to parse Vite dev URL: %v", err)
		}

		proxy := httputil.NewSingleHostReverseProxy(viteURL)

		log.Printf("Dev mode enabled: proxying static assets to http://localhost:%d", s.DevPort)

		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})

		return mux
	}

	// Production mode: serve embedded static files
	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(static))

	// Catch-all for SPA routing
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Serve static assets directly
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			fileServer.ServeHTTP(w, r)
			return
		}

		// For all other routes, serve index.html (SPA routing)
		// Check if file exists first
		if r.URL.Path != "/" {
			_, err := fs.Stat(static, strings.TrimPrefix(r.URL.Path, "/"))
			if err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Serve index.html for SPA routes
		indexFile, err := fs.ReadFile(static, "index.html")
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexFile)
	})

	return mux
}
