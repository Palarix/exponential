package model

import (
	"encoding/json"
	"sort"
	"time"
)

type IssueStatus string

const (
	StatusBacklog IssueStatus = "BACKLOG"
	StatusPlanned IssueStatus = "PLANNED"
	StatusDoing   IssueStatus = "DOING"
	StatusBlocked IssueStatus = "BLOCKED"
	StatusDone    IssueStatus = "DONE"
)

type Issue struct {
	ID          string
	Kind        string
	Title       string
	Description string
	Status      IssueStatus
	ParentID    string
	Estimate    int
	Burned      int
	BlockedBy   string
	BlockReason string
	Deleted     bool
	CreatedAt   time.Time
	CreatedBy   string
	UpdatedAt   time.Time
	Events      []Event
}

func ProjectIssues(events []Event) map[string]*Issue {
	issues := make(map[string]*Issue)

	for _, evt := range events {
		switch evt.Type {
		case EventTypeCreate:
			// Unmarshal payload
			// We need a way to look at payload as CreatePayload
			// Since we stored it as interface{}, we need to re-marshal/unmarshal or map it.
			// Ideally storage reads it into correct types, but plain json decoder might give map[string]interface{}

			// Quick hack: encode release -> decode
			payloadBytes, _ := json.Marshal(evt.Payload)
			var p CreatePayload
			json.Unmarshal(payloadBytes, &p)

			issues[evt.ID] = &Issue{
				ID:          evt.ID,
				Kind:        p.Kind,
				Title:       p.Title,
				Description: p.Description,
				ParentID:    p.ParentID,
				Estimate:    p.Estimate,
				Status:      StatusBacklog, // Default
				CreatedAt:   evt.CreatedAt,
				CreatedBy:   evt.CreatedBy,
				UpdatedAt:   evt.CreatedAt,
				Events:      []Event{evt},
			}

		case EventTypeUpdate:
			issue, exists := issues[evt.ID]
			if !exists {
				continue // Should not happen if log is consistent
			}

			payloadBytes, _ := json.Marshal(evt.Payload)
			var p UpdatePayload
			json.Unmarshal(payloadBytes, &p)

			if p.Title != nil {
				issue.Title = *p.Title
			}
			if p.Description != nil {
				issue.Description = *p.Description
			}
			if p.Status != nil {
				issue.Status = IssueStatus(*p.Status)
			}
			if p.ParentID != nil {
				issue.ParentID = *p.ParentID
			}
			if p.Estimate != nil {
				issue.Estimate = *p.Estimate
			}
			if p.BlockedBy != nil {
				issue.BlockedBy = *p.BlockedBy
			}
			if p.BlockReason != nil {
				issue.BlockReason = *p.BlockReason
			}

			issue.UpdatedAt = evt.CreatedAt
			issue.Events = append(issue.Events, evt)

		case EventTypeWorkLog:
			issue, exists := issues[evt.ID]
			if !exists {
				continue
			}

			payloadBytes, _ := json.Marshal(evt.Payload)
			var p WorkLogPayload
			json.Unmarshal(payloadBytes, &p)

			issue.Burned += p.Amount
			issue.UpdatedAt = evt.CreatedAt
			issue.Events = append(issue.Events, evt)

		case EventTypeDelete:
			issue, exists := issues[evt.ID]
			if !exists {
				continue
			}

			issue.Deleted = true
			issue.UpdatedAt = evt.CreatedAt
			issue.Events = append(issue.Events, evt)
		}
	}

	// Filter out deleted issues
	for id, issue := range issues {
		if issue.Deleted {
			delete(issues, id)
		}
	}

	return issues
}

func SortIssues(issues map[string]*Issue) []*Issue {
	// 1. Group issues by root
	// Map: RootID -> List of Issues in that group
	groups := make(map[string][]*Issue)

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
	list := make([]*Issue, 0, len(issues))
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
