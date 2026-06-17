package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/palarix/exponential/internal/auth"
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

	// handleCap wraps with auth + capability check when auth is enabled.
	perms := s.Config.Permissions
	handleCap := func(pattern string, capability string, handler http.HandlerFunc) {
		if s.SigningKey != nil {
			mux.HandleFunc(pattern, auth.RequireCapability(s.VerifyKey, perms, capability, handler))
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
			respondError(w, http.StatusBadGateway, "remote server unreachable — check that the server is running and the URL is correct")
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
		// Local mode: API routes — read endpoints
		handleCap("GET /api/issues", "issue.read", s.handleGetIssues)
		handleCap("GET /api/issues/{id}", "issue.read", s.handleGetIssue)
		handleCap("GET /api/issues/{id}/history", "issue.read", s.handleGetIssueHistory)
		handleCap("GET /api/issues/{id}/commits", "issue.read", s.handleGetIssueCommits)
		handleCap("GET /api/issues/{id}/files", "issue.read", s.handleGetIssueFiles)
		handleCap("GET /api/issues/{id}/diff", "issue.read", s.handleGetIssueDiff)
		handleCap("GET /api/issues/{id}/commits/{sha}/diff", "issue.read", s.handleGetCommitDiff)
		handleCap("GET /api/issues/{id}/mergeability", "issue.read", s.handleMergeability)
		handleCap("GET /api/config", "issue.read", s.handleGetConfig)
		handleCap("GET /api/pending", "issue.read", s.handleGetPending)
		handleCap("GET /api/instances", "issue.read", s.handleListInstances)
		handleCap("GET /api/metrics", "issue.read", s.handleMetrics)
		handleCap("GET /api/activity", "issue.read", s.handleActivity)
		handleCap("GET /api/inbox", "issue.read", s.handleInbox)
		handleCap("GET /api/inbox/status", "issue.read", s.handleInboxStatus)
		handleCap("GET /api/cycles", "issue.read", s.handleGetCycles)
		handleCap("GET /api/cycles/{id}/progress", "issue.read", s.handleCycleProgress)
		handle("GET /api/user", s.handleGetUser)

		// Write endpoints
		handleCap("POST /api/draft", "issue.create", s.handleDraft)
		handleCap("POST /api/save", "issue.update", s.handleSave)
		handleCap("DELETE /api/pending", "issue.update", s.handleDiscardPending)
		handleCap("POST /api/config/labels", "issue.update", s.handleAddLabel)
		handleCap("PUT /api/config/labels", "issue.update", s.handleUpdateLabel)
		handleCap("DELETE /api/config/labels", "issue.update", s.handleDeleteLabel)
		handleCap("POST /api/issues/{id}/merge", "issue.merge", s.handleMergeIssue)
		handleCap("POST /api/issues/{id}/start", "issue.start", s.handleStartWork)
		handleCap("POST /api/inbox/read", "issue.read", s.handleInboxRead)
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
