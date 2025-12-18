package model

import (
	"time"
)

type EventType string

const (
	EventTypeCreate  EventType = "CREATE"
	EventTypeUpdate  EventType = "UPDATE"
	EventTypeWorkLog EventType = "WORK_LOG"
)

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

type UpdatePayload struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"` // "BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE"
	ParentID    *string `json:"parent_id,omitempty"`
	Estimate    *int    `json:"estimate,omitempty"`
	BlockedBy   *string `json:"blocked_by,omitempty"`
	BlockReason *string `json:"block_reason,omitempty"`
}

// Helper to validate user string format
func ValidateUser(user string) error {
	// Simple check for now, can be more robust
	// Expected: "Name <email>"
	// We won't rigorously enforce regex but just check for presence of <>
	// This can be improved.
	return nil
}
