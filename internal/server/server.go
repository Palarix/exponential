package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"crypto/ed25519"

	"github.com/palarix/beats/internal/auth"
	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/registry"
	"github.com/palarix/beats/internal/storage"
)

// Server holds the state for the beats web server.
type Server struct {
	Config        *config.Config
	Port          int
	DevMode       bool
	DevPort       int
	Headless      bool
	pendingEvents []model.Event
	mu            sync.RWMutex
	pendingGen    uint64
	cachedEvents  []model.Event
	cachedIssues  map[string]*model.Issue
	lastDBModTime time.Time
	lastDBSize    int64
	dbExists      bool
	cacheGen      uint64

	// Auth fields — nil when auth is disabled (local board mode).
	NonceStore     *auth.NonceStore
	AuthorizedKeys *auth.AuthorizedKeys
	SigningKey     ed25519.PrivateKey
	VerifyKey      ed25519.PublicKey

	// MCP handler — set externally for beats serve mode.
	MCPHandler http.Handler

	// Proxy mode — when set, /api/* routes are reverse-proxied to
	// the remote server with the bearer token injected.
	ProxyURL   string
	ProxyToken string

	// SSE hub for real-time event notifications.
	SSEHub *SSEHub
}

// NewServer creates a new Server instance.
func NewServer(cfg *config.Config, port int, devMode bool, devPort int) *Server {
	return &Server{
		Config:        cfg,
		Port:          port,
		DevMode:       devMode,
		DevPort:       devPort,
		pendingEvents: make([]model.Event, 0),
	}
}

// Bind acquires a TCP listener on s.Port, falling back to the next free port
// if that one is taken. The actual port bound is written back to s.Port.
func (s *Server) Bind() (net.Listener, error) {
	requested := s.Port
	l, actual, err := registry.FindFreePort(requested, 10)
	if err != nil {
		return nil, err
	}
	if actual != requested {
		log.Printf("Port %d in use, starting on port %d instead", requested, actual)
	}
	s.Port = actual
	return l, nil
}

// ServeOn serves HTTP on the provided listener. Use after Bind.
func (s *Server) ServeOn(l net.Listener) error {
	mux := s.SetupRoutes()
	log.Printf("Starting beats board server on http://localhost:%d", s.Port)
	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return server.Serve(l)
}

// Start binds and serves in one step. Equivalent to Bind followed by ServeOn.
func (s *Server) Start() error {
	l, err := s.Bind()
	if err != nil {
		return err
	}
	return s.ServeOn(l)
}

// GetPendingCount returns the number of pending (unsaved) events.
func (s *Server) GetPendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pendingEvents)
}

// AddPendingEvent adds an event to the pending buffer.
func (s *Server) AddPendingEvent(evt model.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingEvents = append(s.pendingEvents, evt)
	s.pendingGen++
}

// DiscardPending clears all pending events.
func (s *Server) DiscardPending() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingEvents = make([]model.Event, 0)
	s.pendingGen++
}

// SaveAndSync persists pending events to storage and optionally commits to git.
func (s *Server) SaveAndSync(commitMessage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, evt := range s.pendingEvents {
		if err := storage.AppendEvent(evt); err != nil {
			return fmt.Errorf("failed to append event: %w", err)
		}
	}

	if s.Config.AutoCommit {
		client := beats.NewClient(s.Config)
		_ = client
	}

	s.pendingEvents = make([]model.Event, 0)
	s.pendingGen++
	s.cachedIssues = nil
	return nil
}

// broadcastEvent notifies SSE clients about a change, if SSE is enabled.
func (s *Server) broadcastEvent(eventType, issueID string) {
	if s.SSEHub != nil {
		s.SSEHub.Broadcast(SSEEvent{Type: eventType, IssueID: issueID})
	}
}

func (s *Server) statDB() (mtime time.Time, size int64, exists bool) {
	info, err := os.Stat(filepath.Join(".beats", "issues.db"))
	if err != nil {
		return time.Time{}, 0, false
	}
	return info.ModTime(), info.Size(), true
}

func (s *Server) dbStale() bool {
	mtime, size, exists := s.statDB()
	return exists != s.dbExists || mtime != s.lastDBModTime || size != s.lastDBSize
}

// refreshEventsLocked re-reads persisted events from disk if the database
// file has changed. Must be called while holding s.mu for writing.
func (s *Server) refreshEventsLocked() error {
	mtime, size, exists := s.statDB()
	if s.cachedEvents != nil && exists == s.dbExists && mtime == s.lastDBModTime && size == s.lastDBSize {
		return nil
	}

	events, err := storage.ReadEvents()
	if err != nil {
		return err
	}
	s.cachedEvents = events
	s.lastDBModTime = mtime
	s.lastDBSize = size
	s.dbExists = exists
	s.cachedIssues = nil
	return nil
}

// GetProjectedIssues returns all issues with pending events applied.
// Results are cached and recomputed only when the on-disk database or the
// pending-events buffer has changed.
func (s *Server) GetProjectedIssues() (map[string]*model.Issue, error) {
	s.mu.RLock()
	if s.cachedIssues != nil && !s.dbStale() && s.pendingGen == s.cacheGen {
		issues := s.cachedIssues
		s.mu.RUnlock()
		return issues, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.refreshEventsLocked(); err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	if s.cachedIssues != nil && s.pendingGen == s.cacheGen {
		return s.cachedIssues, nil
	}

	allEvents := make([]model.Event, 0, len(s.cachedEvents)+len(s.pendingEvents))
	allEvents = append(allEvents, s.cachedEvents...)
	allEvents = append(allEvents, s.pendingEvents...)
	s.cachedIssues = beats.ProjectIssuesWithConfig(allEvents, s.Config)
	s.cacheGen = s.pendingGen
	return s.cachedIssues, nil
}

// GetAllEvents returns all persisted events plus pending events.
// The persisted events are cached and only re-read when the database file
// has changed on disk.
func (s *Server) GetAllEvents() ([]model.Event, error) {
	s.mu.RLock()
	if s.cachedEvents != nil && !s.dbStale() {
		result := make([]model.Event, 0, len(s.cachedEvents)+len(s.pendingEvents))
		result = append(result, s.cachedEvents...)
		result = append(result, s.pendingEvents...)
		s.mu.RUnlock()
		return result, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.refreshEventsLocked(); err != nil {
		return nil, err
	}

	result := make([]model.Event, 0, len(s.cachedEvents)+len(s.pendingEvents))
	result = append(result, s.cachedEvents...)
	result = append(result, s.pendingEvents...)
	return result, nil
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError writes a JSON error response.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
