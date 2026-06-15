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
func (s *Server) SetupRoutes() *http.ServeMux {
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

	// SSE endpoint (when hub is configured)
	if s.SSEHub != nil && s.ProxyURL == "" {
		handle("GET /api/events", s.handleSSE)
	}

	// Proxy mode: reverse-proxy /api/* to remote server with bearer token
	if s.ProxyURL != "" {
		remoteURL, err := url.Parse(s.ProxyURL)
		if err != nil {
			log.Fatalf("Invalid proxy URL: %v", err)
		}
		proxy := httputil.NewSingleHostReverseProxy(remoteURL)
		originalDirector := proxy.Director
		token := s.ProxyToken
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			req.Host = remoteURL.Host
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			respondError(w, http.StatusBadGateway, fmt.Sprintf("remote server unreachable: %v", err))
		}

		proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})
		mux.HandleFunc("GET /api/{path...}", proxyHandler)
		mux.HandleFunc("POST /api/{path...}", proxyHandler)
		mux.HandleFunc("PUT /api/{path...}", proxyHandler)
		mux.HandleFunc("DELETE /api/{path...}", proxyHandler)
		mux.HandleFunc("POST /auth/{path...}", proxyHandler)

		log.Printf("Proxy mode: forwarding /api/* to %s", s.ProxyURL)
	} else {
		// Local mode: API routes
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
		handle("GET /api/inbox", s.handleInbox)
		handle("GET /api/cycles", s.handleGetCycles)
		handle("GET /api/cycles/{id}/progress", s.handleCycleProgress)
		handle("GET /api/user", s.handleGetUser)
	}

	// MCP endpoint (when configured)
	if s.MCPHandler != nil {
		if s.SigningKey != nil {
			mux.Handle("POST /mcp", auth.RequireAuthHandler(s.VerifyKey, s.MCPHandler))
			mux.Handle("GET /mcp", auth.RequireAuthHandler(s.VerifyKey, s.MCPHandler))
			mux.Handle("DELETE /mcp", auth.RequireAuthHandler(s.VerifyKey, s.MCPHandler))
		} else {
			mux.Handle("POST /mcp", s.MCPHandler)
			mux.Handle("GET /mcp", s.MCPHandler)
			mux.Handle("DELETE /mcp", s.MCPHandler)
		}
	}

	// Headless mode: no static assets
	if s.Headless {
		return mux
	}

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
