package server

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func makeMetricIssue(id string, status model.IssueStatus, opts ...func(*model.Issue)) *model.Issue {
	i := &model.Issue{
		ID:        id,
		Title:     id,
		Status:    status,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	for _, o := range opts {
		o(i)
	}
	// Add a status-setting event so latestStatusTimestamps works
	statusStr := string(status)
	i.Events = append(i.Events, model.Event{
		ID: id, Type: model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Status: &statusStr},
		CreatedAt: i.UpdatedAt,
	})
	return i
}

func withEstimate(e int) func(*model.Issue) {
	return func(i *model.Issue) { i.Estimate = e }
}

func withAssignee(a string) func(*model.Issue) {
	return func(i *model.Issue) { i.Assignee = a }
}

func withLabels(l ...string) func(*model.Issue) {
	return func(i *model.Issue) { i.Labels = l }
}

func withPriority(p int) func(*model.Issue) {
	return func(i *model.Issue) { i.Priority = p }
}

func withCreatedAt(t time.Time) func(*model.Issue) {
	return func(i *model.Issue) { i.CreatedAt = t }
}

func withUpdatedAt(t time.Time) func(*model.Issue) {
	return func(i *model.Issue) { i.UpdatedAt = t }
}

func withParent(p string) func(*model.Issue) {
	return func(i *model.Issue) { i.ParentID = p }
}

func TestMetrics_Empty(t *testing.T) {
	m := computePulseMetrics(map[string]*model.Issue{}, time.Now())
	if m.Velocity.Last7dPoints != 0 {
		t.Errorf("velocity should be 0 for empty")
	}
	if m.WIP.Total != 0 {
		t.Errorf("WIP should be 0 for empty")
	}
}

func TestMetrics_WIP(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusDoing, withUpdatedAt(now)),
		"b": makeMetricIssue("b", model.StatusDoing, withUpdatedAt(now.Add(-10*24*time.Hour))),
		"c": makeMetricIssue("c", model.StatusPlanned),
	}

	m := computePulseMetrics(issues, now)
	if m.WIP.Total != 2 {
		t.Errorf("WIP total = %d, want 2", m.WIP.Total)
	}
	if m.WIP.Stale != 1 {
		t.Errorf("WIP stale = %d, want 1 (b is stale)", m.WIP.Stale)
	}
}

func TestMetrics_Blockers(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusBlocked, withUpdatedAt(now.Add(-72*time.Hour))),
		"b": makeMetricIssue("b", model.StatusBlocked, withUpdatedAt(now.Add(-24*time.Hour))),
		"c": makeMetricIssue("c", model.StatusDoing),
	}

	m := computePulseMetrics(issues, now)
	if m.Blockers.Total != 2 {
		t.Errorf("blockers total = %d, want 2", m.Blockers.Total)
	}
	if m.Blockers.OldestDays < 2 {
		t.Errorf("oldest blocker = %d days, want ≥2", m.Blockers.OldestDays)
	}
}

func TestMetrics_Throughput(t *testing.T) {
	// Use a fixed Wednesday so calendar-week boundaries are predictable.
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.Local) // Wednesday
	lastWeekWed := now.AddDate(0, 0, -7)                     // last Wed = inside last completed week
	priorWeekWed := now.AddDate(0, 0, -14)                   // two weeks ago = inside prior week
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusDone, withUpdatedAt(lastWeekWed)),
		"b": makeMetricIssue("b", model.StatusDone, withUpdatedAt(priorWeekWed)),
		"c": makeMetricIssue("c", model.StatusDoing),
	}

	m := computePulseMetrics(issues, now)
	if m.Throughput.Last7d != 1 {
		t.Errorf("throughput last week = %d, want 1 (only a)", m.Throughput.Last7d)
	}
}

func TestMetrics_Velocity(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.Local) // Wednesday
	lastWeekWed := now.AddDate(0, 0, -7)
	priorWeekWed := now.AddDate(0, 0, -14)
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusDone, withEstimate(3), withUpdatedAt(lastWeekWed)),
		"b": makeMetricIssue("b", model.StatusDone, withEstimate(5), withUpdatedAt(priorWeekWed)),
	}

	m := computePulseMetrics(issues, now)
	if m.Velocity.Last7dPoints != 3 {
		t.Errorf("velocity last week = %d, want 3 (only a)", m.Velocity.Last7dPoints)
	}
	if m.Velocity.Prior7dPoints != 5 {
		t.Errorf("velocity prior week = %d, want 5 (only b)", m.Velocity.Prior7dPoints)
	}
}

func TestMetrics_Attention_Blockers(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusBlocked, withUpdatedAt(now.Add(-48*time.Hour))),
	}

	m := computePulseMetrics(issues, now)
	found := false
	for _, item := range m.Attention {
		if item.IssueID == "a" && item.Kind == attentionBlocker {
			found = true
		}
	}
	if !found {
		t.Error("blocked issue should appear in attention items")
	}
}

func TestMetrics_Attention_StaleWIP(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusDoing, withUpdatedAt(now.Add(-10*24*time.Hour))),
	}

	m := computePulseMetrics(issues, now)
	found := false
	for _, item := range m.Attention {
		if item.IssueID == "a" && item.Kind == attentionStaleWIP {
			found = true
		}
	}
	if !found {
		t.Error("stale WIP issue should appear in attention items")
	}
}

func TestMetrics_Attention_HighPriority(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusPlanned, withPriority(1)),
	}

	m := computePulseMetrics(issues, now)
	found := false
	for _, item := range m.Attention {
		if item.IssueID == "a" && item.Kind == attentionHighPriority {
			found = true
		}
	}
	if !found {
		t.Error("high-priority issue should appear in attention items")
	}
}

func TestMetrics_Workload(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeMetricIssue("a", model.StatusDoing, withAssignee("Alice <a@b.com>"), withEstimate(3)),
		"c": makeMetricIssue("c", model.StatusBlocked, withAssignee("Alice <a@b.com>")),
	}

	m := computePulseMetrics(issues, now)
	if len(m.Workload) != 1 {
		t.Fatalf("expected 1 workload entry, got %d", len(m.Workload))
	}
	w := m.Workload[0]
	if w.InProgress < 1 {
		t.Errorf("in_progress = %d, want ≥1", w.InProgress)
	}
	if w.Blocked != 1 {
		t.Errorf("blocked = %d, want 1", w.Blocked)
	}
}

func TestMetrics_Epics(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"epic": makeMetricIssue("epic", model.StatusDoing, withLabels("epic"), withEstimate(0)),
		"c1":   makeMetricIssue("c1", model.StatusDone, withParent("epic"), withEstimate(3), withUpdatedAt(now)),
		"c2":   makeMetricIssue("c2", model.StatusPlanned, withParent("epic"), withEstimate(5)),
	}

	m := computePulseMetrics(issues, now)
	if len(m.Epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(m.Epics))
	}
	e := m.Epics[0]
	if e.ChildrenDone != 1 || e.ChildrenTotal != 2 {
		t.Errorf("children: %d/%d, want 1/2", e.ChildrenDone, e.ChildrenTotal)
	}
	if e.PointsDone != 3 || e.PointsTotal != 8 {
		t.Errorf("points: %d/%d, want 3/8", e.PointsDone, e.PointsTotal)
	}
}

func TestMetrics_CycleTime(t *testing.T) {
	now := time.Now()
	doingStr := "DOING"
	doneStr := "DONE"
	issues := map[string]*model.Issue{
		"a": {
			ID: "a", Title: "a", Status: model.StatusDone,
			CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now,
			Events: []model.Event{
				{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: &doingStr}, CreatedAt: now.Add(-48 * time.Hour)},
				{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: &doneStr}, CreatedAt: now.Add(-24 * time.Hour)},
			},
		},
	}
	m := computePulseMetrics(issues, now)
	if m.Flow.CycleCount != 1 {
		t.Errorf("CycleCount = %d, want 1", m.Flow.CycleCount)
	}
	if m.Flow.CycleTimeHrs < 20 || m.Flow.CycleTimeHrs > 28 {
		t.Errorf("CycleTimeHrs = %f, want ~24", m.Flow.CycleTimeHrs)
	}
}

func TestMetrics_LeadTime(t *testing.T) {
	now := time.Now()
	doneStr := "DONE"
	issues := map[string]*model.Issue{
		"a": {
			ID: "a", Title: "a", Status: model.StatusDone,
			CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now,
			Events: []model.Event{
				{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: &doneStr}, CreatedAt: now},
			},
		},
	}
	m := computePulseMetrics(issues, now)
	if m.Flow.LeadCount != 1 {
		t.Errorf("LeadCount = %d, want 1", m.Flow.LeadCount)
	}
	if m.Flow.LeadTimeHrs < 68 || m.Flow.LeadTimeHrs > 76 {
		t.Errorf("LeadTimeHrs = %f, want ~72", m.Flow.LeadTimeHrs)
	}
}

func TestMetrics_Staleness(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"fresh":  makeMetricIssue("fresh", model.StatusPlanned, withCreatedAt(now.Add(-12*time.Hour))),
		"days3":  makeMetricIssue("days3", model.StatusDoing, withCreatedAt(now.Add(-2*24*time.Hour))),
		"week":   makeMetricIssue("week", model.StatusPlanned, withCreatedAt(now.Add(-5*24*time.Hour))),
		"old":    makeMetricIssue("old", model.StatusBlocked, withCreatedAt(now.Add(-20*24*time.Hour))),
		"ancient": makeMetricIssue("ancient", model.StatusBacklog, withCreatedAt(now.Add(-60*24*time.Hour))),
	}
	m := computePulseMetrics(issues, now)
	if m.Flow.StalenessTotal != 5 {
		t.Errorf("StalenessTotal = %d, want 5", m.Flow.StalenessTotal)
	}
	if m.Flow.Staleness.Under1d != 1 {
		t.Errorf("Under1d = %d, want 1", m.Flow.Staleness.Under1d)
	}
	if m.Flow.Staleness.Over30d != 1 {
		t.Errorf("Over30d = %d, want 1", m.Flow.Staleness.Over30d)
	}
}

func TestMetrics_BugAge(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"new_bug": makeMetricIssue("new_bug", model.StatusBacklog, withLabels("bug"), withCreatedAt(now.Add(-12*time.Hour))),
		"old_bug": makeMetricIssue("old_bug", model.StatusDoing, withLabels("bug"), withCreatedAt(now.Add(-40*24*time.Hour))),
	}
	m := computePulseMetrics(issues, now)
	if m.Trends.BugAge.Under24h != 1 {
		t.Errorf("BugAge.Under24h = %d, want 1", m.Trends.BugAge.Under24h)
	}
	if m.Trends.BugAge.Over1mo != 1 {
		t.Errorf("BugAge.Over1mo = %d, want 1", m.Trends.BugAge.Over1mo)
	}
}

func TestMetrics_VelocityExcludesParents(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.Local) // Wednesday
	lastWeekTue := now.AddDate(0, 0, -8)
	issues := map[string]*model.Issue{
		"epic": makeMetricIssue("epic", model.StatusDone, withLabels("epic"), withEstimate(0), withUpdatedAt(lastWeekTue)),
		"c1":   makeMetricIssue("c1", model.StatusDone, withParent("epic"), withEstimate(5), withUpdatedAt(lastWeekTue)),
		"c2":   makeMetricIssue("c2", model.StatusDone, withParent("epic"), withEstimate(8), withUpdatedAt(lastWeekTue)),
	}

	m := computePulseMetrics(issues, now)
	if m.Velocity.Last7dPoints != 13 {
		t.Errorf("velocity last week = %d, want 13 (children only, not parent)", m.Velocity.Last7dPoints)
	}
	if m.Throughput.Last7d != 2 {
		t.Errorf("throughput last week = %d, want 2 (children only)", m.Throughput.Last7d)
	}
}

func TestMetrics_VelocityExcludesCurrentWeek(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.Local) // Wednesday
	lastWeekWed := now.AddDate(0, 0, -7)
	issues := map[string]*model.Issue{
		"this_week": makeMetricIssue("this_week", model.StatusDone, withEstimate(10), withUpdatedAt(now.Add(-1*time.Hour))),
		"last_week": makeMetricIssue("last_week", model.StatusDone, withEstimate(3), withUpdatedAt(lastWeekWed)),
	}

	m := computePulseMetrics(issues, now)
	if m.Velocity.Last7dPoints != 3 {
		t.Errorf("velocity last week = %d, want 3 (current week excluded)", m.Velocity.Last7dPoints)
	}
	if m.Velocity.CurrentWeekPoints != 10 {
		t.Errorf("velocity current week = %d, want 10", m.Velocity.CurrentWeekPoints)
	}
}

func TestMetrics_AttentionCappedAt5(t *testing.T) {
	now := time.Now()
	issues := make(map[string]*model.Issue)
	for i := 0; i < 10; i++ {
		id := "blocked-" + string(rune('a'+i))
		issues[id] = makeMetricIssue(id, model.StatusBlocked, withUpdatedAt(now.Add(-48*time.Hour)))
	}

	m := computePulseMetrics(issues, now)
	if len(m.Attention) > 5 {
		t.Errorf("attention items = %d, should be capped at 5", len(m.Attention))
	}
}
