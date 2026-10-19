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
func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health check (always public)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// The event stream (when the hub is configured).
	s.registerLiveRoutes(mux)

	// Local machine routes: they describe the viewer's machine (e.g.
	// worktree paths).
	mux.HandleFunc("GET /api/local/issues/{id}/worktree", s.handleGetLocalWorktree)
	mux.HandleFunc("GET /api/local/mode", s.handleLocalMode)
	s.registerChatRoutes(mux)

	// API routes — read endpoints
	mux.HandleFunc("GET /api/issues", s.handleGetIssues)
	mux.HandleFunc("GET /api/issues/{id}", s.handleGetIssue)
	mux.HandleFunc("GET /api/issues/{id}/history", s.handleGetIssueHistory)
	mux.HandleFunc("GET /api/issues/{id}/commits", s.handleGetIssueCommits)
	mux.HandleFunc("GET /api/issues/{id}/files", s.handleGetIssueFiles)
	mux.HandleFunc("GET /api/issues/{id}/diff", s.handleGetIssueDiff)
	mux.HandleFunc("GET /api/issues/{id}/commits/{sha}/diff", s.handleGetCommitDiff)
	mux.HandleFunc("GET /api/issues/{id}/artifacts/{filename}", s.handleGetArtifact)
	mux.HandleFunc("GET /api/issues/{id}/mergeability", s.handleMergeability)
	mux.HandleFunc("GET /api/issues/{id}/merged", s.handleMergedChanges)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("GET /api/pending", s.handleGetPending)
	mux.HandleFunc("GET /api/instances", s.handleListInstances)
	mux.HandleFunc("GET /api/metrics", s.handleMetrics)
	mux.HandleFunc("GET /api/activity", s.handleActivity)
	mux.HandleFunc("GET /api/timeline", s.handleTimeline)
	mux.HandleFunc("GET /api/commits/{sha}", s.handleCommitDetail)
	mux.HandleFunc("GET /api/inbox", s.handleInbox)
	mux.HandleFunc("GET /api/inbox/status", s.handleInboxStatus)
	mux.HandleFunc("GET /api/cycles", s.handleGetCycles)
	mux.HandleFunc("GET /api/cycles/{id}/progress", s.handleCycleProgress)
	mux.HandleFunc("GET /api/releases", s.handleGetReleases)
	mux.HandleFunc("GET /api/releases/{id}/progress", s.handleReleaseProgress)
	mux.HandleFunc("GET /api/user", s.handleGetUser)

	// Write endpoints
	mux.HandleFunc("POST /api/draft", s.handleDraft)
	mux.HandleFunc("POST /api/save", s.handleSave)
	mux.HandleFunc("DELETE /api/pending", s.handleDiscardPending)
	mux.HandleFunc("POST /api/config/labels", s.handleAddLabel)
	mux.HandleFunc("PUT /api/config/labels", s.handleUpdateLabel)
	mux.HandleFunc("DELETE /api/config/labels", s.handleDeleteLabel)
	mux.HandleFunc("POST /api/issues/{id}/merge", s.handleMergeIssue)
	mux.HandleFunc("POST /api/issues/{id}/start", s.handleStartWork)
	mux.HandleFunc("POST /api/inbox/read", s.handleInboxRead)
	mux.HandleFunc("POST /api/releases", s.handleCreateRelease)
	mux.HandleFunc("POST /api/releases/reorder", s.handleReorderRelease)
	mux.HandleFunc("PUT /api/releases/{id}", s.handleUpdateRelease)
	mux.HandleFunc("POST /api/releases/{id}/complete", s.handleCompleteRelease)
	mux.HandleFunc("DELETE /api/releases/{id}", s.handleDeleteRelease)

	// Issue drafts (unrelated to POST /api/draft above)
	mux.HandleFunc("GET /api/issue-drafts", s.handleListIssueDrafts)
	mux.HandleFunc("POST /api/issue-drafts", s.handleCreateIssueDraft)
	mux.HandleFunc("PUT /api/issue-drafts/{id}", s.handleUpdateIssueDraft)
	mux.HandleFunc("DELETE /api/issue-drafts/{id}", s.handleDeleteIssueDraft)

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
	mux.HandleFunc("GET /", spaHandler(static))

	return mux
}

// isBackendPath reports whether path belongs to the API or auth
// endpoints, which must never fall through to the SPA's index.html.
func isBackendPath(path string) bool {
	for _, prefix := range []string{"/api", "/auth"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// spaHandler serves the built frontend in static: asset files as they are,
// and index.html for every other route so client-side routing works.
// Unrouted backend paths get a 404.
func spaHandler(static fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(static))
	return func(w http.ResponseWriter, r *http.Request) {
		if isBackendPath(r.URL.Path) {
			respondError(w, http.StatusNotFound, "not found")
			return
		}

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
	}
}
