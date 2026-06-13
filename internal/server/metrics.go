package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/palarix/beats/internal/model"
)

// attentionKind classifies why an issue is surfaced in the Attention card.
type attentionKind string

const (
	attentionBlocker      attentionKind = "blocker"
	attentionStaleWIP     attentionKind = "stale_wip"
	attentionHighPriority attentionKind = "high_priority"
)

type attentionItem struct {
	IssueID  string        `json:"issue_id"`
	Title    string        `json:"title"`
	Kind     attentionKind `json:"kind"`
	AgeDays  int           `json:"age_days,omitempty"`
	Priority int           `json:"priority,omitempty"`
}

type workloadEntry struct {
	Assignee      string `json:"assignee"`
	InProgress    int    `json:"in_progress"`
	OpenPoints    int    `json:"open_points"`
	Blocked       int    `json:"blocked"`
	LastCompleted string `json:"last_completed,omitempty"` // YYYY-MM-DD
}

type weeklyTrend struct {
	WeekStart string `json:"week_start"`
	Created   int    `json:"created"`
	Completed int    `json:"completed"`
}

type ageBuckets struct {
	Under1d  int `json:"under_1d"`
	Under3d  int `json:"under_3d"`
	Under7d  int `json:"under_7d"`
	Under14d int `json:"under_14d"`
	Under30d int `json:"under_30d"`
	Over30d  int `json:"over_30d"`
}

type bugAgeBuckets struct {
	Under24h int `json:"under_24h"`
	Under48h int `json:"under_48h"`
	Under5d  int `json:"under_5d"`
	Under14d int `json:"under_14d"`
	Under1mo int `json:"under_1mo"`
	Over1mo  int `json:"over_1mo"`
}

type trendsBlock struct {
	Weekly             []weeklyTrend `json:"weekly"`
	MedianTriageMins   int           `json:"median_triage_mins"`
	TriagedCount       int           `json:"triaged_count"`
	BugAge             bugAgeBuckets `json:"bug_age"`
}

type epicProgress struct {
	IssueID         string `json:"issue_id"`
	Title           string `json:"title"`
	ChildrenDone    int    `json:"children_done"`
	ChildrenTotal   int    `json:"children_total"`
	PointsDone      int    `json:"points_done"`
	PointsTotal     int    `json:"points_total"`
	PointsRemaining int    `json:"points_remaining"`
	Stale           bool   `json:"stale"`
}

// staleWipDays is the threshold beyond which an issue in DOING is considered stale.
const staleWipDays = 5

type velocityBucket struct {
	WeekStart string `json:"week_start"` // YYYY-MM-DD (Monday)
	Points    int    `json:"points"`
}

type dailyBucket struct {
	Date   string `json:"date"` // YYYY-MM-DD
	Points int    `json:"points"`
}

type pulseMetrics struct {
	Velocity struct {
		Last7dPoints  int              `json:"last_7d_points"`
		Prior7dPoints int              `json:"prior_7d_points"`
		Delta         int              `json:"delta"`
		WeeklyBuckets []velocityBucket `json:"weekly_buckets"`
		DailyBuckets  []dailyBucket    `json:"daily_buckets"`
	} `json:"velocity"`
	Throughput struct {
		Last7d  int `json:"last_7d"`
		Prior7d int `json:"prior_7d"`
		Delta   int `json:"delta"` // signed absolute count delta
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
		Staleness        ageBuckets `json:"staleness"`
		StalenessTotal   int        `json:"staleness_total"`
	} `json:"flow"`
	Attention []attentionItem `json:"attention"`
	Workload  []workloadEntry `json:"workload"`
	Epics     []epicProgress  `json:"epics"`
	Trends    trendsBlock     `json:"trends"`
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load issues: %v", err))
		return
	}
	respondJSON(w, http.StatusOK, computePulseMetrics(issues, time.Now()))
}

func computePulseMetrics(issues map[string]*model.Issue, now time.Time) pulseMetrics {
	var m pulseMetrics
	m.WIP.StaleThresholdDays = staleWipDays

	last7dStart := now.AddDate(0, 0, -7)
	prior7dStart := now.AddDate(0, 0, -14)

	// Pre-build 8 weekly buckets (oldest → newest, ending on the current week).
	bucketsByKey := make(map[string]int)
	bucketOrder := make([]string, 0, 8)
	for w := 7; w >= 0; w-- {
		key := startOfWeek(now.AddDate(0, 0, -7*w)).Format("2006-01-02")
		if _, exists := bucketsByKey[key]; !exists {
			bucketsByKey[key] = 0
			bucketOrder = append(bucketOrder, key)
		}
	}

	// Pre-build 14 daily buckets (oldest → newest).
	dailyByKey := make(map[string]int)
	dailyOrder := make([]string, 0, 14)
	for d := 13; d >= 0; d-- {
		key := now.AddDate(0, 0, -d).Format("2006-01-02")
		dailyByKey[key] = 0
		dailyOrder = append(dailyOrder, key)
	}

	// Trend buckets (created/completed issue counts per week, same 8-week window).
	trendByKey := make(map[string]*weeklyTrend, len(bucketOrder))
	for _, k := range bucketOrder {
		trendByKey[k] = &weeklyTrend{WeekStart: k}
	}

	var triageDurations []time.Duration
	var cycleTimes []time.Duration
	var leadTimes []time.Duration
	var staleness ageBuckets
	var stalenessTotal int
	var bugAge bugAgeBuckets

	var blockerCandidates, staleCandidates, highPriorityCandidates []attentionItem
	type workAccum struct {
		entry            workloadEntry
		lastCompletedAt  *time.Time
	}
	workload := make(map[string]*workAccum)

	for _, issue := range issues {
		if issue.Deleted {
			continue
		}

		doneAt, doingAt, blockedAt := latestStatusTimestamps(issue)

		// Effective points: no-estimate counts as 1.
		points := issue.Estimate
		if points == 0 {
			points = 1
		}

		if doneAt != nil {
			if doneAt.After(last7dStart) {
				m.Throughput.Last7d++
				m.Velocity.Last7dPoints += points
			} else if doneAt.After(prior7dStart) {
				m.Throughput.Prior7d++
				m.Velocity.Prior7dPoints += points
			}
			weekKey := startOfWeek(*doneAt).Format("2006-01-02")
			if _, ok := bucketsByKey[weekKey]; ok {
				bucketsByKey[weekKey] += points
			}
			dayKey := doneAt.Format("2006-01-02")
			if _, ok := dailyByKey[dayKey]; ok {
				dailyByKey[dayKey] += points
			}
			if t, ok := trendByKey[weekKey]; ok {
				t.Completed++
			}
		}

		// Trend "created" count: bucket by CreatedAt week.
		createdWeekKey := startOfWeek(issue.CreatedAt).Format("2006-01-02")
		if t, ok := trendByKey[createdWeekKey]; ok {
			t.Created++
		}

		// Flow metrics
		if doneAt != nil {
			lt := doneAt.Sub(issue.CreatedAt)
			if lt > 0 {
				leadTimes = append(leadTimes, lt)
			}
			if doingAt != nil && doneAt.After(*doingAt) {
				cycleTimes = append(cycleTimes, doneAt.Sub(*doingAt))
			}
		}
		if issue.Status != model.StatusDone && !issue.Deleted {
			age := now.Sub(issue.CreatedAt)
			stalenessTotal++
			days := age.Hours() / 24
			switch {
			case days < 1:
				staleness.Under1d++
			case days < 3:
				staleness.Under3d++
			case days < 7:
				staleness.Under7d++
			case days < 14:
				staleness.Under14d++
			case days < 30:
				staleness.Under30d++
			default:
				staleness.Over30d++
			}
		}

		// Triage time: from CreatedAt to the first UPDATE event that sets status away from BACKLOG.
		for _, evt := range issue.Events {
			if evt.Type != model.EventTypeUpdate {
				continue
			}
			payloadBytes, _ := json.Marshal(evt.Payload)
			var p model.UpdatePayload
			if err := json.Unmarshal(payloadBytes, &p); err != nil || p.Status == nil {
				continue
			}
			if *p.Status == string(model.StatusBacklog) {
				continue
			}
			triageDurations = append(triageDurations, evt.CreatedAt.Sub(issue.CreatedAt))
			break
		}

		// Bug age: open issues (non-DONE) labeled "bug", bucketed by age from CreatedAt.
		if issue.Status != model.StatusDone && hasBugLabel(issue.Labels) {
			age := now.Sub(issue.CreatedAt)
			switch {
			case age < 24*time.Hour:
				bugAge.Under24h++
			case age < 48*time.Hour:
				bugAge.Under48h++
			case age < 5*24*time.Hour:
				bugAge.Under5d++
			case age < 14*24*time.Hour:
				bugAge.Under14d++
			case age < 30*24*time.Hour:
				bugAge.Under1mo++
			default:
				bugAge.Over1mo++
			}
		}

		if issue.Status == model.StatusDoing {
			m.WIP.Total++
			if doingAt != nil && int(now.Sub(*doingAt).Hours()/24) > staleWipDays {
				age := int(now.Sub(*doingAt).Hours() / 24)
				m.WIP.Stale++
				staleCandidates = append(staleCandidates, attentionItem{
					IssueID: issue.ID,
					Title:   issue.Title,
					Kind:    attentionStaleWIP,
					AgeDays: age,
				})
			}
		}

		if issue.Status == model.StatusBlocked {
			m.Blockers.Total++
			age := 0
			if blockedAt != nil {
				age = int(now.Sub(*blockedAt).Hours() / 24)
				if age > m.Blockers.OldestDays {
					m.Blockers.OldestDays = age
				}
			}
			blockerCandidates = append(blockerCandidates, attentionItem{
				IssueID: issue.ID,
				Title:   issue.Title,
				Kind:    attentionBlocker,
				AgeDays: age,
			})
		}

		// High priority not started: priority Urgent (1) or High (2), status BACKLOG or PLANNED.
		if issue.Priority > 0 && issue.Priority <= 2 &&
			(issue.Status == model.StatusBacklog || issue.Status == model.StatusPlanned) {
			highPriorityCandidates = append(highPriorityCandidates, attentionItem{
				IssueID:  issue.ID,
				Title:    issue.Title,
				Kind:     attentionHighPriority,
				Priority: issue.Priority,
			})
		}

		// Workload — only count issues with an assignee.
		if issue.Assignee != "" {
			wa := workload[issue.Assignee]
			if wa == nil {
				wa = &workAccum{entry: workloadEntry{Assignee: issue.Assignee}}
				workload[issue.Assignee] = wa
			}
			if issue.Status == model.StatusDone {
				if doneAt != nil && (wa.lastCompletedAt == nil || doneAt.After(*wa.lastCompletedAt)) {
					t := *doneAt
					wa.lastCompletedAt = &t
				}
			} else {
				wa.entry.OpenPoints += points
				if issue.Status == model.StatusDoing || issue.Status == model.StatusBlocked {
					wa.entry.InProgress++
				}
				if issue.Status == model.StatusBlocked {
					wa.entry.Blocked++
				}
			}
		}
	}

	m.Throughput.Delta = m.Throughput.Last7d - m.Throughput.Prior7d
	m.Velocity.Delta = m.Velocity.Last7dPoints - m.Velocity.Prior7dPoints

	for _, key := range bucketOrder {
		m.Velocity.WeeklyBuckets = append(m.Velocity.WeeklyBuckets, velocityBucket{
			WeekStart: key,
			Points:    bucketsByKey[key],
		})
	}

	m.Velocity.DailyBuckets = make([]dailyBucket, 0, len(dailyOrder))
	for _, key := range dailyOrder {
		m.Velocity.DailyBuckets = append(m.Velocity.DailyBuckets, dailyBucket{
			Date:   key,
			Points: dailyByKey[key],
		})
	}

	// Attention: blockers (by age desc) → stale WIP (by age desc) → high priority (by priority asc). Cap 5.
	sort.Slice(blockerCandidates, func(i, j int) bool {
		return blockerCandidates[i].AgeDays > blockerCandidates[j].AgeDays
	})
	sort.Slice(staleCandidates, func(i, j int) bool {
		return staleCandidates[i].AgeDays > staleCandidates[j].AgeDays
	})
	sort.Slice(highPriorityCandidates, func(i, j int) bool {
		return highPriorityCandidates[i].Priority < highPriorityCandidates[j].Priority
	})
	m.Attention = make([]attentionItem, 0, 5)
	for _, group := range [][]attentionItem{blockerCandidates, staleCandidates, highPriorityCandidates} {
		for _, item := range group {
			if len(m.Attention) == 5 {
				break
			}
			m.Attention = append(m.Attention, item)
		}
		if len(m.Attention) == 5 {
			break
		}
	}

	// Workload: stable sort by in-progress desc, then open points desc.
	m.Workload = make([]workloadEntry, 0, len(workload))
	for _, wa := range workload {
		if wa.lastCompletedAt != nil {
			wa.entry.LastCompleted = wa.lastCompletedAt.Format("2006-01-02")
		}
		m.Workload = append(m.Workload, wa.entry)
	}
	sort.SliceStable(m.Workload, func(i, j int) bool {
		if m.Workload[i].InProgress != m.Workload[j].InProgress {
			return m.Workload[i].InProgress > m.Workload[j].InProgress
		}
		return m.Workload[i].OpenPoints > m.Workload[j].OpenPoints
	})

	m.Epics = computeEpicProgress(issues, now)

	// Trends block.
	m.Trends.Weekly = make([]weeklyTrend, 0, len(bucketOrder))
	for _, k := range bucketOrder {
		m.Trends.Weekly = append(m.Trends.Weekly, *trendByKey[k])
	}
	m.Trends.TriagedCount = len(triageDurations)
	if len(triageDurations) > 0 {
		sort.Slice(triageDurations, func(i, j int) bool { return triageDurations[i] < triageDurations[j] })
		median := triageDurations[len(triageDurations)/2]
		m.Trends.MedianTriageMins = int(median.Minutes())
	}
	m.Trends.BugAge = bugAge

	// Flow metrics
	hoursAtPercentile := func(sorted []time.Duration, p float64) float64 {
		idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
		if idx < 0 { idx = 0 }
		if idx >= len(sorted) { idx = len(sorted) - 1 }
		return math.Round(sorted[idx].Hours()*10) / 10
	}
	m.Flow.CycleCount = len(cycleTimes)
	if len(cycleTimes) > 0 {
		sort.Slice(cycleTimes, func(i, j int) bool { return cycleTimes[i] < cycleTimes[j] })
		m.Flow.CycleTimeHrs = hoursAtPercentile(cycleTimes, 50)
		m.Flow.CycleTimeP75Hrs = hoursAtPercentile(cycleTimes, 75)
		m.Flow.CycleTimeP90Hrs = hoursAtPercentile(cycleTimes, 90)
		m.Flow.CycleTimeMinHrs = hoursAtPercentile(cycleTimes, 0)
		m.Flow.CycleTimeMaxHrs = hoursAtPercentile(cycleTimes, 100)
	}
	m.Flow.LeadCount = len(leadTimes)
	if len(leadTimes) > 0 {
		sort.Slice(leadTimes, func(i, j int) bool { return leadTimes[i] < leadTimes[j] })
		m.Flow.LeadTimeHrs = hoursAtPercentile(leadTimes, 50)
		m.Flow.LeadTimeP75Hrs = hoursAtPercentile(leadTimes, 75)
		m.Flow.LeadTimeP90Hrs = hoursAtPercentile(leadTimes, 90)
		m.Flow.LeadTimeMinHrs = hoursAtPercentile(leadTimes, 0)
		m.Flow.LeadTimeMaxHrs = hoursAtPercentile(leadTimes, 100)
	}
	m.Flow.Staleness = staleness
	m.Flow.StalenessTotal = stalenessTotal

	return m
}

func hasBugLabel(labels []string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, "bug") {
			return true
		}
	}
	return false
}

// computeEpicProgress returns one row per active epic. An "epic" is an issue
// labeled "epic" (case-insensitive), with at least one child, in an active
// status (PLANNED / DOING / BLOCKED). BACKLOG epics are queued, not active,
// and DONE epics are completed — both excluded.
// Sorted by remaining points desc — biggest active initiative first.
func computeEpicProgress(issues map[string]*model.Issue, now time.Time) []epicProgress {
	// Index immediate children by parent id.
	childrenByParent := make(map[string][]*model.Issue)
	for _, issue := range issues {
		if issue.Deleted || issue.ParentID == "" {
			continue
		}
		childrenByParent[issue.ParentID] = append(childrenByParent[issue.ParentID], issue)
	}

	staleThreshold := now.Add(-staleWipDays * 24 * time.Hour)
	out := make([]epicProgress, 0)
	for _, issue := range issues {
		if issue.Deleted || !hasEpicLabel(issue.Labels) {
			continue
		}
		if issue.Status != model.StatusPlanned && issue.Status != model.StatusDoing && issue.Status != model.StatusBlocked {
			continue
		}
		children := childrenByParent[issue.ID]
		if len(children) == 0 {
			continue
		}

		ep := epicProgress{
			IssueID:       issue.ID,
			Title:         issue.Title,
			ChildrenTotal: len(children),
		}
		var mostRecentChildStatusChange *time.Time
		for _, child := range children {
			pts := child.Estimate
			if pts == 0 {
				pts = 1
			}
			ep.PointsTotal += pts
			if child.Status == model.StatusDone {
				ep.ChildrenDone++
				ep.PointsDone += pts
			}
			// Walk events for the latest status-change to an active status.
			// Transitions *to* BACKLOG don't count as activity (deprioritization
			// or initial assignment); children still sitting in BACKLOG with no
			// non-BACKLOG transition contribute nothing.
			for i := range child.Events {
				evt := child.Events[i]
				if evt.Type != model.EventTypeUpdate {
					continue
				}
				payloadBytes, _ := json.Marshal(evt.Payload)
				var p model.UpdatePayload
				if err := json.Unmarshal(payloadBytes, &p); err != nil || p.Status == nil {
					continue
				}
				if *p.Status == string(model.StatusBacklog) {
					continue
				}
				if mostRecentChildStatusChange == nil || evt.CreatedAt.After(*mostRecentChildStatusChange) {
					t := evt.CreatedAt
					mostRecentChildStatusChange = &t
				}
			}
		}
		ep.PointsRemaining = ep.PointsTotal - ep.PointsDone
		// Stale: at least one child has had a status change and the most recent
		// was more than the staleness threshold ago. Brand-new epics with no
		// status activity yet get the benefit of the doubt.
		if mostRecentChildStatusChange != nil && mostRecentChildStatusChange.Before(staleThreshold) {
			ep.Stale = true
		}
		out = append(out, ep)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PointsRemaining != out[j].PointsRemaining {
			return out[i].PointsRemaining > out[j].PointsRemaining
		}
		return out[i].Title < out[j].Title
	})

	return out
}

func hasEpicLabel(labels []string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, "epic") {
			return true
		}
	}
	return false
}

// latestStatusTimestamps walks an issue's events and returns the timestamps of
// the most recent UPDATE event that set the status to DONE / DOING / BLOCKED.
// Auto-progressed parents (no explicit status event) return nil for the relevant fields.
func latestStatusTimestamps(issue *model.Issue) (done, doing, blocked *time.Time) {
	for i := range issue.Events {
		evt := issue.Events[i]
		if evt.Type != model.EventTypeUpdate {
			continue
		}
		payloadBytes, _ := json.Marshal(evt.Payload)
		var p model.UpdatePayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil || p.Status == nil {
			continue
		}
		t := evt.CreatedAt
		switch *p.Status {
		case string(model.StatusDone):
			done = &t
		case string(model.StatusDoing):
			doing = &t
		case string(model.StatusBlocked):
			blocked = &t
		}
	}
	return
}

// startOfWeek returns midnight on the Monday of t's week (local time).
func startOfWeek(t time.Time) time.Time {
	wd := int(t.Weekday())
	// Go's Weekday: Sunday=0, Monday=1, ..., Saturday=6. We want Monday as start.
	offset := wd - 1
	if offset < 0 {
		offset = 6
	}
	return time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, t.Location())
}
