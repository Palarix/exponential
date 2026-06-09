package model

import (
	"time"
)

type EventType string

const (
	EventTypeCreate  EventType = "CREATE"
	EventTypeUpdate  EventType = "UPDATE"
	EventTypeDelete  EventType = "DELETE"
	EventTypeComment EventType = "COMMENT"
)

type DeletePayload struct {
	Reason  string `json:"reason,omitempty"`
	Cascade bool   `json:"cascade,omitempty"`
}

type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
	CreatedBy string      `json:"created_by"` // Format: "First Last <email>"
}

type CreatePayload struct {
	Title        string       `json:"title"`
	Description  string       `json:"description,omitempty"`
	Status       string       `json:"status,omitempty"`
	ParentID     string       `json:"parent_id,omitempty"`
	Estimate     int          `json:"estimate,omitempty"`
	Priority     int          `json:"priority,omitempty"`
	SortOrder    string       `json:"sort_order,omitempty"`
	Assignee     string       `json:"assignee,omitempty"`
	CycleID      string       `json:"cycle_id,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Labels       []string     `json:"labels,omitempty"`
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
	Title        *string      `json:"title,omitempty"`
	Description  *string      `json:"description,omitempty"`
	Status       *string      `json:"status,omitempty"`
	ParentID     *string      `json:"parent_id,omitempty"`
	Estimate     *int         `json:"estimate,omitempty"`
	Priority     *int         `json:"priority,omitempty"`
	SortOrder    *string      `json:"sort_order,omitempty"`
	Assignee     *string      `json:"assignee,omitempty"`
	CycleID      *string      `json:"cycle_id,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Labels       []string     `json:"labels,omitempty"`
}

type IssueStatus string

const (
	StatusBacklog IssueStatus = "BACKLOG"
	StatusPlanned IssueStatus = "PLANNED"
	StatusDoing   IssueStatus = "DOING"
	StatusBlocked IssueStatus = "BLOCKED"
	StatusDone    IssueStatus = "DONE"
)

type BranchStats struct {
	Branch       string `json:"branch"`
	Commits      int    `json:"commits"`
	FilesChanged int    `json:"files_changed"`
	Insertions   int    `json:"insertions"`
	Deletions    int    `json:"deletions"`
}

type Issue struct {
	ID           string
	Title        string
	Description  string
	Status       IssueStatus
	ParentID     string
	Estimate     int
	Priority     int
	SortOrder    string
	Assignee     string
	CycleID          string
	EffectiveCycleID string
	InferredStatus   bool
	BranchStats      *BranchStats
	Deleted          bool

	Dependencies []Dependency
	Labels       []string

	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	Events    []Event
	Comments  []Comment
}

type DependencyKind string

const (
	DependencyDependsOn    DependencyKind = "depends_on"
	DependencyDependencyOf DependencyKind = "dependency_of"
	DependencyBlockedBy    DependencyKind = "blocked_by"
	DependencyBlocks       DependencyKind = "blocks"
	DependencyDuplicatedBy DependencyKind = "duplicated_by"
	DependencyDuplicates   DependencyKind = "duplicates"
)

// InverseKind returns the inverse dependency kind.
func InverseKind(k DependencyKind) DependencyKind {
	switch k {
	case DependencyDependsOn:
		return DependencyDependencyOf
	case DependencyDependencyOf:
		return DependencyDependsOn
	case DependencyBlockedBy:
		return DependencyBlocks
	case DependencyBlocks:
		return DependencyBlockedBy
	case DependencyDuplicatedBy:
		return DependencyDuplicates
	case DependencyDuplicates:
		return DependencyDuplicatedBy
	default:
		return k
	}
}

type Dependency struct {
	SourceID string         `json:"source_id"`
	TargetID string         `json:"target_id"`
	Kind     DependencyKind `json:"kind"`
}

// NormalizeDependencyKind maps loose user input ("blocks", "blocked_by",
// "BlockedBy", "relates_to", etc.) onto canonical kind strings. Returns an
// empty string for unrecognized inputs.
func NormalizeDependencyKind(kind string) string {
	lower := ""
	for _, r := range kind {
		if r == '_' || r == '-' || r == ' ' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 32
		}
		lower += string(r)
	}
	switch lower {
	case "blocks":
		return string(DependencyBlocks)
	case "blockedby":
		return string(DependencyBlockedBy)
	case "dependson":
		return string(DependencyDependsOn)
	case "dependencyof":
		return string(DependencyDependencyOf)
	case "duplicates":
		return string(DependencyDuplicates)
	case "duplicatedby":
		return string(DependencyDuplicatedBy)
	case "relatesto":
		return "relates_to"
	}
	return ""
}
