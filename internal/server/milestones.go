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

// MilestoneResponse is one milestone on the board. A placeholder has a number
// but no ID; the first write to it saves it.
type MilestoneResponse struct {
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
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	CreatedAt   *time.Time     `json:"created_at,omitempty"`
	Counts      map[string]int `json:"counts"`
	Total       int            `json:"total"`
	Done        int            `json:"done"`
	// Points is left out for a past milestone completed before points were recorded.
	Points *model.MilestonePoints `json:"points,omitempty"`
}

// MilestonesResponse is the milestone board: Current and Next always exist.
type MilestonesResponse struct {
	Current  *MilestoneResponse  `json:"current"`
	Next     *MilestoneResponse  `json:"next"`
	Upcoming []MilestoneResponse `json:"upcoming"`
	Past     []MilestoneResponse `json:"past"`
}

type MilestoneProgressResponse struct {
	Milestone MilestoneResponse          `json:"milestone"`
	Counts    map[string]int             `json:"counts"`
	Total     int                        `json:"total"`
	Done      int                        `json:"done"`
	Days      []exponential.MilestoneDay `json:"days"`
}

func milestoneToResponse(r *model.Milestone, slot string, scope exponential.MilestoneScope) MilestoneResponse {
	counts := scope.Counts
	if counts == nil {
		counts = map[string]int{}
	}
	resp := MilestoneResponse{
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
		CompletedAt: r.CompletedAt,
		Counts:      counts,
		Total:       scope.Total,
		Done:        scope.Done,
		Points:      scope.Points,
	}
	if !r.CreatedAt.IsZero() {
		at := r.CreatedAt
		resp.CreatedAt = &at
	}
	return resp
}

// milestoneClient is the client for milestone writes from the web app.
func (s *Server) milestoneClient() *exponential.Client {
	client := exponential.NewClient(s.Config)
	client.Collapse = true
	client.Source = "web"
	return client
}

// respondMilestoneError maps a milestone write error to a status: a ref that
// names nothing is 404, every other refusal (placeholder, COMPLETED,
// duplicate label, bad input) is 400.
func respondMilestoneError(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "not found") {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondError(w, http.StatusBadRequest, err.Error())
}

// respondMilestone writes the board entry for the milestone with id.
func (s *Server) respondMilestone(w http.ResponseWriter, id string) {
	events, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load milestones")
		return
	}
	for _, v := range exponential.BuildMilestoneViews(events) {
		if v.ID == id {
			respondJSON(w, http.StatusOK, milestoneToResponse(v.Milestone, v.Slot, v.MilestoneScope))
			return
		}
	}
	respondError(w, http.StatusNotFound, fmt.Sprintf("milestone %s not found", id))
}

func (s *Server) handleGetMilestones(w http.ResponseWriter, r *http.Request) {
	events, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load milestones")
		return
	}
	resp := MilestonesResponse{Upcoming: []MilestoneResponse{}, Past: []MilestoneResponse{}}
	for _, v := range exponential.BuildMilestoneViews(events) {
		rr := milestoneToResponse(v.Milestone, v.Slot, v.MilestoneScope)
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

func (s *Server) handleMilestoneProgress(w http.ResponseWriter, r *http.Request) {
	events, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load events")
		return
	}
	milestone, err := exponential.ResolveMilestone(exponential.ProjectMilestones(events), r.PathValue("id"))
	if err == nil && milestone == nil {
		err = fmt.Errorf("milestone %q not found", r.PathValue("id"))
	}
	if err != nil {
		respondMilestoneError(w, err)
		return
	}
	// Archived events keep the burndown of a past milestone whole.
	archived, err := storage.ReadArchivedEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load archived events")
		return
	}
	p := exponential.ComputeMilestoneProgress(append(archived, events...), milestone, time.Now())
	respondJSON(w, http.StatusOK, MilestoneProgressResponse{
		Milestone: milestoneToResponse(p.Milestone, "", p.MilestoneScope),
		Counts:    p.Counts,
		Total:     p.Total,
		Done:      p.Done,
		Days:      p.Days,
	})
}

type milestoneFieldsRequest struct {
	Label      *string `json:"label"`
	Goal       *string `json:"goal"`
	TargetDate *string `json:"target_date"`
}

func (f milestoneFieldsRequest) input() exponential.MilestoneInput {
	return exponential.MilestoneInput{Label: f.Label, Goal: f.Goal, TargetDate: f.TargetDate}
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

func (s *Server) handleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	var req milestoneFieldsRequest
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	milestone, err := s.milestoneClient().CreateMilestone(req.input())
	if err != nil {
		respondMilestoneError(w, err)
		return
	}
	s.notify(TopicIssues)
	s.respondMilestone(w, milestone.ID)
}

func (s *Server) handleUpdateMilestone(w http.ResponseWriter, r *http.Request) {
	var req milestoneFieldsRequest
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	milestone, err := s.milestoneClient().UpdateMilestone(r.PathValue("id"), req.input())
	if err != nil {
		respondMilestoneError(w, err)
		return
	}
	s.notify(TopicIssues)
	s.respondMilestone(w, milestone.ID)
}

func (s *Server) handleReorderMilestone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Before string `json:"before"`
		After  string `json:"after"`
	}
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	milestone, err := s.milestoneClient().ReorderMilestone(req.ID, req.Before, req.After)
	if err != nil {
		respondMilestoneError(w, err)
		return
	}
	s.notify(TopicIssues)
	s.respondMilestone(w, milestone.ID)
}

func (s *Server) handleCompleteMilestone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Issues string `json:"issues"`
		To     string `json:"to"`
	}
	if err := decodeOptional(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	milestone, msgs, err := s.milestoneClient().CompleteMilestone(r.PathValue("id"), req.Issues, req.To)
	if err != nil {
		respondMilestoneError(w, err)
		return
	}
	s.notify(TopicIssues)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"milestone": milestoneToResponse(milestone, exponential.SlotPast, exponential.MilestoneScope{}),
		"messages":  msgs,
	})
}

func (s *Server) handleDeleteMilestone(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.milestoneClient().DeleteMilestone(r.PathValue("id"), r.URL.Query().Get("issues"), r.URL.Query().Get("to"))
	if err != nil {
		respondMilestoneError(w, err)
		return
	}
	s.notify(TopicIssues)
	respondJSON(w, http.StatusOK, map[string]interface{}{"messages": msgs})
}
