package jsonio

import (
	"time"

	"github.com/palarix/exponential/internal/model"
)

type ErrorOutput struct {
	Error string `json:"error"`
}

type IssueSummary struct {
	ID               string             `json:"id"`
	Title            string             `json:"title"`
	Status           string             `json:"status"`
	IsInferred       bool               `json:"is_inferred,omitempty"`
	Labels           []string           `json:"labels,omitempty"`
	ParentID         string             `json:"parent_id,omitempty"`
	StoryPoints      int                `json:"story_points,omitempty"`
	Assignee         string             `json:"assignee,omitempty"`
	CycleID          string             `json:"cycle_id,omitempty"`
	EffectiveCycleID string             `json:"effective_cycle_id,omitempty"`
	BranchStats      *model.BranchStats `json:"branch_stats,omitempty"`
	UpdatedAt        string             `json:"updated_at"`
}

type CommentSummary struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type EventSummary struct {
	Type      string `json:"type"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type ListOutput struct {
	Issues []IssueSummary `json:"issues"`
}

type ShowOutput struct {
	ID               string             `json:"id"`
	Title            string             `json:"title"`
	Status           string             `json:"status"`
	IsInferred       bool               `json:"is_inferred,omitempty"`
	Description      string             `json:"description,omitempty"`
	Labels           []string           `json:"labels,omitempty"`
	ParentID         string             `json:"parent_id,omitempty"`
	StoryPoints      int                `json:"story_points,omitempty"`
	Assignee         string             `json:"assignee,omitempty"`
	CycleID          string             `json:"cycle_id,omitempty"`
	EffectiveCycleID string             `json:"effective_cycle_id,omitempty"`
	BranchStats      *model.BranchStats `json:"branch_stats,omitempty"`
	CreatedBy        string             `json:"created_by"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
	Dependencies     []model.Dependency `json:"dependencies,omitempty"`
	Artifacts        []ArtifactEntry    `json:"artifacts,omitempty"`
	Comments         []CommentSummary   `json:"comments,omitempty"`
	Events           []EventSummary     `json:"events,omitempty"`
	Children         []IssueSummary     `json:"children,omitempty"`
	Archived         bool               `json:"archived,omitempty"`
}

type CommentsOutput struct {
	Comments []CommentSummary `json:"comments"`
}

type HistoryOutput struct {
	Timeline []TimelineEntry `json:"timeline"`
}

type TimelineEntry struct {
	Kind       string      `json:"kind"`
	Timestamp  string      `json:"timestamp"`
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

type AddOutput struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type UpdateOutput struct {
	ID       string   `json:"id"`
	Messages []string `json:"messages"`
}

type CommentOutput struct {
	ID string `json:"id"`
}

type StartOutput struct {
	ID           string   `json:"id"`
	Branch       string   `json:"branch,omitempty"`
	WorktreePath string   `json:"worktree_path,omitempty"`
	Messages     []string `json:"messages"`
}

type MergeOutput struct {
	ID       string   `json:"id"`
	MergeSHA string   `json:"merge_sha"`
	Messages []string `json:"messages"`
}

type LinkOutput struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type SpecOutput struct {
	OK      bool   `json:"ok"`
	IssueID string `json:"issue_id"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

type WalkthroughOutput struct {
	OK      bool   `json:"ok"`
	IssueID string `json:"issue_id"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ArtifactEntry struct {
	ArtifactType string `json:"type"`
	Filename     string `json:"filename"`
	UpdatedAt    string `json:"updated_at"`
	UpdatedBy    string `json:"updated_by"`
}

type ArtifactOutput struct {
	OK        bool            `json:"ok"`
	IssueID   string          `json:"issue_id"`
	Path      string          `json:"path,omitempty"`
	Content   string          `json:"content,omitempty"`
	Artifacts []ArtifactEntry `json:"artifacts,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type RationaleOutput struct {
	Results      []RationaleHit `json:"results"`
	Query        string         `json:"query"`
	TotalMatches int            `json:"total_matches"`
}

type RationaleHit struct {
	IssueID   string   `json:"issue_id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Labels    []string `json:"labels,omitempty"`
	Document  string   `json:"document"`
	Fragment  string   `json:"fragment"`
	Score     float64  `json:"score"`
	UpdatedAt string   `json:"updated_at"`
}

type InboxOutput struct {
	Items []InboxEntry `json:"items"`
}

type AttentionKind string

const (
	AttentionBlocker      AttentionKind = "blocker"
	AttentionStaleWIP     AttentionKind = "stale_wip"
	AttentionHighPriority AttentionKind = "high_priority"
)

type AttentionItem struct {
	IssueID  string        `json:"issue_id"`
	Title    string        `json:"title"`
	Kind     AttentionKind `json:"kind"`
	AgeDays  int           `json:"age_days,omitempty"`
	Priority int           `json:"priority,omitempty"`
}

type WorkloadEntry struct {
	Assignee      string `json:"assignee"`
	InProgress    int    `json:"in_progress"`
	OpenPoints    int    `json:"open_points"`
	Blocked       int    `json:"blocked"`
	LastCompleted string `json:"last_completed,omitempty"`
}

type WeeklyTrend struct {
	WeekStart string `json:"week_start"`
	Created   int    `json:"created"`
	Completed int    `json:"completed"`
}

type AgeBuckets struct {
	Under1d  int `json:"under_1d"`
	Under3d  int `json:"under_3d"`
	Under7d  int `json:"under_7d"`
	Under14d int `json:"under_14d"`
	Under30d int `json:"under_30d"`
	Over30d  int `json:"over_30d"`
}

type BugAgeBuckets struct {
	Under24h int `json:"under_24h"`
	Under48h int `json:"under_48h"`
	Under5d  int `json:"under_5d"`
	Under14d int `json:"under_14d"`
	Under1mo int `json:"under_1mo"`
	Over1mo  int `json:"over_1mo"`
}

type TrendsBlock struct {
	Weekly           []WeeklyTrend `json:"weekly"`
	MedianTriageMins int           `json:"median_triage_mins"`
	TriagedCount     int           `json:"triaged_count"`
	BugAge           BugAgeBuckets `json:"bug_age"`
}

type EpicProgress struct {
	IssueID         string `json:"issue_id"`
	Title           string `json:"title"`
	ChildrenDone    int    `json:"children_done"`
	ChildrenTotal   int    `json:"children_total"`
	PointsDone      int    `json:"points_done"`
	PointsTotal     int    `json:"points_total"`
	PointsRemaining int    `json:"points_remaining"`
	Stale           bool   `json:"stale"`
}

type VelocityBucket struct {
	WeekStart string `json:"week_start"`
	Points    int    `json:"points"`
}

type DailyBucket struct {
	Date   string `json:"date"`
	Points int    `json:"points"`
}

type PulseOutput struct {
	Velocity struct {
		CurrentWeekPoints int              `json:"current_week_points"`
		Last7dPoints      int              `json:"last_7d_points"`
		Prior7dPoints     int              `json:"prior_7d_points"`
		Delta             int              `json:"delta"`
		WeeklyBuckets     []VelocityBucket `json:"weekly_buckets"`
		DailyBuckets      []DailyBucket    `json:"daily_buckets"`
	} `json:"velocity"`
	Throughput struct {
		Last7d  int `json:"last_7d"`
		Prior7d int `json:"prior_7d"`
		Delta   int `json:"delta"`
	} `json:"throughput"`
	WIP struct {
		Total              int `json:"total"`
		Stale              int `json:"stale"`
		StaleThresholdDays int `json:"stale_threshold_days"`
	} `json:"wip"`
	Blockers struct {
		Total      int `json:"total"`
		OldestDays int `json:"oldest_days"`
	} `json:"blockers"`
	Flow struct {
		CycleTimeHrs    float64    `json:"cycle_time_hrs"`
		CycleTimeP75Hrs float64    `json:"cycle_time_p75_hrs"`
		CycleTimeP90Hrs float64    `json:"cycle_time_p90_hrs"`
		CycleTimeMinHrs float64    `json:"cycle_time_min_hrs"`
		CycleTimeMaxHrs float64    `json:"cycle_time_max_hrs"`
		CycleCount      int        `json:"cycle_count"`
		LeadTimeHrs     float64    `json:"lead_time_hrs"`
		LeadTimeP75Hrs  float64    `json:"lead_time_p75_hrs"`
		LeadTimeP90Hrs  float64    `json:"lead_time_p90_hrs"`
		LeadTimeMinHrs  float64    `json:"lead_time_min_hrs"`
		LeadTimeMaxHrs  float64    `json:"lead_time_max_hrs"`
		LeadCount       int        `json:"lead_count"`
		Staleness       AgeBuckets `json:"staleness"`
		StalenessTotal  int        `json:"staleness_total"`
	} `json:"flow"`
	Attention []AttentionItem `json:"attention"`
	Workload  []WorkloadEntry `json:"workload"`
	Epics     []EpicProgress  `json:"epics"`
	Trends    TrendsBlock     `json:"trends"`
}

type InboxEntry struct {
	IssueID    string      `json:"issue_id"`
	IssueTitle string      `json:"issue_title"`
	Type       string      `json:"type"`
	Payload    interface{} `json:"payload,omitempty"`
	CreatedAt  string      `json:"created_at"`
	CreatedBy  string      `json:"created_by"`
	OnBehalfOf string      `json:"on_behalf_of,omitempty"`
}

func ToIssueSummary(i *model.Issue) IssueSummary {
	return IssueSummary{
		ID:               i.ID,
		Title:            i.Title,
		Status:           string(i.Status),
		IsInferred:       i.InferredStatus,
		Labels:           i.Labels,
		ParentID:         i.ParentID,
		StoryPoints:      i.Estimate,
		Assignee:         i.Assignee,
		CycleID:          i.CycleID,
		EffectiveCycleID: i.EffectiveCycleID,
		BranchStats:      i.BranchStats,
		UpdatedAt:        i.UpdatedAt.Format(time.RFC3339),
	}
}

func ToArtifactEntries(as []model.ArtifactSummary) []ArtifactEntry {
	if len(as) == 0 {
		return nil
	}
	out := make([]ArtifactEntry, len(as))
	for i, a := range as {
		out[i] = ArtifactEntry{
			ArtifactType: a.ArtifactType,
			Filename:     a.Filename,
			UpdatedAt:    a.UpdatedAt.Format(time.RFC3339),
			UpdatedBy:    a.UpdatedBy,
		}
	}
	return out
}

func ToCommentSummaries(cs []model.Comment) []CommentSummary {
	if len(cs) == 0 {
		return nil
	}
	out := make([]CommentSummary, len(cs))
	for i, c := range cs {
		out[i] = CommentSummary{
			ID:        c.ID,
			Text:      c.Text,
			CreatedBy: c.CreatedBy,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}

func ToEventSummaries(es []model.Event) []EventSummary {
	if len(es) == 0 {
		return nil
	}
	out := make([]EventSummary, len(es))
	for i, e := range es {
		out[i] = EventSummary{
			Type:      string(e.Type),
			CreatedBy: e.CreatedBy,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}

func ToTimelineEntries(entries []model.TimelineEntry) []TimelineEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]TimelineEntry, len(entries))
	for i, e := range entries {
		out[i] = TimelineEntry{
			Kind:       e.Kind,
			Timestamp:  e.Timestamp.Format(time.RFC3339),
			IssueID:    e.IssueID,
			IssueTitle: e.IssueTitle,
			EventType:  e.EventType,
			Payload:    e.Payload,
			CreatedBy:  e.CreatedBy,
			OnBehalfOf: e.OnBehalfOf,
			Source:     e.Source,
			SHA:        e.SHA,
			Message:    e.Message,
			Author:     e.Author,
			Branch:     e.Branch,
		}
	}
	return out
}

func EventsToTimelineEntries(es []model.Event) []TimelineEntry {
	if len(es) == 0 {
		return nil
	}
	out := make([]TimelineEntry, len(es))
	for i, e := range es {
		out[i] = TimelineEntry{
			Kind:       "issue_event",
			Timestamp:  e.CreatedAt.Format(time.RFC3339),
			IssueID:    e.ID,
			EventType:  string(e.Type),
			Payload:    e.Payload,
			CreatedBy:  e.CreatedBy,
			OnBehalfOf: e.OnBehalfOf,
			Source:     e.Source,
		}
	}
	return out
}
