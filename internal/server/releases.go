package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// ReleaseResponse is one release on the board. A placeholder has a number
// but no ID; the first write to it saves it.
type ReleaseResponse struct {
	ID          string         `json:"id,omitempty"`
	Number      int            `json:"number"`
	Name        string         `json:"name"`
	Label       string         `json:"label,omitempty"`
	Goal        string         `json:"goal,omitempty"`
	State       string         `json:"state"`
	Slot        string         `json:"slot,omitempty"`
	Placeholder bool           `json:"placeholder,omitempty"`
	Locked      bool           `json:"locked,omitempty"` // a Next placeholder that refuses writes until Current is saved
	SortOrder   string         `json:"sort_order,omitempty"`
	TargetDate  string         `json:"target_date,omitempty"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	ReleasedAt  *time.Time     `json:"released_at,omitempty"`
	CreatedAt   *time.Time     `json:"created_at,omitempty"`
	Counts      map[string]int `json:"counts"`
	Total       int            `json:"total"`
	Done        int            `json:"done"`
}

// ReleasesResponse is the release board: Current and Next always exist.
type ReleasesResponse struct {
	Current  *ReleaseResponse  `json:"current"`
	Next     *ReleaseResponse  `json:"next"`
	Upcoming []ReleaseResponse `json:"upcoming"`
	Past     []ReleaseResponse `json:"past"`
}

type ReleaseProgressResponse struct {
	Release ReleaseResponse          `json:"release"`
	Counts  map[string]int           `json:"counts"`
	Total   int                      `json:"total"`
	Done    int                      `json:"done"`
	Days    []exponential.ReleaseDay `json:"days"`
}

func releaseToResponse(r *model.Release, slot string, counts map[string]int, total, done int) ReleaseResponse {
	if counts == nil {
		counts = map[string]int{}
	}
	resp := ReleaseResponse{
		ID:          r.ID,
		Number:      r.Number,
		Name:        r.Display(),
		Label:       r.Label,
		Goal:        r.Goal,
		State:       string(r.State),
		Slot:        slot,
		Placeholder: r.Placeholder,
		Locked:      r.Locked,
		SortOrder:   r.SortOrder,
		TargetDate:  r.TargetDate,
		StartedAt:   r.StartedAt,
		ReleasedAt:  r.ReleasedAt,
		Counts:      counts,
		Total:       total,
		Done:        done,
	}
	if !r.CreatedAt.IsZero() {
		at := r.CreatedAt
		resp.CreatedAt = &at
	}
	return resp
}

// releaseClient is the client for release writes from the web app.
func (s *Server) releaseClient() *exponential.Client {
	client := exponential.NewClient(s.Config)
	client.Collapse = true
	client.Source = "web"
	return client
}

// respondReleaseError maps a release write error to a status: a ref that
// names nothing is 404, every other refusal (placeholder, RELEASED,
// duplicate label, bad input) is 400.
func respondReleaseError(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "not found") {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondError(w, http.StatusBadRequest, err.Error())
}

// respondRelease writes the board entry for the release with id.
func (s *Server) respondRelease(w http.ResponseWriter, id string) {
	events, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load releases")
		return
	}
	for _, v := range exponential.BuildReleaseViews(events) {
		if v.ID == id {
			respondJSON(w, http.StatusOK, releaseToResponse(v.Release, v.Slot, v.Counts, v.Total, v.Done))
			return
		}
	}
	respondError(w, http.StatusNotFound, fmt.Sprintf("release %s not found", id))
}

func (s *Server) handleGetReleases(w http.ResponseWriter, r *http.Request) {
	events, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load releases")
		return
	}
	resp := ReleasesResponse{Upcoming: []ReleaseResponse{}, Past: []ReleaseResponse{}}
	for _, v := range exponential.BuildReleaseViews(events) {
		rr := releaseToResponse(v.Release, v.Slot, v.Counts, v.Total, v.Done)
		switch v.Slot {
		case exponential.SlotCurrent:
			resp.Current = &rr
		case exponential.SlotNext:
			resp.Next = &rr
		case exponential.SlotUpcoming:
			resp.Upcoming = append(resp.Upcoming, rr)
		case exponential.SlotPast:
			resp.Past = append(resp.Past, rr)
		}
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleReleaseProgress(w http.ResponseWriter, r *http.Request) {
	events, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load events")
		return
	}
	release, err := exponential.ResolveRelease(exponential.ProjectReleases(events), r.PathValue("id"))
	if err == nil && release == nil {
		err = fmt.Errorf("release %q not found", r.PathValue("id"))
	}
	if err != nil {
		respondReleaseError(w, err)
		return
	}
	// Archived events keep the burndown of a past release whole.
	archived, err := storage.ReadArchivedEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load archived events")
		return
	}
	p := exponential.ComputeReleaseProgress(append(archived, events...), release, time.Now())
	respondJSON(w, http.StatusOK, ReleaseProgressResponse{
		Release: releaseToResponse(p.Release, "", p.Counts, p.Total, p.Done),
		Counts:  p.Counts,
		Total:   p.Total,
		Done:    p.Done,
		Days:    p.Days,
	})
}

type releaseFieldsRequest struct {
	Label      *string `json:"label"`
	Goal       *string `json:"goal"`
	TargetDate *string `json:"target_date"`
}

func (f releaseFieldsRequest) input() exponential.ReleaseInput {
	return exponential.ReleaseInput{Label: f.Label, Goal: f.Goal, TargetDate: f.TargetDate}
}

// decodeOptional decodes a JSON body into dst; an empty body is allowed.
func decodeOptional(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return nil
	}
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (s *Server) handleCreateRelease(w http.ResponseWriter, r *http.Request) {
	var req releaseFieldsRequest
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	release, err := s.releaseClient().CreateRelease(req.input())
	if err != nil {
		respondReleaseError(w, err)
		return
	}
	s.notify(TopicIssues)
	s.respondRelease(w, release.ID)
}

func (s *Server) handleUpdateRelease(w http.ResponseWriter, r *http.Request) {
	var req releaseFieldsRequest
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	release, err := s.releaseClient().UpdateRelease(r.PathValue("id"), req.input())
	if err != nil {
		respondReleaseError(w, err)
		return
	}
	s.notify(TopicIssues)
	s.respondRelease(w, release.ID)
}

func (s *Server) handleReorderRelease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Before string `json:"before"`
		After  string `json:"after"`
	}
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	release, err := s.releaseClient().ReorderRelease(req.ID, req.Before, req.After)
	if err != nil {
		respondReleaseError(w, err)
		return
	}
	s.notify(TopicIssues)
	s.respondRelease(w, release.ID)
}

func (s *Server) handleCompleteRelease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Issues string `json:"issues"`
		To     string `json:"to"`
	}
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	release, msgs, err := s.releaseClient().CompleteRelease(r.PathValue("id"), req.Issues, req.To)
	if err != nil {
		respondReleaseError(w, err)
		return
	}
	s.notify(TopicIssues)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"release":  releaseToResponse(release, exponential.SlotPast, nil, 0, 0),
		"messages": msgs,
	})
}

func (s *Server) handleDeleteRelease(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.releaseClient().DeleteRelease(r.PathValue("id"), r.URL.Query().Get("issues"), r.URL.Query().Get("to"))
	if err != nil {
		respondReleaseError(w, err)
		return
	}
	s.notify(TopicIssues)
	respondJSON(w, http.StatusOK, map[string]interface{}{"messages": msgs})
}
