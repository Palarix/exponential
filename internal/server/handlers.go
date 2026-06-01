package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/registry"
	"github.com/palarix/beats/internal/storage"
	"github.com/palarix/beats/internal/version"
)

func (s *Server) handleGetIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sorted := beats.SortIssues(issues)

	s.mu.RLock()
	pendingIDs := make(map[string]bool)
	for _, evt := range s.pendingEvents {
		pendingIDs[evt.ID] = true
	}
	s.mu.RUnlock()

	var response []IssueResponse
	for _, issue := range sorted {
		resp := issueToResponse(issue)
		if pendingIDs[issue.ID] {
			resp.IsPending = true
		}
		response = append(response, resp)
	}

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return
	}

	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	issue, exists := issues[id]
	if !exists {
		respondError(w, http.StatusNotFound, "issue not found")
		return
	}

	respondJSON(w, http.StatusOK, issueToResponse(issue))
}

func (s *Server) handleDraft(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID string          `json:"issue_id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	client := beats.NewClient(s.Config)
	client.Collapse = true

	switch model.EventType(req.Type) {
	case model.EventTypeCreate:
		var p model.CreatePayload
		json.Unmarshal(req.Payload, &p)
		issue, err := client.AddIssue(p)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": issue.ID})

	case model.EventTypeUpdate:
		var p model.UpdatePayload
		json.Unmarshal(req.Payload, &p)
		if _, err := client.UpdateIssue(req.IssueID, p, "update"); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	case model.EventTypeComment:
		var p model.CommentPayload
		json.Unmarshal(req.Payload, &p)
		if err := client.AddComment(req.IssueID, p.Text); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	case model.EventTypeDelete:
		var p model.DeletePayload
		json.Unmarshal(req.Payload, &p)
		if err := client.DeleteIssue(req.IssueID, p.Reason); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	default:
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Unknown event type: %s", req.Type))
	}
}

func (s *Server) handleGetPending(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	pending := s.pendingEvents
	s.mu.RUnlock()

	resp := PendingResponse{
		HasPending: len(pending) > 0,
	}

	issueIDSet := make(map[string]bool)
	for _, evt := range pending {
		resp.Events = append(resp.Events, PendingEventEntry{
			IssueID:   evt.ID,
			Type:      string(evt.Type),
			Payload:   evt.Payload,
			CreatedAt: evt.CreatedAt,
		})
		issueIDSet[evt.ID] = true
	}
	for id := range issueIDSet {
		resp.IssueIDs = append(resp.IssueIDs, id)
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	pending := s.pendingEvents
	s.mu.Unlock()

	if len(pending) == 0 {
		respondJSON(w, http.StatusOK, map[string]string{"status": "no_changes"})
		return
	}

	for _, evt := range pending {
		if err := storage.AppendEvent(evt); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
	}

	s.DiscardPending()

	committed := false
	if s.Config.AutoCommit {
		msg := req.Message
		if msg == "" {
			msg = "beats: web UI batch save"
		}
		beats.GitCommit(msg)
		committed = true
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "saved",
		"count":     len(pending),
		"committed": committed,
	})
}

func (s *Server) handleDiscardPending(w http.ResponseWriter, r *http.Request) {
	s.DiscardPending()
	respondJSON(w, http.StatusOK, map[string]string{"status": "discarded"})
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	entries, err := registry.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list instances: %v", err))
		return
	}
	type respEntry struct {
		Name      string `json:"name"`
		Port      int    `json:"port"`
		PID       int    `json:"pid"`
		RootDir   string `json:"root_dir"`
		StartedAt string `json:"started_at"`
		IsCurrent bool   `json:"is_current"`
	}
	out := make([]respEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, respEntry{
			Name:      e.Name,
			Port:      e.Port,
			PID:       e.PID,
			RootDir:   e.RootDir,
			StartedAt: e.StartedAt.Format(time.RFC3339),
			IsCurrent: e.Port == s.Port,
		})
	}
	respondJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	prefix := "beats-"
	if s.Config.Prefix != "" {
		prefix = s.Config.Prefix
	}
	labels := s.Config.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	name := s.Config.Name
	if name == "" {
		if cwd, err := os.Getwd(); err == nil {
			name = filepath.Base(cwd)
		}
	}
	contributors := s.Config.Contributors
	if contributors == nil {
		contributors = []string{}
	}
	resp := map[string]interface{}{
		"auto_commit":         s.Config.AutoCommit,
		"prefix":              prefix,
		"version":             version.CLIVersion,
		"labels":              labels,
		"name":                name,
		"hide_default_labels": s.Config.HideDefaultLabels,
		"default_labels":      s.Config.DefaultLabels,
		"contributors":        contributors,
	}
	if s.Config.Cycles.Enabled {
		resp["cycles"] = map[string]interface{}{
			"enabled":    true,
			"duration":   s.Config.Cycles.Duration,
			"start_day":  s.Config.Cycles.StartDay,
			"anchor_date": s.Config.Cycles.AnchorDate,
		}
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetCycles(w http.ResponseWriter, r *http.Request) {
	if !s.Config.Cycles.Enabled {
		respondJSON(w, http.StatusOK, map[string]interface{}{"enabled": false, "cycles": []struct{}{}})
		return
	}

	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	cycles := s.Config.Cycles.EnumerateCycles(now, 5, 2)
	durationDays := s.Config.Cycles.DurationDays()

	type cycleResponse struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
		Start  string `json:"start"`
		End    string `json:"end"`
		Status string `json:"status"`
		Done   int    `json:"done"`
		Total  int    `json:"total"`
	}

	current := s.Config.Cycles.CurrentCycle()
	var result []cycleResponse
	for _, c := range cycles {
		done, total := 0, 0
		for _, issue := range issues {
			eid := issue.EffectiveCycleID
			if issue.Status == model.StatusDone {
				eid = issue.CycleID
			}
			if eid == c.ID {
				total++
				if issue.Status == model.StatusDone {
					done++
				}
			}
		}

		status := "completed"
		if c.ID == current.ID {
			status = "current"
		} else if c.Start.After(current.End) {
			nextStart := current.Start.AddDate(0, 0, durationDays)
			if c.Start.Equal(nextStart) {
				status = "upcoming"
			} else {
				status = "planned"
			}
		}

		result = append(result, cycleResponse{
			ID:     c.ID,
			Number: c.Number,
			Start:  c.Start.Format("2006-01-02"),
			End:    c.End.Format("2006-01-02"),
			Status: status,
			Done:   done,
			Total:  total,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled": true,
		"cycles":  result,
	})
}

func (s *Server) handleCycleProgress(w http.ResponseWriter, r *http.Request) {
	cycleID := r.PathValue("id")
	if cycleID == "" {
		respondError(w, http.StatusBadRequest, "cycle ID required")
		return
	}
	if !s.Config.Cycles.Enabled {
		respondError(w, http.StatusBadRequest, "cycles not enabled")
		return
	}

	cycle, err := s.Config.Cycles.CycleForID(cycleID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	events, readErr := storage.ReadEvents()
	if readErr != nil {
		respondError(w, http.StatusInternalServerError, readErr.Error())
		return
	}

	s.mu.RLock()
	allEvents := append(events, s.pendingEvents...)
	s.mu.RUnlock()

	endDate := cycle.End
	now := time.Now()
	if now.Before(endDate) {
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}

	type issueState struct {
		cycleID string
		status  string
		deleted bool
	}

	type dayPoint struct {
		Date      string `json:"date"`
		Scope     int    `json:"scope"`
		Started   int    `json:"started"`
		Completed int    `json:"completed"`
	}

	states := make(map[string]*issueState)
	var points []dayPoint
	eventIdx := 0

	for d := cycle.Start; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayEnd := d.AddDate(0, 0, 1)

		for eventIdx < len(allEvents) && allEvents[eventIdx].CreatedAt.Before(dayEnd) {
			evt := allEvents[eventIdx]
			eventIdx++

			switch evt.Type {
			case model.EventTypeCreate:
				b, _ := json.Marshal(evt.Payload)
				var p model.CreatePayload
				json.Unmarshal(b, &p)
				st := p.Status
				if st == "" {
					st = "BACKLOG"
				}
				states[evt.ID] = &issueState{cycleID: p.CycleID, status: st}

			case model.EventTypeUpdate:
				is, ok := states[evt.ID]
				if !ok {
					is = &issueState{}
					states[evt.ID] = is
				}
				b, _ := json.Marshal(evt.Payload)
				var p model.UpdatePayload
				json.Unmarshal(b, &p)
				if p.Status != nil {
					is.status = *p.Status
				}
				if p.CycleID != nil {
					is.cycleID = *p.CycleID
				}

			case model.EventTypeDelete:
				if is, ok := states[evt.ID]; ok {
					is.deleted = true
				}
			}
		}

		scope, started, completed := 0, 0, 0
		for _, is := range states {
			if is.deleted || is.cycleID != cycleID {
				continue
			}
			scope++
			switch model.IssueStatus(is.status) {
			case model.StatusDone:
				completed++
			case model.StatusDoing, model.StatusBlocked:
				started++
			}
		}

		points = append(points, dayPoint{
			Date:      d.Format("2006-01-02"),
			Scope:     scope,
			Started:   started,
			Completed: completed,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"days": points})
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	name, email := getUserNameEmail(s.Config)
	respondJSON(w, http.StatusOK, map[string]string{
		"name":  name,
		"email": email,
	})
}

func (s *Server) handleAddLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Color == "" {
		respondError(w, http.StatusBadRequest, "name and color required")
		return
	}

	if err := config.AddLabel(req.Name, req.Color); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.Config.Labels == nil {
		s.Config.Labels = make(map[string]string)
	}
	s.Config.Labels[req.Name] = req.Color

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleUpdateLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldName string `json:"old_name"`
		NewName string `json:"new_name"`
		Color   string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.OldName == "" || req.NewName == "" || req.Color == "" {
		respondError(w, http.StatusBadRequest, "old_name, new_name, and color required")
		return
	}

	if err := config.UpdateLabel(req.OldName, req.NewName, req.Color); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	delete(s.Config.Labels, req.OldName)
	if s.Config.Labels == nil {
		s.Config.Labels = make(map[string]string)
	}
	s.Config.Labels[req.NewName] = req.Color

	if req.OldName != req.NewName {
		client := beats.NewClient(s.Config)
		client.Collapse = true
		issues, err := s.GetProjectedIssues()
		if err == nil {
			for _, issue := range issues {
				for _, l := range issue.Labels {
					if l == req.OldName {
						newLabels := make([]string, 0, len(issue.Labels))
						for _, ll := range issue.Labels {
							if ll == req.OldName {
								newLabels = append(newLabels, req.NewName)
							} else {
								newLabels = append(newLabels, ll)
							}
						}
						client.UpdateIssue(issue.ID, model.UpdatePayload{Labels: newLabels}, "update")
						break
					}
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name required")
		return
	}

	if err := config.DeleteLabel(req.Name); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	delete(s.Config.Labels, req.Name)

	client := beats.NewClient(s.Config)
	client.Collapse = true
	issues, err := s.GetProjectedIssues()
	if err == nil {
		for _, issue := range issues {
			for _, l := range issue.Labels {
				if l == req.Name {
					newLabels := make([]string, 0, len(issue.Labels))
					for _, ll := range issue.Labels {
						if ll != req.Name {
							newLabels = append(newLabels, ll)
						}
					}
					client.UpdateIssue(issue.ID, model.UpdatePayload{Labels: newLabels}, "update")
					break
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	const maxItems = 30

	events, err := storage.ReadEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.RLock()
	allEvents := append(events, s.pendingEvents...)
	s.mu.RUnlock()

	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	titles := make(map[string]string, len(issues))
	for id, issue := range issues {
		titles[id] = issue.Title
	}

	type activityItem struct {
		IssueID    string      `json:"issue_id"`
		IssueTitle string      `json:"issue_title"`
		Type       string      `json:"type"`
		Payload    interface{} `json:"payload"`
		CreatedAt  time.Time   `json:"created_at"`
		CreatedBy  string      `json:"created_by"`
	}

	out := make([]activityItem, 0, maxItems)
	for i := len(allEvents) - 1; i >= 0 && len(out) < maxItems; i-- {
		evt := allEvents[i]
		if !isMeaningfulActivityEvent(evt) {
			continue
		}
		out = append(out, activityItem{
			IssueID:    evt.ID,
			IssueTitle: titles[evt.ID],
			Type:       string(evt.Type),
			Payload:    evt.Payload,
			CreatedAt:  evt.CreatedAt,
			CreatedBy:  evt.CreatedBy,
		})
	}

	respondJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetIssueHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return
	}

	events, err := storage.ReadEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.mu.RLock()
	allEvents := append(events, s.pendingEvents...)
	s.mu.RUnlock()

	var history []map[string]interface{}
	for _, evt := range allEvents {
		if evt.ID != id {
			continue
		}
		history = append(history, map[string]interface{}{
			"type":       evt.Type,
			"payload":    evt.Payload,
			"created_at": evt.CreatedAt,
			"created_by": evt.CreatedBy,
		})
	}

	respondJSON(w, http.StatusOK, history)
}
