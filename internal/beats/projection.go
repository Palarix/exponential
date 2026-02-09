package beats

import (
	"encoding/json"
	"sort"

	"github.com/palarix/beats/internal/model"
)

func ProjectIssues(events []model.Event) map[string]*model.Issue {
	issues := make(map[string]*model.Issue)

	// Pass 1: Process Events
	for _, evt := range events {
		switch evt.Type {
		case model.EventTypeCreate:
			// Unmarshal payload
			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.CreatePayload
			json.Unmarshal(payloadBytes, &p)

			issue := &model.Issue{
				ID:           evt.ID,
				Kind:         p.Kind,
				Title:        p.Title,
				Description:  p.Description,
				Estimate:     p.Estimate,
				Status:       model.StatusBacklog, // Default
				CreatedAt:    evt.CreatedAt,
				CreatedBy:    evt.CreatedBy,
				UpdatedAt:    evt.CreatedAt,
				Events:       []model.Event{evt},
				Checklist:    p.Checklist,
				Dependencies: p.Dependencies,
				Labels:       p.Labels,
			}

			// Backward Compatibility: ParentID from payload -> Dependency
			if p.ParentID != "" {
				// Deduplicate
				exists := false
				for _, d := range issue.Dependencies {
					if d.Kind == model.DependencyChild && d.TargetID == p.ParentID {
						exists = true
						break
					}
				}
				if !exists {
					issue.Dependencies = append(issue.Dependencies, model.Dependency{
						SourceID: issue.ID,
						TargetID: p.ParentID,
						Kind:     model.DependencyChild,
					})
				}
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

			// Lists are replaced if provided, logic could vary (merge vs replace)
			// For now, assuming replace for simplicity and consistency with REST semantics
			if p.Checklist != nil {
				issue.Checklist = p.Checklist
			}
			if p.Dependencies != nil {
				issue.Dependencies = p.Dependencies
			}
			if p.Labels != nil {
				issue.Labels = p.Labels
			}

			// Backward Compatibility: Updates to ParentID, BlockedBy
			if p.ParentID != nil {
				// Deduplicate
				exists := false
				for _, d := range issue.Dependencies {
					if d.Kind == model.DependencyChild && d.TargetID == *p.ParentID {
						exists = true
						break
					}
				}
				if !exists {
					issue.Dependencies = append(issue.Dependencies, model.Dependency{
						SourceID: issue.ID,
						TargetID: *p.ParentID,
						Kind:     model.DependencyChild,
					})
				}
			}
			if p.BlockedBy != nil && *p.BlockedBy != "" {
				issue.Dependencies = append(issue.Dependencies, model.Dependency{
					SourceID: issue.ID,
					TargetID: *p.BlockedBy,
					Kind:     model.DependencyBlockedBy,
				})
			}
			// BlockReason is just text, maybe attach to the dependency?
			// Ignoring for now as it doesn't map cleanly to structure without ID.
			// Ideally BlockReason should be part of the Dependency struct/metadata.
			if p.BlockReason != nil {
				issue.BlockReason = *p.BlockReason // Keep it on struct for now
			}

			issue.UpdatedAt = evt.CreatedAt
			issue.Events = append(issue.Events, evt)

		case model.EventTypeWorkLog:
			issue, exists := issues[evt.ID]
			if !exists {
				continue
			}

			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.WorkLogPayload
			json.Unmarshal(payloadBytes, &p)

			issue.LoggedEffort += p.Amount
			issue.Burned = issue.LoggedEffort // Sync deprecated field
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

	// Filter out deleted issues explicitly before Pass 2?
	// Or keep them for referential integrity?
	// The original code filtered them at the end.
	// Let's filter at the end, but check for deletion in loops.

	// Pass 2: Derive State & Relationships
	for _, issue := range issues {
		if issue.Deleted {
			continue
		}

		// Derive ParentID & BlockedBy from Dependencies
		for _, dep := range issue.Dependencies {
			if dep.Kind == model.DependencyChild {
				issue.ParentID = dep.TargetID
			}
			if dep.Kind == model.DependencyBlockedBy {
				issue.BlockedBy = dep.TargetID // Simple "last one wins" for UI compat
			}
		}
	}

	// Pass 3: Derive Epic State (requires all children to be processed)
	// We need to iterate again or do a graph traversal.
	// Simple iteration over all issues is fine for now.
	for _, issue := range issues {
		if issue.Kind == "EPIC" && !issue.Deleted {
			// Find children
			var children []*model.Issue
			for _, potentialChild := range issues {
				if !potentialChild.Deleted && potentialChild.ParentID == issue.ID {
					children = append(children, potentialChild)
				}
			}

			if len(children) > 0 {
				allDone := true
				anyDoing := false
				totalEstimate := 0

				for _, child := range children {
					totalEstimate += child.Estimate
					if child.Status != model.StatusDone {
						allDone = false
					}
					if child.Status == model.StatusDoing || child.Status == model.StatusPlanned {
						anyDoing = true
					}
				}

				issue.Estimate = totalEstimate

				// State Derivation Logic:
				// If manually set to DONE, keep it?
				// Design doc says: "Epic status... is derived".
				// "An Epic is IN PROGRESS if any child is PLANNED or DOING"
				// "Only DONE when all children are DONE"

				if allDone {
					// We could auto-mark done, but doc says "prompt".
					// However, for the "State" field on the struct, it should probably reflect reality.
					// Let's make it DONE if all children are DONE.
					issue.Status = model.StatusDone
				} else if anyDoing {
					issue.Status = model.StatusDoing
				} else {
					// If children exist but none are doing/done, likely Backlog/Planned
					// We leave it as is (default or manually set) OR force to PLANNED?
					// Let's leave it unless we want to enforce.
					// A safe bet is: if any child is Doing, Epic is Doing.
				}
			}
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

func SortIssues(issues map[string]*model.Issue) []*model.Issue {
	// 1. Group issues by root
	// Map: RootID -> List of Issues in that group
	groups := make(map[string][]*model.Issue)

	for _, i := range issues {
		rootID := i.ID
		if i.ParentID != "" {
			// If parent exists in our map, use it as root
			if _, ok := issues[i.ParentID]; ok {
				rootID = i.ParentID
			}
			// If parent doesn't exist (orphan), treating as its own root for now
		}
		groups[rootID] = append(groups[rootID], i)
	}

	// 2. Sort the Roots
	var roots []string
	for rootID := range groups {
		roots = append(roots, rootID)
	}

	sort.Slice(roots, func(i, j int) bool {
		// Sort groups by the Root Issue's CreatedAt (Chronological)
		rootI := issues[roots[i]]
		rootJ := issues[roots[j]]
		return rootI.CreatedAt.Before(rootJ.CreatedAt)
	})

	// 3. Flatten
	list := make([]*model.Issue, 0, len(issues))
	for _, rootID := range roots {
		groupIssues := groups[rootID]

		// Sort within group: Parent first, then Children by CreatedAt
		sort.Slice(groupIssues, func(i, j int) bool {
			a := groupIssues[i]
			b := groupIssues[j]

			// Parent always comes first
			if a.ID == rootID {
				return true
			}
			if b.ID == rootID {
				return false
			}

			// Both are children, sort by CreatedAt
			return a.CreatedAt.Before(b.CreatedAt)
		})

		list = append(list, groupIssues...)
	}

	return list
}
