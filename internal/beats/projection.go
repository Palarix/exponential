package beats

import (
	"encoding/json"
	"sort"

	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
)

func ProjectIssues(events []model.Event) map[string]*model.Issue {
	issues := make(map[string]*model.Issue)

	// Pass 1: Process Events
	for _, evt := range events {
		switch evt.Type {
		case model.EventTypeCreate:
			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.CreatePayload
			json.Unmarshal(payloadBytes, &p)

			status := model.IssueStatus(p.Status)
			if status == "" {
				status = model.StatusBacklog
			}

			issue := &model.Issue{
				ID:           evt.ID,
				Title:        p.Title,
				Description:  p.Description,
				ParentID:     p.ParentID,
				Estimate:     p.Estimate,
				Priority:     p.Priority,
				SortOrder:    p.SortOrder,
				Assignee:     p.Assignee,
				Status:       status,
				CreatedAt:    evt.CreatedAt,
				CreatedBy:    evt.CreatedBy,
				UpdatedAt:    evt.CreatedAt,
				Events:       []model.Event{evt},
				Dependencies: p.Dependencies,
				Labels:       p.Labels,
			}
			issues[evt.ID] = issue

		case model.EventTypeUpdate:
			issue, exists := issues[evt.ID]
			if !exists {
				continue
			}

			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.UpdatePayload
			json.Unmarshal(payloadBytes, &p)

			if p.Title != nil {
				issue.Title = *p.Title
			}
			if p.Description != nil {
				issue.Description = *p.Description
			}
			if p.Status != nil {
				issue.Status = model.IssueStatus(*p.Status)
			}
			if p.Estimate != nil {
				issue.Estimate = *p.Estimate
			}
			if p.Priority != nil {
				issue.Priority = *p.Priority
			}
			if p.SortOrder != nil {
				issue.SortOrder = *p.SortOrder
			}
			if p.ParentID != nil {
				issue.ParentID = *p.ParentID
			}
			if p.Assignee != nil {
				issue.Assignee = *p.Assignee
			}
			if p.Dependencies != nil {
				issue.Dependencies = p.Dependencies
			}
			if p.Labels != nil {
				issue.Labels = p.Labels
			}

			issue.UpdatedAt = evt.CreatedAt
			issue.Events = append(issue.Events, evt)

		case model.EventTypeDelete:
			issue, exists := issues[evt.ID]
			if !exists {
				continue
			}
			issue.Deleted = true
			issue.UpdatedAt = evt.CreatedAt
			issue.Events = append(issue.Events, evt)

		case model.EventTypeComment:
			issue, exists := issues[evt.ID]
			if !exists {
				continue
			}

			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.CommentPayload
			json.Unmarshal(payloadBytes, &p)

			comment := model.Comment{
				ID:        p.ID,
				Text:      p.Text,
				CreatedBy: evt.CreatedBy,
				CreatedAt: evt.CreatedAt,
			}
			issue.Comments = append(issue.Comments, comment)
			issue.UpdatedAt = evt.CreatedAt
			issue.Events = append(issue.Events, evt)
		}
	}

	// Filter out deleted issues
	finalIssues := make(map[string]*model.Issue)
	for id, issue := range issues {
		if !issue.Deleted {
			finalIssues[id] = issue
		}
	}

	return finalIssues
}

// ProjectIssuesWithConfig projects issues and applies config-driven automations.
func ProjectIssuesWithConfig(events []model.Event, cfg *config.Config) map[string]*model.Issue {
	issues := ProjectIssues(events)

	if cfg == nil {
		return issues
	}

	// Apply automations as derived state
	applyAutomations(issues, cfg)

	return issues
}

// applyAutomations applies config-driven automation rules as derived state.
func applyAutomations(issues map[string]*model.Issue, cfg *config.Config) {
	if cfg.Automations.AutoCompleteParent {
		// Auto-complete parent when all children are done
		for _, issue := range issues {
			if issue.ParentID == "" {
				continue
			}
			// Find parent
			parent, ok := issues[issue.ParentID]
			if !ok || parent.Status == model.StatusDone {
				continue
			}
			// Check if all children of this parent are done
			allDone := true
			hasChildren := false
			for _, child := range issues {
				if child.ParentID == parent.ID {
					hasChildren = true
					if child.Status != model.StatusDone {
						allDone = false
						break
					}
				}
			}
			if hasChildren && allDone {
				parent.Status = model.StatusDone
			}
		}
	}

	if cfg.Automations.AutoProgressParent {
		// Auto-progress parent when a sub-issue is progressed
		for _, issue := range issues {
			if issue.ParentID == "" {
				continue
			}
			parent, ok := issues[issue.ParentID]
			if !ok {
				continue
			}
			// If child is DOING/PLANNED and parent is BACKLOG, progress parent
			if (issue.Status == model.StatusDoing || issue.Status == model.StatusPlanned) &&
				parent.Status == model.StatusBacklog {
				parent.Status = model.StatusPlanned
			}
			if issue.Status == model.StatusDoing && parent.Status == model.StatusPlanned {
				parent.Status = model.StatusDoing
			}
		}
	}

	// Note: auto_close_sub_issues and auto_progress_sub_issues are applied at event-time
	// in UpdateIssue, not at projection time, because they generate actual events.

	// Aggregate parent estimates from children
	for _, issue := range issues {
		if issue.ParentID != "" {
			continue // Only aggregate for potential parents
		}
		// Check if this issue has children
		totalEstimate := 0
		hasChildren := false
		for _, child := range issues {
			if child.ParentID == issue.ID {
				hasChildren = true
				est := child.Estimate
				if est == 0 && cfg.CountUnestimated {
					est = 1
				}
				totalEstimate += est
			}
		}
		if hasChildren {
			issue.Estimate = totalEstimate
		}
	}
}

func SortIssues(issues map[string]*model.Issue) []*model.Issue {
	// 1. Group issues by root
	groups := make(map[string][]*model.Issue)

	for _, i := range issues {
		rootID := i.ID
		if i.ParentID != "" {
			if _, ok := issues[i.ParentID]; ok {
				rootID = i.ParentID
			}
		}
		groups[rootID] = append(groups[rootID], i)
	}

	// 2. Sort the Roots
	var roots []string
	for rootID := range groups {
		roots = append(roots, rootID)
	}

	sort.Slice(roots, func(i, j int) bool {
		rootI := issues[roots[i]]
		rootJ := issues[roots[j]]
		return rootI.CreatedAt.Before(rootJ.CreatedAt)
	})

	// 3. Flatten
	list := make([]*model.Issue, 0, len(issues))
	for _, rootID := range roots {
		groupIssues := groups[rootID]

		sort.Slice(groupIssues, func(i, j int) bool {
			a := groupIssues[i]
			b := groupIssues[j]

			if a.ID == rootID {
				return true
			}
			if b.ID == rootID {
				return false
			}

			return a.CreatedAt.Before(b.CreatedAt)
		})

		list = append(list, groupIssues...)
	}

	return list
}
