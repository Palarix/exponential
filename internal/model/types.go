package model

import (
	"fmt"
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

	// Release events share issues.db with issue events. Their IDs carry the
	// "rel-" prefix, so ProjectIssues never mistakes them for issues.
	EventTypeReleaseCreate   EventType = "RELEASE_CREATE"
	EventTypeReleaseUpdate   EventType = "RELEASE_UPDATE"
	EventTypeReleaseComplete EventType = "RELEASE_COMPLETE"
	EventTypeReleaseDelete   EventType = "RELEASE_DELETE"
)

// ReleaseIDPrefix starts the ID of every release event.
const ReleaseIDPrefix = "rel-"

// IsReleaseEvent reports whether t is one of the release event types.
func IsReleaseEvent(t EventType) bool {
	switch t {
	case EventTypeReleaseCreate, EventTypeReleaseUpdate, EventTypeReleaseComplete, EventTypeReleaseDelete:
		return true
	}
	return false
}

type ReleaseState string

const (
	ReleaseOpen     ReleaseState = "OPEN"
	ReleaseReleased ReleaseState = "RELEASED"
)

// What complete and delete do with a release's issues.
const (
	ReleaseMoveToNext = "move_to_next"
	ReleaseRemove     = "remove"
)

type ReleaseCreatePayload struct {
	Number     int    `json:"number"`
	Label      string `json:"label,omitempty"`
	Goal       string `json:"goal,omitempty"`
	TargetDate string `json:"target_date,omitempty"`
	SortOrder  string `json:"sort_order,omitempty"`
}

type ReleaseUpdatePayload struct {
	Label      *string    `json:"label,omitempty"`
	Goal       *string    `json:"goal,omitempty"`
	TargetDate *string    `json:"target_date,omitempty"`
	SortOrder  *string    `json:"sort_order,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
}

// ReleaseStats is the final record of a completed release: counts by status,
// story points and the IDs of its leaf issues. It survives `xpo archive`.
// Releases completed before points were recorded have no Points.
type ReleaseStats struct {
	Counts  map[string]int `json:"counts"`
	Points  *ReleasePoints `json:"points,omitempty"`
	LeafIDs []string       `json:"leaf_ids"`
}

// ReleasePoints is the story points of a release's scope, all of it and the
// done part. Unestimated issues count as 0.
type ReleasePoints struct {
	Total int `json:"total"`
	Done  int `json:"done"`
}

// Moved counts the issues the operation moved (or removed, when MovedTo is
// ""), so its single feed entry can say so.
type ReleaseCompletePayload struct {
	OpenIssues string       `json:"open_issues"`
	Moved      int          `json:"moved,omitempty"`
	MovedTo    string       `json:"moved_to,omitempty"`
	Stats      ReleaseStats `json:"stats"`
}

type ReleaseDeletePayload struct {
	Issues  string `json:"issues"`
	Moved   int    `json:"moved,omitempty"`
	MovedTo string `json:"moved_to,omitempty"`
}

// ChildrenMove records, on an epic's update, how many open children an
// update with_children moved and to which release ("" = removed). It is not
// an issue field.
type ChildrenMove struct {
	Count     int    `json:"count"`
	ReleaseID string `json:"release_id"`
}

// Release is the projected state of one release. A placeholder is an unsaved
// Current or Next slot: it has a number but no ID. The Next placeholder is
// Locked while Current is a placeholder too: writes to it are refused.
type Release struct {
	ID          string
	Number      int
	Label       string
	Goal        string
	State       ReleaseState
	SortOrder   string
	TargetDate  string
	StartedAt   *time.Time
	ReleasedAt  *time.Time
	CreatedAt   time.Time
	CreatedBy   string
	Deleted     bool
	Placeholder bool
	Locked      bool
	Stats       *ReleaseStats
}

// Ref is "#<number>", the ref that always names this release.
func (r *Release) Ref() string {
	return fmt.Sprintf("#%d", r.Number)
}

// Display is the label, or the ref when there is none.
func (r *Release) Display() string {
	if r.Label != "" {
		return r.Label
	}
	return r.Ref()
}

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
	OnBehalfOf   string    `json:"on_behalf_of,omitempty"`
}

type Event struct {
	ID         string      `json:"id"`
	Type       EventType   `json:"type"`
	Payload    interface{} `json:"payload"`
	CreatedAt  time.Time   `json:"created_at"`
	CreatedBy  string      `json:"created_by"`            // Actor: "First Last <email>" or agent identity
	OnBehalfOf string      `json:"on_behalf_of,omitempty"` // Principal: user the actor is working for
	Source     string      `json:"source,omitempty"`       // Origin channel: "web", "mcp", "cli"
	// Cause is the release or epic issue whose bulk operation wrote this
	// event (a release complete/delete, an update with_children). Feeds
	// show the operation once and skip its caused events; the issue's own
	// history keeps them.
	Cause string `json:"cause,omitempty"`
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
	ReleaseID    string       `json:"release_id,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Labels       []string     `json:"labels,omitempty"`
}

// IssueDraft is an unfinished new issue saved from the web UI. It lives in
// .xpo/drafts/<id>.json, outside the event log, and has no issue ID.
type IssueDraft struct {
	ID        string            `json:"id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	CreatedBy string            `json:"created_by"`
	Payload   IssueDraftPayload `json:"payload"`
}

// IssueDraftPayload mirrors the fields of the new-issue modal.
type IssueDraftPayload struct {
	Title            string   `json:"title"`
	Description      string   `json:"description,omitempty"`
	Labels           []string `json:"labels,omitempty"`
	AdditionalLabels []string `json:"additional_labels,omitempty"`
	Status           string   `json:"status,omitempty"`
	Estimate         int      `json:"estimate,omitempty"`
	ParentID         string   `json:"parent_id,omitempty"`
	Assignee         string   `json:"assignee,omitempty"`
	// Release is the release ref picked in the form ("none" for no release).
	// Empty lets a sub-issue copy its parent's release.
	Release string `json:"release,omitempty"`
}

type CommentPayload struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Comment struct {
	ID         string    `json:"id"`
	Text       string    `json:"text"`
	CreatedBy  string    `json:"created_by"`
	OnBehalfOf string    `json:"on_behalf_of,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
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
	ReleaseID    *string      `json:"release_id,omitempty"`
	Children     *ChildrenMove `json:"children_moved,omitempty"`
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

// BlockerResolved reports whether a blocked_by blocker in status s no longer
// blocks. Only DONE resolves it: a CANCELED or DUPLICATE blocker means the
// link is stale and must be removed or pointed at another issue (xpo-35943d).
func BlockerResolved(s IssueStatus) bool {
	return s == StatusDone
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
	Assignee     string // principal: the person accountable, never an agent
	AssigneeVia  string // agent identity doing the work for Assignee, or ""
	CycleID          string
	EffectiveCycleID string
	ReleaseID        string
	ReleaseName      string // display name of ReleaseID, filled in by projection
	InferredStatus   bool
	BranchStats      *BranchStats
	Deleted          bool

	// Dependencies holds the links stored on this issue (the event-sourced
	// truth). Links is the combined view: Dependencies plus the inverse of
	// every link other live issues store against this one, marked Derived.
	Dependencies []Dependency
	Links        []Dependency
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
	// Derived marks a link stored on the other issue and shown here as its
	// inverse. It is never persisted.
	Derived bool `json:"derived,omitempty"`
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
	Release    string      `json:"release,omitempty"`    // display name of the release the event sets or is about
	MovedTo    string      `json:"moved_to,omitempty"`   // release events: display name of the release the issues moved to
	Cause      string      `json:"cause,omitempty"`      // see Event.Cause
	CauseNote  string      `json:"cause_note,omitempty"` // e.g. "when 1.5 was released"
	SHA        string      `json:"sha,omitempty"`
	Message    string      `json:"message,omitempty"`
	Author     string      `json:"author,omitempty"`
	Branch     string      `json:"branch,omitempty"`
}
