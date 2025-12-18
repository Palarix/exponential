package model

import (
	"fmt"
	"regexp"
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

// ValidateUser validates that a user string matches the expected format "Name <email>".
func ValidateUser(user string) error {
	// Pattern: "One or more chars" followed by space(s), then "<email@domain>"
	re := regexp.MustCompile(`^.+\s+<[^<>]+@[^<>]+>$`)
	if !re.MatchString(user) {
		return fmt.Errorf("invalid user format: expected 'Name <email>', got %q", user)
	}
	return nil
}
