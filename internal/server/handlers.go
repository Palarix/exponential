package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/auth"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/registry"
	"github.com/palarix/exponential/internal/storage"
	"github.com/palarix/exponential/internal/version"
	"golang.org/x/crypto/ssh"
)

func (s *Server) handleGetIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load issues")
		return
	}

	sorted := exponential.SortIssues(issues)

	s.mu.RLock()
	pendingIDs := make(map[string]bool)
	for _, evt := range s.pendingEvents {
		pendingIDs[evt.ID] = true
	}
	s.mu.RUnlock()

	response := make([]IssueResponse, 0, len(sorted))
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
		respondError(w, http.StatusInternalServerError, "failed to load issues")
		return
	}

	issue, exists := issues[id]
	if !exists {
		respondError(w, http.StatusNotFound, fmt.Sprintf("issue %s not found", id))
		return
	}

	respondJSON(w, http.StatusOK, issueToResponse(issue))
}

func (s *Server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	filename := r.PathValue("filename")
	if id == "" || filename == "" {
		respondError(w, http.StatusBadRequest, "issue ID and filename required")
		return
	}

	client := exponential.NewClient(s.Config)
	content, err := client.ReadArtifact(id, filename)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"issue_id": id,
		"filename": filename,
		"content":  content,
	})
}

func (s *Server) handleDraft(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID string          `json:"issue_id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}

	client := exponential.NewClient(s.Config)
	client.Collapse = true
	client.Source = "web"
	if user, ok := auth.UserFromContext(r.Context()); ok {
		client.UserOverride = user.Raw
	}

	switch model.EventType(req.Type) {
	case model.EventTypeCreate:
		if !s.requireCapability(w, r, "issue.create") {
			return
		}
		var p model.CreatePayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid CREATE payload")
			return
		}
		if err := client.ValidateCreatePayload(&p); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		issue, err := client.AddIssue(p)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create issue")
			return
		}
		s.broadcastEvent("CREATE", issue.ID)
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": issue.ID})

	case model.EventTypeUpdate:
		if !s.requireCapability(w, r, "issue.update") {
			return
		}
		if req.IssueID == "" {
			respondError(w, http.StatusBadRequest, "issue_id is required")
			return
		}
		var p model.UpdatePayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid UPDATE payload")
			return
		}
		if err := client.ValidateUpdatePayload(&p); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if _, err := client.UpdateIssue(req.IssueID, p, "update"); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update issue %s", req.IssueID))
			return
		}
		s.broadcastEvent("UPDATE", req.IssueID)
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	case model.EventTypeComment:
		if !s.requireCapability(w, r, "issue.comment") {
			return
		}
		if req.IssueID == "" {
			respondError(w, http.StatusBadRequest, "issue_id is required")
			return
		}
		var p model.CommentPayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid COMMENT payload")
			return
		}
		if p.Text == "" {
			respondError(w, http.StatusBadRequest, "comment text is required")
			return
		}
		if err := client.AddComment(req.IssueID, p.Text); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add comment to issue %s", req.IssueID))
			return
		}
		s.broadcastEvent("COMMENT", req.IssueID)
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	case model.EventTypeDelete:
		if !s.requireCapability(w, r, "issue.delete") {
			return
		}
		if req.IssueID == "" {
			respondError(w, http.StatusBadRequest, "issue_id is required")
			return
		}
		var p model.DeletePayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid DELETE payload")
			return
		}
		if err := client.DeleteIssue(req.IssueID, p.Reason, p.Cascade); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete issue %s", req.IssueID))
			return
		}
		s.broadcastEvent("DELETE", req.IssueID)
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
			respondError(w, http.StatusInternalServerError, "failed to persist pending events")
			return
		}
	}

	s.DiscardPending()

	committed := false
	if s.Config.AutoCommit {
		msg := req.Message
		if msg == "" {
			msg = "xpo: web UI batch save"
		}
		exponential.GitCommit(msg)
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
		respondError(w, http.StatusInternalServerError, "failed to list server instances")
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
	prefix := "issue"
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
		respondError(w, http.StatusInternalServerError, "failed to load issues")
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
	result := make([]cycleResponse, 0)
	for _, c := range cycles {
		done, total := 0, 0
		for _, issue := range issues {
			eid := issue.EffectiveCycleID
			if model.IsTerminal(issue.Status) {
				eid = issue.CycleID
			}
			if eid == c.ID {
				total++
				if model.IsCompleted(issue.Status) {
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
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid cycle ID %q", cycleID))
		return
	}

	allEvents, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load events")
		return
	}

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
	points := make([]dayPoint, 0)
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
			st := model.IssueStatus(is.status)
			switch {
			case model.IsCompleted(st):
				completed++
			case st == model.StatusDoing || st == model.StatusBlocked:
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
	if user, ok := auth.UserFromContext(r.Context()); ok {
		respondJSON(w, http.StatusOK, map[string]string{
			"user":  user.Raw,
			"name":  user.Name,
			"email": user.Email,
		})
		return
	}
	name, email := getUserNameEmail(s.Config)
	respondJSON(w, http.StatusOK, map[string]string{
		"user":  fmt.Sprintf("%s <%s>", name, email),
		"name":  name,
		"email": email,
	})
}

var hexColorRe = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)

func (s *Server) handleAddLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	if req.Name == "" || req.Color == "" {
		respondError(w, http.StatusBadRequest, "name and color required")
		return
	}
	if !hexColorRe.MatchString(req.Color) {
		respondError(w, http.StatusBadRequest, "color must be a 6-digit hex value (e.g. #FF5733 or FF5733)")
		return
	}

	if err := config.AddLabel(req.Name, req.Color); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add label %q", req.Name))
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
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	if req.OldName == "" || req.NewName == "" || req.Color == "" {
		respondError(w, http.StatusBadRequest, "old_name, new_name, and color required")
		return
	}
	if !hexColorRe.MatchString(req.Color) {
		respondError(w, http.StatusBadRequest, "color must be a 6-digit hex value (e.g. #FF5733 or FF5733)")
		return
	}

	if err := config.UpdateLabel(req.OldName, req.NewName, req.Color); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update label %q", req.OldName))
		return
	}

	delete(s.Config.Labels, req.OldName)
	if s.Config.Labels == nil {
		s.Config.Labels = make(map[string]string)
	}
	s.Config.Labels[req.NewName] = req.Color

	if req.OldName != req.NewName {
		client := exponential.NewClient(s.Config)
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
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name required")
		return
	}

	if err := config.DeleteLabel(req.Name); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete label %q", req.Name))
		return
	}

	delete(s.Config.Labels, req.Name)

	client := exponential.NewClient(s.Config)
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

	allEvents, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load activity events")
		return
	}

	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load issues")
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
		OnBehalfOf string      `json:"on_behalf_of,omitempty"`
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
			OnBehalfOf: evt.OnBehalfOf,
		})
	}

	respondJSON(w, http.StatusOK, out)
}

func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	// Resolve the requesting user: prefer the authenticated identity, fall
	// back to the server's configured user when auth is disabled.
	var me string
	if u, ok := auth.UserFromContext(r.Context()); ok {
		me = u.Raw
	} else {
		me = exponential.NewClient(s.Config).GetUser()
	}

	var since time.Time
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid since parameter: expected RFC3339 format (e.g. 2006-01-02T15:04:05Z)")
			return
		}
		since = t
	}

	allEvents, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load events")
		return
	}
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load issues")
		return
	}

	items := exponential.BuildInbox(allEvents, issues, me, since)

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := fmt.Sscanf(raw, "%d", &limit); n != 1 || err != nil || limit < 1 {
			limit = 200
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}

	respondJSON(w, http.StatusOK, items)
}

func (s *Server) handleInboxStatus(w http.ResponseWriter, r *http.Request) {
	var me string
	if u, ok := auth.UserFromContext(r.Context()); ok {
		me = u.Raw
	} else {
		me = exponential.NewClient(s.Config).GetUser()
	}

	remoteURL := config.ReadRemoteURL()
	lastRead := config.GetInboxLastRead(remoteURL)

	allEvents, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load events")
		return
	}
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load issues")
		return
	}

	inboxItems := exponential.BuildInbox(allEvents, issues, me, lastRead)
	seen := make(map[string]bool, len(inboxItems))
	for _, item := range inboxItems {
		seen[item.IssueID] = true
	}
	unread := len(seen)

	resp := struct {
		LastRead string `json:"last_read"`
		Unread   int    `json:"unread"`
	}{Unread: unread}
	if !lastRead.IsZero() {
		resp.LastRead = lastRead.UTC().Format(time.RFC3339)
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleInboxRead(w http.ResponseWriter, r *http.Request) {
	remoteURL := config.ReadRemoteURL()
	now := time.Now()
	if err := config.SetInboxLastRead(remoteURL, now); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update inbox read cursor")
		return
	}
	respondJSON(w, http.StatusOK, struct {
		LastRead string `json:"last_read"`
	}{LastRead: now.UTC().Format(time.RFC3339)})
}

func (s *Server) handleGetIssueHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return
	}

	allEvents, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load issue history")
		return
	}

	history := make([]map[string]interface{}, 0)
	for _, evt := range allEvents {
		if evt.ID != id {
			continue
		}
		entry := map[string]interface{}{
			"type":       evt.Type,
			"payload":    evt.Payload,
			"created_at": evt.CreatedAt,
			"created_by": evt.CreatedBy,
		}
		if evt.OnBehalfOf != "" {
			entry["on_behalf_of"] = evt.OnBehalfOf
		}
		history = append(history, entry)
	}

	respondJSON(w, http.StatusOK, history)
}

func (s *Server) handleStartWork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return
	}

	force := r.URL.Query().Get("force") == "true"

	// Serialize the check-and-transition to prevent claim races.
	s.mu.Lock()
	client := exponential.NewClient(s.Config)
	client.Collapse = true
	branch, _, msgs, err := client.StartWork(id, force)
	s.mu.Unlock()

	if err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	s.broadcastEvent("UPDATE", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"issue_id": id,
		"branch":   branch,
		"messages": msgs,
	})
}

func (s *Server) resolveIssueBranch(w http.ResponseWriter, r *http.Request) (*model.Issue, string, bool) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return nil, "", false
	}

	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load issues")
		return nil, "", false
	}

	issue, exists := issues[id]
	if !exists {
		respondError(w, http.StatusNotFound, fmt.Sprintf("issue %s not found", id))
		return nil, "", false
	}

	// Fill local branch stats if no remote branch detected
	if issue.BranchStats == nil {
		client := exponential.NewClient(s.Config)
		client.FillLocalBranchStats(issue)
	}

	if issue.BranchStats == nil {
		respondError(w, http.StatusNotFound, "no branch found for this issue")
		return nil, "", false
	}

	base := exponential.DefaultBranch()
	return issue, base, true
}

func (s *Server) handleGetIssueCommits(w http.ResponseWriter, r *http.Request) {
	issue, base, ok := s.resolveIssueBranch(w, r)
	if !ok {
		return
	}

	commits := exponential.ListBranchCommitsDetailed(issue.BranchStats.Branch, base)
	respondJSON(w, http.StatusOK, commits)
}

func (s *Server) handleGetCommitDiff(w http.ResponseWriter, r *http.Request) {
	sha := r.PathValue("sha")
	if sha == "" {
		respondError(w, http.StatusBadRequest, "commit SHA required")
		return
	}
	diff := exponential.GetCommitDiffText(sha)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(diff))
}

func (s *Server) handleGetIssueFiles(w http.ResponseWriter, r *http.Request) {
	issue, base, ok := s.resolveIssueBranch(w, r)
	if !ok {
		return
	}

	var files []exponential.FileStat
	if r.URL.Query().Get("scope") == "uncommitted" {
		dir, found := resolveWorkingDir(issue.BranchStats.Branch)
		if !found {
			respondError(w, http.StatusNotFound, "no working directory for branch")
			return
		}
		files = exponential.ListWorkingTreeAllFilesChanged(dir)
	} else {
		files = exponential.ListFilesChanged(issue.BranchStats.Branch, base)
	}
	type fileJSON struct {
		Status     string `json:"status"`
		Path       string `json:"path"`
		Insertions int    `json:"insertions"`
		Deletions  int    `json:"deletions"`
	}
	out := make([]fileJSON, len(files))
	for i, f := range files {
		out[i] = fileJSON{Status: f.Status, Path: f.Path, Insertions: f.Insertions, Deletions: f.Deletions}
	}
	respondJSON(w, http.StatusOK, out)
}

func resolveWorkingDir(branch string) (string, bool) {
	if dir, ok := exponential.FindWorktreeForBranch(branch); ok {
		return dir, true
	}
	if branch == exponential.CurrentBranch() {
		return "", true
	}
	return "", false
}

func (s *Server) handleGetIssueDiff(w http.ResponseWriter, r *http.Request) {
	issue, base, ok := s.resolveIssueBranch(w, r)
	if !ok {
		return
	}

	var diff string
	if r.URL.Query().Get("scope") == "uncommitted" {
		dir, found := resolveWorkingDir(issue.BranchStats.Branch)
		if !found {
			respondError(w, http.StatusNotFound, "no working directory for branch")
			return
		}
		diff = exponential.GetWorkingTreeFullDiffText(dir)
	} else {
		diff = exponential.GetDiffText(issue.BranchStats.Branch, base)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(diff))
}

func (s *Server) handleMergeability(w http.ResponseWriter, r *http.Request) {
	issue, _, ok := s.resolveIssueBranch(w, r)
	if !ok {
		return
	}

	type blocker struct {
		Message string   `json:"message"`
		Files   []string `json:"files,omitempty"`
	}

	blockers := make([]blocker, 0)
	warnings := make([]string, 0)

	conflicts, err := exponential.CheckMergeConflicts(issue.BranchStats.Branch)
	if err != nil {
		blockers = append(blockers, blocker{Message: "Failed to check merge conflicts"})
	} else if len(conflicts) > 0 {
		blockers = append(blockers, blocker{
			Message: fmt.Sprintf("%d merge %s with %s", len(conflicts), pluralize(len(conflicts), "conflict", "conflicts"), exponential.DefaultBranch()),
			Files:   conflicts,
		})
	}

	wtDirty := exponential.WorktreeDirtyFiles(issue.BranchStats.Branch)
	if len(wtDirty) > 0 {
		blockers = append(blockers, blocker{
			Message: fmt.Sprintf("%d uncommitted %s in worktree", len(wtDirty), pluralize(len(wtDirty), "file", "files")),
			Files:   wtDirty,
		})
	}

	if dirty := exponential.HubDirtyTrackedFiles(); len(dirty) > 0 {
		warnings = append(warnings, fmt.Sprintf("uncommitted changes: %s — commit before merging", strings.Join(dirty, ", ")))
	}

	dirtyFiles := wtDirty
	if dirtyFiles == nil {
		dirtyFiles = []string{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"can_merge":   len(blockers) == 0,
		"blockers":    blockers,
		"warnings":    warnings,
		"dirty_files": dirtyFiles,
	})
}

func (s *Server) handleMergeIssue(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return
	}

	var body struct {
		Strategy      string `json:"strategy"`
		CommitMessage string `json:"commit_message"`
		KeepBranch    bool   `json:"keep_branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}

	strategy := exponential.MergeStrategySquash
	switch body.Strategy {
	case "", "squash":
		// default
	case "merge":
		strategy = exponential.MergeStrategyMerge
	case "ff":
		strategy = exponential.MergeStrategyFF
	default:
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid merge strategy %q: must be one of squash, merge, ff", body.Strategy))
		return
	}

	client := exponential.NewClient(s.Config)
	client.Collapse = true

	issue, err := client.ResolveReviewIssue(id)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("issue %s not found or has no branch", id))
		return
	}
	if issue.BranchStats == nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("no branch found for %s", id))
		return
	}

	conflicts, mergeErr := exponential.CheckMergeConflicts(issue.BranchStats.Branch)
	if mergeErr != nil {
		respondError(w, http.StatusInternalServerError, mergeErr.Error())
		return
	}
	if len(conflicts) > 0 {
		msgs := make([]string, len(conflicts))
		for i, f := range conflicts {
			msgs[i] = fmt.Sprintf("merge conflict in %s", f)
		}
		respondError(w, http.StatusConflict, strings.Join(msgs, "; "))
		return
	}
	if mergeErr = exponential.HubRequireCleanTree(); mergeErr != nil {
		respondError(w, http.StatusConflict, mergeErr.Error())
		return
	}
	if mergeErr = exponential.WorktreeRequireClean(issue.BranchStats.Branch); mergeErr != nil {
		respondError(w, http.StatusConflict, mergeErr.Error())
		return
	}

	result, err := client.MergeIssue(issue.ID, exponential.MergeOptions{
		Strategy:      strategy,
		CommitMessage: body.CommitMessage,
		DeleteBranch:  !body.KeepBranch,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to merge issue %s", id))
		return
	}

	s.broadcastEvent("MERGE", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"merge_sha": result.MergeSHA,
		"messages":  result.Messages,
	})
}

// --- Auth handlers ---

func (s *Server) handleAuthChallenge(w http.ResponseWriter, r *http.Request) {
	if s.NonceStore == nil {
		respondError(w, http.StatusServiceUnavailable, "auth not enabled")
		return
	}
	nonce, err := s.NonceStore.Generate()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate challenge")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"nonce": nonce})
}

func (s *Server) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if s.NonceStore == nil || s.AuthorizedKeys == nil {
		respondError(w, http.StatusServiceUnavailable, "auth not enabled")
		return
	}

	var req struct {
		PublicKey string `json:"public_key"`
		Signature string `json:"signature"`
		Nonce     string `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}

	if !s.NonceStore.Validate(req.Nonce) {
		respondError(w, http.StatusUnauthorized, "invalid or expired nonce")
		return
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid public key encoding")
		return
	}

	sshPubKey, err := ssh.ParsePublicKey(pubKeyBytes)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid public key")
		return
	}

	identity, ok := s.AuthorizedKeys.Lookup(sshPubKey)
	if !ok {
		respondError(w, http.StatusUnauthorized, "public key not authorized")
		return
	}

	sigBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid signature encoding")
		return
	}

	sig := new(ssh.Signature)
	if err := ssh.Unmarshal(sigBytes, sig); err != nil {
		respondError(w, http.StatusBadRequest, "invalid signature format")
		return
	}

	if err := sshPubKey.Verify([]byte(req.Nonce), sig); err != nil {
		respondError(w, http.StatusUnauthorized, "signature verification failed")
		return
	}

	expiry := time.Now().Add(7 * 24 * time.Hour)
	claims := auth.Claims{
		Sub: identity.Raw,
		Exp: expiry.Unix(),
		Iat: time.Now().Unix(),
	}

	token, err := auth.SignToken(s.SigningKey, claims)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to sign token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"token":      token,
		"expires_at": expiry.Format(time.RFC3339),
	})
}

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	kindFilter := r.URL.Query().Get("kind")

	allEvents, err := s.GetAllEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load events")
		return
	}
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load issues")
		return
	}

	timeline := exponential.BuildTimeline(allEvents, issues, limit, kindFilter)
	respondJSON(w, http.StatusOK, timeline)
}

func (s *Server) handleCommitDetail(w http.ResponseWriter, r *http.Request) {
	sha := r.PathValue("sha")
	if sha == "" {
		respondError(w, http.StatusBadRequest, "commit SHA required")
		return
	}
	detail := exponential.GetCommitDetail(sha)
	if detail == nil {
		respondError(w, http.StatusNotFound, "commit not found")
		return
	}
	respondJSON(w, http.StatusOK, detail)
}
