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
	EventTypeComment EventType = "COMMENT"
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
	Kind         string          `json:"kind"`
	Title        string          `json:"title"`
	Description  string          `json:"description,omitempty"`
	ParentID     string          `json:"parent_id,omitempty"` // Kept for backward compat / convenience
	Estimate     int             `json:"estimate,omitempty"`
	Checklist    []ChecklistItem `json:"checklist,omitempty"`
	Dependencies []Dependency    `json:"dependencies,omitempty"`
	Labels       []string        `json:"labels,omitempty"`
}

type WorkLogPayload struct {
	Amount int `json:"amount"`
}

type CommentPayload struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Comment struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdatePayload struct {
	Title        *string         `json:"title,omitempty"`
	Description  *string         `json:"description,omitempty"`
	Status       *string         `json:"status,omitempty"`
	ParentID     *string         `json:"parent_id,omitempty"`
	Estimate     *int            `json:"estimate,omitempty"`
	Checklist    []ChecklistItem `json:"checklist,omitempty"`
	Dependencies []Dependency    `json:"dependencies,omitempty"`
	Labels       []string        `json:"labels,omitempty"`

	// Deprecated fields, kept for parsing old events if needed,
	// but generally we should migrate away from them.
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
	ID           string
	Kind         string
	Title        string
	Description  string
	Status       IssueStatus
	ParentID     string // Derived from Dependencies
	Estimate     int
	LoggedEffort int // Replaces Burned
	Burned       int // Deprecated: use LoggedEffort
	Locked       bool
	Deleted      bool

	// Derived / Compatibility Fields
	BlockedBy   string // Derived from Dependencies
	BlockReason string // Derived from Dependencies

	// New Metadata
	Checklist    []ChecklistItem
	Dependencies []Dependency
	Labels       []string

	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	Events    []Event
	Comments  []Comment
}

type ChecklistItem struct {
	Title string `json:"title"`
	State string `json:"state"` // "open", "done"
}

type DependencyKind string

const (
	DependencyBlocks       DependencyKind = "blocks"
	DependencyBlockedBy    DependencyKind = "blocked_by"
	DependencyPrecedes     DependencyKind = "precedes"
	DependencyFollows      DependencyKind = "follows"
	DependencyParent       DependencyKind = "parent"
	DependencyChild        DependencyKind = "child"
	DependencyRelatesTo    DependencyKind = "relates_to"
	DependencyDuplicates   DependencyKind = "duplicates"
	DependencyDuplicatedBy DependencyKind = "duplicated_by"
	DependencyCauses       DependencyKind = "causes"
	DependencyCausedBy     DependencyKind = "caused_by"
	DependencyFixes        DependencyKind = "fixes"
	DependencyFixedBy      DependencyKind = "fixed_by"
)

type Dependency struct {
	SourceID string         `json:"source_id"`
	TargetID string         `json:"target_id"`
	Kind     DependencyKind `json:"kind"`
}
