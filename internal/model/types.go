package model

import (
	"time"
)

type EventType string

const (
	EventTypeCreate  EventType = "CREATE"
	EventTypeUpdate  EventType = "UPDATE"
	EventTypeWorkLog EventType = "WORK_LOG"
	EventTypeDelete  EventType = "DELETE"
)

type DeletePayload struct {
	Reason string `json:"reason,omitempty"`
}

type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
	CreatedBy string      `json:"created_by"` // Format: "First Last <email>"
}

type CreatePayload struct {
	Kind        string `json:"kind"` // "TASK", "EPIC", or "BUG"
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	Estimate    int    `json:"estimate,omitempty"`
}

type WorkLogPayload struct {
	Amount int `json:"amount"`
}

type UpdatePayload struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"` // "BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE"
	ParentID    *string `json:"parent_id,omitempty"`
	Estimate    *int    `json:"estimate,omitempty"`
	BlockedBy   *string `json:"blocked_by,omitempty"`
	BlockReason *string `json:"block_reason,omitempty"`
}

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
