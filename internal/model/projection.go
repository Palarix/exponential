package model

import (
	"encoding/json"
	"sort"
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
	BlockedBy   string
	BlockReason string
	CreatedAt   string
	CreatedBy   string
	UpdatedAt   string
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
				CreatedAt:   evt.CreatedAt.Format("2006-01-02 15:04"),
				CreatedBy:   evt.CreatedBy,
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

			issue.Events = append(issue.Events, evt)
		}
	}
	return issues
}

func SortIssues(issues map[string]*Issue) []*Issue {
	list := make([]*Issue, 0, len(issues))
	for _, i := range issues {
		list = append(list, i)
	}
	// Sort by CreatedAt ?? Or just random?
	// Let's sort by ID for now or CreatedAt string
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt < list[j].CreatedAt
	})
	return list
}
