package model

import (
	"time"
)

type EventType string

const (
	EventTypeCreate   EventType = "CREATE"
	EventTypeUpdate   EventType = "UPDATE"
	EventTypeDelete   EventType = "DELETE"
	EventTypeComment  EventType = "COMMENT"
	EventTypeMerge    EventType = "MERGE"
	EventTypeArtifact EventType = "ARTIFACT"
)

type DeletePayload struct {
	Reason  string `json:"reason,omitempty"`
	Cascade bool   `json:"cascade,omitempty"`
}

type MergePayload struct {
	Branch   string `json:"branch"`
	BaseSHA  string `json:"base_sha"`
	MergeSHA string `json:"merge_sha"`
	Strategy string `json:"strategy"`
}

type ArtifactPayload struct {
	ArtifactType string `json:"artifact_type"`
	Filename     string `json:"filename"`
	Action       string `json:"action"`
}

type ArtifactSummary struct {
	ArtifactType string    `json:"artifact_type"`
	Filename     string    `json:"filename"`
	UpdatedAt    time.Time `json:"updated_at"`
	UpdatedBy    string    `json:"updated_by"`
}

type Event struct {
	ID         string      `json:"id"`
	Type       EventType   `json:"type"`
	Payload    interface{} `json:"payload"`
	CreatedAt  time.Time   `json:"created_at"`
	CreatedBy  string      `json:"created_by"`            // Actor: "First Last <email>" or agent identity
	OnBehalfOf string      `json:"on_behalf_of,omitempty"` // Principal: user the actor is working for
	Source     string      `json:"source,omitempty"`       // Origin channel: "web", "mcp", "cli"
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
	Dependencies []Dependency `json:"dependencies"`
	Labels       []string     `json:"labels"`
}

type IssueStatus string

const (
	StatusBacklog   IssueStatus = "BACKLOG"
	StatusPlanned   IssueStatus = "PLANNED"
	StatusDoing     IssueStatus = "DOING"
	StatusBlocked   IssueStatus = "BLOCKED"
	StatusDone      IssueStatus = "DONE"
	StatusCanceled  IssueStatus = "CANCELED"
	StatusDuplicate IssueStatus = "DUPLICATE"
)

func IsTerminal(s IssueStatus) bool {
	return s == StatusDone || s == StatusCanceled || s == StatusDuplicate
}

func IsCompleted(s IssueStatus) bool {
	return s == StatusDone
}

type BranchStats struct {
	Branch         string `json:"branch"`
	HeadSHA        string `json:"head_sha"`
	Commits        int    `json:"commits"`
	FilesChanged   int    `json:"files_changed"`
	Insertions     int    `json:"insertions"`
	Deletions      int    `json:"deletions"`
	HasUncommitted bool   `json:"has_uncommitted"`
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
	Artifacts []ArtifactSummary
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

type TimelineEntry struct {
	Kind       string      `json:"kind"`
	Timestamp  time.Time   `json:"timestamp"`
	IssueID    string      `json:"issue_id,omitempty"`
	IssueTitle string      `json:"issue_title,omitempty"`
	EventType  string      `json:"event_type,omitempty"`
	Payload    interface{} `json:"payload,omitempty"`
	CreatedBy  string      `json:"created_by,omitempty"`
	OnBehalfOf string      `json:"on_behalf_of,omitempty"`
	Source     string      `json:"source,omitempty"`
	SHA        string      `json:"sha,omitempty"`
	Message    string      `json:"message,omitempty"`
	Author     string      `json:"author,omitempty"`
	Branch     string      `json:"branch,omitempty"`
}
