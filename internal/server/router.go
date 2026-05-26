package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// setupRoutes configures the HTTP routes for the server.
func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/issues", s.handleGetIssues)
	mux.HandleFunc("GET /api/issues/{id}", s.handleGetIssue)
	mux.HandleFunc("POST /api/draft", s.handleDraft)
	mux.HandleFunc("GET /api/pending", s.handleGetPending)
	mux.HandleFunc("POST /api/save", s.handleSave)
	mux.HandleFunc("DELETE /api/pending", s.handleDiscardPending)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("POST /api/config/labels", s.handleAddLabel)
	mux.HandleFunc("PUT /api/config/labels", s.handleUpdateLabel)
	mux.HandleFunc("DELETE /api/config/labels", s.handleDeleteLabel)
	mux.HandleFunc("GET /api/issues/{id}/history", s.handleGetIssueHistory)
	mux.HandleFunc("GET /api/instances", s.handleListInstances)
	mux.HandleFunc("GET /api/metrics", s.handleMetrics)
	mux.HandleFunc("GET /api/activity", s.handleActivity)

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
