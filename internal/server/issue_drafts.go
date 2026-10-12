package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/palarix/exponential/internal/auth"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// Issue drafts: unfinished new issues saved from the new-issue modal.
// Not to be confused with handleDraft (POST /api/draft), the generic
// write-event endpoint.

// issueDraftClient returns a client acting as the request's user, plus the
// owner that scopes issue draft access. With auth off the owner is empty,
// so every issue draft is visible.
func (s *Server) issueDraftClient(r *http.Request) (*exponential.Client, string) {
	client := exponential.NewClient(s.Config)
	client.Source = "web"
	owner := ""
	if user, ok := auth.UserFromContext(r.Context()); ok {
		client.UserOverride = user.Raw
		owner = user.Raw
	}
	return client, owner
}

func respondIssueDraftError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrDraftNotFound), errors.Is(err, storage.ErrInvalidDraftID):
		respondError(w, http.StatusNotFound, "draft not found")
	case errors.Is(err, exponential.ErrLocalOnly):
		respondError(w, http.StatusNotImplemented, err.Error())
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
	client, owner := s.issueDraftClient(r)
	drafts, err := client.ListIssueDrafts(owner)
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
	client, _ := s.issueDraftClient(r)
	d, err := client.CreateIssueDraft(p)
	if err != nil {
		respondIssueDraftError(w, err)
		return
	}
	s.broadcastEvent("draft_changed", d.ID)
	respondJSON(w, http.StatusCreated, d)
}

func (s *Server) handleUpdateIssueDraft(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeIssueDraftPayload(w, r)
	if !ok {
		return
	}
	client, owner := s.issueDraftClient(r)
	d, err := client.UpdateIssueDraft(r.PathValue("id"), owner, p)
	if err != nil {
		respondIssueDraftError(w, err)
		return
	}
	s.broadcastEvent("draft_changed", d.ID)
	respondJSON(w, http.StatusOK, d)
}

func (s *Server) handleDeleteIssueDraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	client, owner := s.issueDraftClient(r)
	if err := client.DeleteIssueDraft(id, owner); err != nil {
		respondIssueDraftError(w, err)
		return
	}
	s.broadcastEvent("draft_changed", id)
	w.WriteHeader(http.StatusNoContent)
}
