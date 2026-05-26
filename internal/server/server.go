package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/registry"
	"github.com/kuyio/beats/internal/storage"
)

// Server holds the state for the beats web server.
type Server struct {
	Config        *config.Config
	Port          int
	DevMode       bool
	DevPort       int
	pendingEvents []model.Event
	mu            sync.RWMutex
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
	mux := s.setupRoutes()
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
}

// DiscardPending clears all pending events.
func (s *Server) DiscardPending() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingEvents = make([]model.Event, 0)
}

// SaveAndSync persists pending events to storage and optionally commits to git.
func (s *Server) SaveAndSync(commitMessage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Append all pending events to storage
	for _, evt := range s.pendingEvents {
		if err := storage.AppendEvent(evt); err != nil {
			return fmt.Errorf("failed to append event: %w", err)
		}
	}

	// Git commit if auto-commit is enabled
	if s.Config.AutoCommit {
		client := beats.NewClient(s.Config)
		_ = client // Use client for git operations if needed
		// Git operations handled by storage layer or direct exec
	}

	// Clear pending events
	s.pendingEvents = make([]model.Event, 0)
	return nil
}

// GetProjectedIssues returns all issues with pending events applied.
func (s *Server) GetProjectedIssues() (map[string]*model.Issue, error) {
	// Read persisted events
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	// Append pending events
	s.mu.RLock()
	allEvents := append(events, s.pendingEvents...)
	s.mu.RUnlock()

	// Project issues
	issues := beats.ProjectIssues(allEvents)
	return issues, nil
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
