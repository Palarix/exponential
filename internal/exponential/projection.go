package exponential

import (
	"encoding/json"
	"sort"

	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/sortorder"
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
				CycleID:      p.CycleID,
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
			if p.CycleID != nil {
				issue.CycleID = *p.CycleID
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

		case model.EventTypeMerge:
			issue, exists := issues[evt.ID]
			if !exists {
				continue
			}
			issue.Status = model.StatusDone
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

	backfillSortOrder(finalIssues)

	return finalIssues
}

// backfillSortOrder assigns sort_order to issues that don't have one.
// Unkeyed issues are appended after the global max sort_order across all
// issues, in creation-time order.
func backfillSortOrder(issues map[string]*model.Issue) {
	maxKey := ""
	for _, issue := range issues {
		if issue.SortOrder != "" && issue.SortOrder > maxKey {
			maxKey = issue.SortOrder
		}
	}

	var unkeyed []*model.Issue
	for _, issue := range issues {
		if issue.SortOrder == "" {
			unkeyed = append(unkeyed, issue)
		}
	}
	if len(unkeyed) == 0 {
		return
	}

	sort.Slice(unkeyed, func(i, j int) bool {
		if !unkeyed[i].CreatedAt.Equal(unkeyed[j].CreatedAt) {
			return unkeyed[i].CreatedAt.Before(unkeyed[j].CreatedAt)
		}
		return unkeyed[i].ID < unkeyed[j].ID
	})

	keys, err := sortorder.GenerateNKeysBetween(maxKey, "", len(unkeyed))
	if err != nil {
		return
	}
	for i, issue := range unkeyed {
		issue.SortOrder = keys[i]
	}
}

// ProjectIssuesWithConfig projects issues and applies config-driven automations.
func ProjectIssuesWithConfig(events []model.Event, cfg *config.Config) map[string]*model.Issue {
	issues := ProjectIssues(events)

	if cfg == nil {
		return issues
	}

	// Apply automations as derived state
	applyAutomations(issues, cfg)

	// Compute effective cycle IDs (rollover)
	if cfg.Cycles.Enabled {
		applyCycleRollover(issues, cfg.Cycles, time.Now())
	}

	// Infer DOING status from remote git branches
	applyBranchInference(issues)

	return issues
}

// applyCycleRollover sets EffectiveCycleID on every issue.
// Done issues keep their original cycle. Not-done issues in a past cycle
// roll forward to the current cycle. Issues with a current or future cycle
// (or no cycle) are unchanged.
func applyCycleRollover(issues map[string]*model.Issue, cc config.CycleConfig, now time.Time) {
	current := cc.CycleForDate(now)
	for _, issue := range issues {
		if issue.CycleID == "" {
			continue
		}
		if issue.Status == model.StatusDone {
			issue.EffectiveCycleID = issue.CycleID
		} else if cc.IsPastCycle(issue.CycleID, now) {
			issue.EffectiveCycleID = current.ID
		} else {
			issue.EffectiveCycleID = issue.CycleID
		}
	}
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
				parent.InferredStatus = true
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
				parent.InferredStatus = true
			}
			if issue.Status == model.StatusDoing && parent.Status == model.StatusPlanned {
				parent.Status = model.StatusDoing
				parent.InferredStatus = true
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
