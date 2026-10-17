package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// Issue drafts: unfinished new issues saved from the new-issue modal.
// Not to be confused with handleDraft (POST /api/draft), the generic
// write-event endpoint.

// issueDraftClient returns a client for the issue draft routes.
func (s *Server) issueDraftClient() *exponential.Client {
	client := exponential.NewClient(s.Config)
	client.Source = "web"
	return client
}

func respondIssueDraftError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrDraftNotFound), errors.Is(err, storage.ErrInvalidDraftID):
		respondError(w, http.StatusNotFound, "draft not found")
	default:
		respondError(w, http.StatusInternalServerError, "failed to save draft")
	}
}

func decodeIssueDraftPayload(w http.ResponseWriter, r *http.Request) (model.IssueDraftPayload, bool) {
	var p model.IssueDraftPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return p, false
	}
	if err := exponential.ValidateIssueDraftPayload(p); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return p, false
	}
	return p, true
}

func (s *Server) handleListIssueDrafts(w http.ResponseWriter, r *http.Request) {
	drafts, err := s.issueDraftClient().ListIssueDrafts()
	if err != nil {
		respondIssueDraftError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, drafts)
}

func (s *Server) handleCreateIssueDraft(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeIssueDraftPayload(w, r)
	if !ok {
		return
	}
	d, err := s.issueDraftClient().CreateIssueDraft(p)
	if err != nil {
		respondIssueDraftError(w, err)
		return
	}
	s.notify(TopicDrafts)
	respondJSON(w, http.StatusCreated, d)
}

func (s *Server) handleUpdateIssueDraft(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeIssueDraftPayload(w, r)
	if !ok {
		return
	}
	d, err := s.issueDraftClient().UpdateIssueDraft(r.PathValue("id"), p)
	if err != nil {
		respondIssueDraftError(w, err)
		return
	}
	s.notify(TopicDrafts)
	respondJSON(w, http.StatusOK, d)
}

func (s *Server) handleDeleteIssueDraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.issueDraftClient().DeleteIssueDraft(id); err != nil {
		respondIssueDraftError(w, err)
		return
	}
	s.notify(TopicDrafts)
	w.WriteHeader(http.StatusNoContent)
}
