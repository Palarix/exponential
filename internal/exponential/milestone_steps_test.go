package exponential

import (
	"reflect"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// stepFixture builds an event log for a milestone started at 09:00 on
// 2026-10-01. Events added with at are one write each, an hour apart unless
// batched with batch.
type stepFixture struct {
	rid    string
	events []model.Event
	clock  time.Time
}

func newStepFixture() *stepFixture {
	base := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	started := base.Add(time.Hour)
	f := &stepFixture{rid: "ms-a", clock: started}
	f.events = []model.Event{
		relEvent(f.rid, model.EventTypeMilestoneCreate, base, model.MilestoneCreatePayload{Number: 1}),
		relEvent(f.rid, model.EventTypeMilestoneUpdate, started, model.MilestoneUpdatePayload{StartedAt: &started}),
	}
	return f
}

// before adds events before the milestone started.
func (f *stepFixture) before(evts ...model.Event) {
	at := f.events[0].CreatedAt
	for i := range evts {
		evts[i].CreatedAt = at.Add(time.Duration(i+1) * time.Minute)
		evts[i].CreatedBy = "T <t@x>"
	}
	// Keep the started_at update last.
	started := f.events[1]
	f.events = append(append(f.events[:1], evts...), started)
}

// write adds one write: the events get times 1 ns apart, an hour after the
// previous write.
func (f *stepFixture) write(evts ...model.Event) {
	f.clock = f.clock.Add(time.Hour)
	for i := range evts {
		evts[i].CreatedAt = f.clock.Add(time.Duration(i) * time.Nanosecond)
		if evts[i].CreatedBy == "" {
			evts[i].CreatedBy = "T <t@x>"
		}
	}
	f.events = append(f.events, evts...)
}

func (f *stepFixture) progress(t *testing.T) MilestoneProgress {
	t.Helper()
	r := ProjectMilestones(f.events)[0]
	return ComputeMilestoneProgress(f.events, r, f.clock.Add(time.Hour))
}

func stepCreate(id, title string, p model.CreatePayload) model.Event {
	p.Title = title
	return model.Event{ID: id, Type: model.EventTypeCreate, Payload: p}
}

func stepUpdate(id string, p model.UpdatePayload) model.Event {
	return model.Event{ID: id, Type: model.EventTypeUpdate, Payload: p}
}

type stepCounts struct{ scope, open, completed, scopePts, openPts, completedPts int }

func countsOf(s MilestoneStep) stepCounts {
	return stepCounts{s.Scope, s.Open, s.Completed, s.ScopePoints, s.OpenPoints, s.CompletedPoints}
}

func TestSteps_FirstStepIsTheStateAtStart(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid, Estimate: 3}),
		stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid}),
	)
	p := f.progress(t)
	if len(p.Steps) != 1 {
		t.Fatalf("steps = %+v", p.Steps)
	}
	s := p.Steps[0]
	if got, want := countsOf(s), (stepCounts{2, 2, 0, 4, 4, 0}); got != want {
		t.Errorf("step 0 = %+v, want %+v", got, want)
	}
	if !s.At.Equal(*ProjectMilestones(f.events)[0].StartedAt) || s.Changes == nil || len(s.Changes) != 0 {
		t.Errorf("step 0 is at started_at with no changes: %+v", s)
	}
}

func TestSteps_OneStepPerWriteWithItsChanges(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid}),
		stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid}),
		stepCreate("x-3", "three", model.CreatePayload{MilestoneID: f.rid}),
	)
	f.write(stepUpdate("x-1", model.UpdatePayload{Status: sp("DONE")}))
	// A batch: two issues close in one write.
	f.write(
		stepUpdate("x-2", model.UpdatePayload{Status: sp("DONE")}),
		stepUpdate("x-3", model.UpdatePayload{Status: sp("DONE")}),
	)
	p := f.progress(t)
	if len(p.Steps) != 3 {
		t.Fatalf("want 3 steps (start + 2 writes), got %+v", p.Steps)
	}
	if got := countsOf(p.Steps[1]); got != (stepCounts{3, 2, 1, 3, 2, 1}) {
		t.Errorf("step 1 = %+v", got)
	}
	if got := countsOf(p.Steps[2]); got != (stepCounts{3, 0, 3, 3, 0, 3}) {
		t.Errorf("step 2 = %+v", got)
	}
	want := []MilestoneChange{{IssueID: "x-2", Title: "two", Kind: ChangeDone}, {IssueID: "x-3", Title: "three", Kind: ChangeDone}}
	if !reflect.DeepEqual(p.Steps[2].Changes, want) {
		t.Errorf("changes = %+v, want %+v", p.Steps[2].Changes, want)
	}
}

func TestSteps_SeparateActorsAreSeparateWrites(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid}),
		stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid}),
	)
	a := stepUpdate("x-1", model.UpdatePayload{Status: sp("DONE")})
	b := stepUpdate("x-2", model.UpdatePayload{Status: sp("DONE")})
	b.CreatedBy = "Other <o@x>"
	f.write(a, b)
	if p := f.progress(t); len(p.Steps) != 3 {
		t.Errorf("two actors make two steps, got %+v", p.Steps)
	}
}

func TestSteps_NoOpWritesAddNoStep(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid}),
		stepCreate("x-9", "elsewhere", model.CreatePayload{}),
	)
	f.write(stepUpdate("x-1", model.UpdatePayload{Title: sp("renamed")}))
	f.write(stepUpdate("x-1", model.UpdatePayload{Status: sp("DOING")}))
	f.write(stepUpdate("x-9", model.UpdatePayload{Status: sp("DONE")}))
	if p := f.progress(t); len(p.Steps) != 1 {
		t.Errorf("title, open-to-open and other-milestone writes add no step: %+v", p.Steps)
	}
}

func TestSteps_ReopenAddedRemovedCanceled(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid, Status: "DONE"}),
		stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid}),
		stepCreate("x-3", "three", model.CreatePayload{MilestoneID: f.rid}),
		stepCreate("x-5", "five", model.CreatePayload{MilestoneID: f.rid}),
	)
	f.write(stepUpdate("x-5", model.UpdatePayload{Status: sp("DONE")}))
	f.write(stepUpdate("x-1", model.UpdatePayload{Status: sp("PLANNED")}))
	f.write(stepCreate("x-4", "four", model.CreatePayload{MilestoneID: f.rid}))
	f.write(stepUpdate("x-2", model.UpdatePayload{MilestoneID: sp("")}))
	f.write(stepUpdate("x-3", model.UpdatePayload{Status: sp("CANCELED")}))
	f.write(model.Event{ID: "x-4", Type: model.EventTypeDelete})

	p := f.progress(t)
	kinds := []string{}
	for _, s := range p.Steps[2:] {
		if len(s.Changes) != 1 {
			t.Fatalf("one change per step: %+v", s)
		}
		kinds = append(kinds, s.Changes[0].Kind)
	}
	want := []string{ChangeReopened, ChangeAdded, ChangeRemoved, ChangeCanceled, ChangeRemoved}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("kinds = %v, want %v", kinds, want)
	}
	if got := countsOf(p.Steps[len(p.Steps)-1]); got != (stepCounts{2, 1, 1, 2, 1, 1}) {
		t.Errorf("last step = %+v", got)
	}
}

func TestSteps_EstimateChangesArePointSteps(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid, Estimate: 5}),
		stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid, Status: "DONE"}),
	)
	f.write(stepUpdate("x-1", model.UpdatePayload{Estimate: ip(8)}))
	// Removing the estimate counts the issue as 1 point.
	f.write(stepUpdate("x-1", model.UpdatePayload{Estimate: ip(0)}))
	// Setting the estimate to 1 changes nothing.
	f.write(stepUpdate("x-1", model.UpdatePayload{Estimate: ip(1)}))
	p := f.progress(t)
	if len(p.Steps) != 3 {
		t.Fatalf("steps = %+v", p.Steps)
	}
	if p.Steps[1].ScopePoints != 9 || p.Steps[2].ScopePoints != 2 || p.Steps[2].Scope != 2 {
		t.Errorf("points = %d, %d", p.Steps[1].ScopePoints, p.Steps[2].ScopePoints)
	}
	if p.Steps[1].Changes[0].Kind != ChangeEstimated {
		t.Errorf("kind = %q", p.Steps[1].Changes[0].Kind)
	}
}

func TestSteps_ParentGettingAChildIsOneStep(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-epic", "epic", model.CreatePayload{MilestoneID: f.rid}),
		stepCreate("x-d", "done", model.CreatePayload{MilestoneID: f.rid, Status: "DONE"}),
	)
	f.write(stepCreate("x-1", "child", model.CreatePayload{MilestoneID: f.rid, ParentID: "x-epic"}))
	p := f.progress(t)
	if len(p.Steps) != 1 {
		t.Errorf("the scope stays 2 issues, 2 points: no step, got %+v", p.Steps)
	}
	f.write(stepUpdate("x-1", model.UpdatePayload{Estimate: ip(3)}))
	p = f.progress(t)
	if len(p.Steps) != 2 || p.Steps[1].ScopePoints != 4 {
		t.Errorf("steps = %+v", p.Steps)
	}
}

func TestSteps_ParentSwapListsRemovedAndAdded(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-epic", "epic", model.CreatePayload{MilestoneID: f.rid, Estimate: 5}),
		stepCreate("x-d", "done", model.CreatePayload{MilestoneID: f.rid, Status: "DONE"}),
	)
	f.write(stepCreate("x-1", "child", model.CreatePayload{MilestoneID: f.rid, ParentID: "x-epic", Estimate: 2}))
	p := f.progress(t)
	if len(p.Steps) != 2 {
		t.Fatalf("steps = %+v", p.Steps)
	}
	want := []MilestoneChange{{IssueID: "x-1", Title: "child", Kind: ChangeAdded}, {IssueID: "x-epic", Title: "epic", Kind: ChangeRemoved}}
	if !reflect.DeepEqual(p.Steps[1].Changes, want) {
		t.Errorf("changes = %+v", p.Steps[1].Changes)
	}
}

func TestSteps_MilestoneCompleteIsOneStep(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	b := mustCreateMilestone(t, tr, "b")
	first := addIssueIn(t, tr, "first", "", a.ID)
	for range 4 {
		addIssueIn(t, tr, "open", "", a.ID)
	}
	// Writes under 1 ms apart group: keep the work and the complete apart.
	time.Sleep(2 * time.Millisecond)
	if _, err := tr.UpdateIssue(first.ID, model.UpdatePayload{Status: sp("DONE")}, "update"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	before := len(ComputeMilestoneProgress(mustReadEvents(t), milestoneByNumber(t, 1), time.Now()).Steps)
	if _, _, err := tr.CompleteMilestone(a.ID, "", b.ID); err != nil {
		t.Fatal(err)
	}
	events := mustReadEvents(t)
	r, _ := ResolveMilestone(ProjectMilestones(events), a.ID)
	p := ComputeMilestoneProgress(events, r, time.Now())
	if len(p.Steps) != before+1 {
		t.Fatalf("the complete adds one step: %d → %d", before, len(p.Steps))
	}
	last := p.Steps[len(p.Steps)-1]
	if len(last.Changes) != 4 || last.Scope != 1 || last.Open != 0 {
		t.Errorf("last step = %+v", last)
	}
	for _, c := range last.Changes {
		if c.Kind != ChangeRemoved {
			t.Errorf("change = %+v", c)
		}
	}
}

func TestSteps_IssuesMovedInWithTheStartAreTheFirstStep(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	mustCreateMilestone(t, tr, "b")
	for range 3 {
		addIssueIn(t, tr, "open", "", a.ID)
	}
	// Completing a moves its issues to b and makes b Current: the moves and
	// b's started_at are one write.
	if _, _, err := tr.CompleteMilestone(a.ID, "", ""); err != nil {
		t.Fatal(err)
	}
	p := ComputeMilestoneProgress(mustReadEvents(t), milestoneByNumber(t, 2), time.Now())
	if len(p.Steps) != 1 || p.Steps[0].Scope != 3 {
		t.Errorf("steps = %+v", p.Steps)
	}
}

func TestSteps_PlaceholderAndNeverStartedHaveNone(t *testing.T) {
	events := []model.Event{relEvent("ms-a", model.EventTypeMilestoneCreate, time.Now(), model.MilestoneCreatePayload{Number: 1})}
	p := ComputeMilestoneProgress(events, ProjectMilestones(events)[0], time.Now())
	if p.Steps == nil || len(p.Steps) != 0 {
		t.Errorf("steps = %#v, want empty", p.Steps)
	}
}

func TestDays_CarryPoints(t *testing.T) {
	f := newStepFixture()
	f.before(
		stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid, Estimate: 3}),
		stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid}),
	)
	f.write(stepUpdate("x-1", model.UpdatePayload{Status: sp("DONE")}))
	p := f.progress(t)
	if len(p.Days) != 1 {
		t.Fatalf("days = %+v", p.Days)
	}
	d := p.Days[0]
	if d.ScopePoints != 4 || d.OpenPoints != 1 || d.CompletedPoints != 3 {
		t.Errorf("day = %+v", d)
	}
}

func TestSteps_SetupFoldsIntoTheFirstStep(t *testing.T) {
	f := newStepFixture()
	// The milestone starts empty and is filled afterwards, one write each.
	f.write(stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid}))
	f.write(stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid}))
	f.write(stepUpdate("x-2", model.UpdatePayload{Estimate: ip(5)}))
	f.write(stepUpdate("x-1", model.UpdatePayload{Status: sp("DOING")}))
	p := f.progress(t)
	if len(p.Steps) != 1 {
		t.Fatalf("nothing done yet: only the first step, got %+v", p.Steps)
	}
	if got := countsOf(p.Steps[0]); got != (stepCounts{2, 2, 0, 6, 6, 0}) {
		t.Errorf("the first step is the scope so far: %+v", got)
	}
	if !p.Steps[0].At.Equal(*ProjectMilestones(f.events)[0].StartedAt) || len(p.Steps[0].Changes) != 0 {
		t.Errorf("the first step stays at started_at with no changes: %+v", p.Steps[0])
	}

	// Work begins with the first issue done; scope added after it is a step.
	f.write(stepUpdate("x-1", model.UpdatePayload{Status: sp("DONE")}))
	f.write(stepCreate("x-3", "three", model.CreatePayload{MilestoneID: f.rid}))
	p = f.progress(t)
	if len(p.Steps) != 3 || p.Steps[1].Changes[0].Kind != ChangeDone || p.Steps[2].Changes[0].Kind != ChangeAdded {
		t.Errorf("steps = %+v", p.Steps)
	}
}

func TestSteps_CancelBeginsTheWork(t *testing.T) {
	f := newStepFixture()
	f.write(stepCreate("x-1", "one", model.CreatePayload{MilestoneID: f.rid}))
	f.write(stepCreate("x-2", "two", model.CreatePayload{MilestoneID: f.rid}))
	f.write(stepUpdate("x-2", model.UpdatePayload{Status: sp("CANCELED")}))
	p := f.progress(t)
	if len(p.Steps) != 2 || p.Steps[0].Scope != 2 || p.Steps[1].Changes[0].Kind != ChangeCanceled {
		t.Errorf("steps = %+v", p.Steps)
	}
}
