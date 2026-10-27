package exponential

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// --- assignment on add ---

func TestAddIssue_MilestoneRefIsResolved(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "1.5")
	iss := addIssueIn(t, tr, "x", "", "1.5")
	if iss.MilestoneID != a.ID || readAllIssues(t)[iss.ID].MilestoneID != a.ID {
		t.Errorf("milestone = %q, want %q", iss.MilestoneID, a.ID)
	}
}

func TestAddIssue_PlaceholderRefSavesIt(t *testing.T) {
	tr := setupLocalTransport(t)
	iss := addIssueIn(t, tr, "x", "", "@current")
	cur, _ := CurrentNext(readMilestones(t))
	if cur.Placeholder || iss.MilestoneID != cur.ID {
		t.Errorf("assigning to the Current placeholder saves it: %+v, issue %q", cur, iss.MilestoneID)
	}
}

func TestAddIssue_CompletedMilestoneRejected(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	tr.CompleteMilestone(a.ID, model.MilestoneRemove, "")
	if _, err := tr.AddIssue(model.CreatePayload{Title: "x", MilestoneID: a.ID}); err == nil {
		t.Error("assigning to a COMPLETED milestone must fail")
	}
}

func TestAddIssue_ChildCopiesParentsOpenMilestone(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	b := mustCreateMilestone(t, tr, "b")
	epic := addIssueIn(t, tr, "epic", "", a.ID)

	child, err := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: epic.ID})
	if err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[child.ID].MilestoneID != a.ID {
		t.Fatal("a new child copies its parent's milestone")
	}

	// A later parent change does not change the child.
	if _, err := tr.UpdateIssue(epic.ID, model.UpdatePayload{MilestoneID: sp(b.ID)}, "update"); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[child.ID].MilestoneID != a.ID {
		t.Error("the child keeps its own milestone")
	}

	// An explicit "none" wins over the copy.
	other, _ := tr.AddIssue(model.CreatePayload{Title: "c2", ParentID: epic.ID, MilestoneID: "none"})
	if readAllIssues(t)[other.ID].MilestoneID != "" {
		t.Error("milestone none must not copy the parent's milestone")
	}
}

func TestAddIssue_ChildOfCompletedParentGetsNoMilestone(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	epic := addIssueIn(t, tr, "epic", "DONE", a.ID)
	tr.CompleteMilestone(a.ID, model.MilestoneRemove, "")

	child, err := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: epic.ID})
	if err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[child.ID].MilestoneID != "" {
		t.Error("a COMPLETED milestone is not copied")
	}
}

// --- assignment on update ---

func TestUpdateIssue_MilestoneRefs(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	iss := addIssueIn(t, tr, "x", "", "")

	if _, err := tr.UpdateIssue(iss.ID, model.UpdatePayload{MilestoneID: sp("#1")}, "update"); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[iss.ID].MilestoneID != a.ID {
		t.Fatal("#1 should resolve to milestone a")
	}
	if _, err := tr.UpdateIssue(iss.ID, model.UpdatePayload{MilestoneID: sp("@next")}, "update"); err != nil {
		t.Fatal(err)
	}
	next := milestoneByNumber(t, 2)
	if next.Placeholder || readAllIssues(t)[iss.ID].MilestoneID != next.ID {
		t.Error("@next saves the placeholder and assigns the issue")
	}
	if _, err := tr.UpdateIssue(iss.ID, model.UpdatePayload{MilestoneID: sp("none")}, "update"); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[iss.ID].MilestoneID != "" {
		t.Error("none removes the issue from its milestone")
	}
}

func TestUpdateIssue_CompletedMilestoneRejected(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	done := addIssueIn(t, tr, "done", "DONE", a.ID)
	other := addIssueIn(t, tr, "other", "", "")
	tr.CompleteMilestone(a.ID, model.MilestoneRemove, "")

	if _, err := tr.UpdateIssue(other.ID, model.UpdatePayload{MilestoneID: sp(a.ID)}, "update"); err == nil {
		t.Error("assigning to a COMPLETED milestone must fail")
	}
	// Re-sending the milestone an issue already has is a no-op, not an error.
	if _, err := tr.UpdateIssue(done.ID, model.UpdatePayload{MilestoneID: sp(a.ID), Title: sp("renamed")}, "update"); err != nil {
		t.Errorf("unchanged COMPLETED milestone should pass: %v", err)
	}
}

func TestUpdateIssue_ReopenKeepsCompletedMilestone(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	done := addIssueIn(t, tr, "done", "DONE", a.ID)
	tr.CompleteMilestone(a.ID, model.MilestoneRemove, "")
	if _, err := tr.UpdateIssue(done.ID, model.UpdatePayload{Status: sp("PLANNED")}, "reopen"); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[done.ID].MilestoneID != a.ID {
		t.Error("a reopened issue keeps its COMPLETED milestone")
	}
}

func TestUpdateIssue_WithChildrenMovesOpenChildrenOnly(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	b := mustCreateMilestone(t, tr, "b")
	epic := addIssueIn(t, tr, "epic", "", a.ID)
	open, _ := tr.AddIssue(model.CreatePayload{Title: "open", ParentID: epic.ID})
	done, _ := tr.AddIssue(model.CreatePayload{Title: "done", Status: "DONE", ParentID: epic.ID})
	grand, _ := tr.AddIssue(model.CreatePayload{Title: "grand", ParentID: open.ID})

	if _, err := tr.UpdateIssueOpts(epic.ID, model.UpdatePayload{MilestoneID: sp(b.ID)}, "update", UpdateOptions{WithChildren: true}); err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	if issues[epic.ID].MilestoneID != b.ID || issues[open.ID].MilestoneID != b.ID || issues[grand.ID].MilestoneID != b.ID {
		t.Error("the epic and its open descendants move")
	}
	if issues[done.ID].MilestoneID != a.ID {
		t.Error("done children never move")
	}
}

func TestUpdateIssue_WithoutChildrenMovesOnlyTheIssue(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	b := mustCreateMilestone(t, tr, "b")
	epic := addIssueIn(t, tr, "epic", "", a.ID)
	child, _ := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: epic.ID})
	tr.UpdateIssue(epic.ID, model.UpdatePayload{MilestoneID: sp(b.ID)}, "update")
	if readAllIssues(t)[child.ID].MilestoneID != a.ID {
		t.Error("without with_children only the epic moves")
	}
}

func TestUpdateIssue_ReparentKeepsMilestone(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	b := mustCreateMilestone(t, tr, "b")
	epic := addIssueIn(t, tr, "epic", "", b.ID)
	iss := addIssueIn(t, tr, "x", "", a.ID)
	tr.UpdateIssue(iss.ID, model.UpdatePayload{ParentID: sp(epic.ID)}, "update")
	if readAllIssues(t)[iss.ID].MilestoneID != a.ID {
		t.Error("moving under a new parent keeps the issue's milestone")
	}
}

// --- list filter ---

func TestListIssues_MilestoneFilter(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	in := addIssueIn(t, tr, "in", "", a.ID)
	out := addIssueIn(t, tr, "out", "", "")

	ids := func(ref string) map[string]bool {
		t.Helper()
		list, err := tr.ListIssues(FilterOptions{Milestone: ref})
		if err != nil {
			t.Fatalf("list %q: %v", ref, err)
		}
		m := map[string]bool{}
		for _, i := range list {
			m[i.ID] = true
		}
		return m
	}
	if got := ids("@current"); !got[in.ID] || got[out.ID] {
		t.Errorf("@current = %v", got)
	}
	if got := ids("#1"); !got[in.ID] || got[out.ID] {
		t.Errorf("#1 = %v", got)
	}
	if got := ids("none"); got[in.ID] || !got[out.ID] {
		t.Errorf("none = %v", got)
	}
	if got := ids("@next"); len(got) != 0 {
		t.Errorf("an unsaved Next holds no issues, got %v", got)
	}
	if _, err := tr.ListIssues(FilterOptions{Milestone: "nope"}); err == nil {
		t.Error("unknown milestone ref should fail")
	}
}

// --- progress ---

func TestMilestoneCounts_SumsPointsOfTheScope(t *testing.T) {
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	rid := "ms-a"
	events := []model.Event{
		relEvent(rid, model.EventTypeMilestoneCreate, at, model.MilestoneCreatePayload{Number: 1}),
		// The epic's points are its children's: it is left out.
		{ID: "x-epic", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "epic", MilestoneID: rid, Estimate: 8}, CreatedAt: at},
		{ID: "x-1", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "1", ParentID: "x-epic", MilestoneID: rid, Status: "DONE", Estimate: 3}, CreatedAt: at},
		{ID: "x-2", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "2", MilestoneID: rid, Status: "DOING", Estimate: 5}, CreatedAt: at},
		// Unestimated: adds 0.
		{ID: "x-3", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "3", MilestoneID: rid}, CreatedAt: at},
		{ID: "x-4", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "4", MilestoneID: rid, Status: "CANCELED", Estimate: 13}, CreatedAt: at},
		{ID: "x-5", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "5", Estimate: 21}, CreatedAt: at},
	}
	r := ProjectMilestones(events)[0]
	s := MilestoneCounts(ProjectIssues(events), r)
	if s.Points == nil || *s.Points != (model.MilestonePoints{Total: 8, Done: 3}) {
		t.Errorf("points = %+v, want 3/8 (no epic, no canceled, no other milestone)", s.Points)
	}
}

func TestMilestoneCounts_PastMilestoneReadsPointsFromItsSnapshot(t *testing.T) {
	r := &model.Milestone{ID: "ms-a", State: model.MilestoneCompleted, Stats: &model.MilestoneStats{
		Counts: map[string]int{"DONE": 2},
		Points: &model.MilestonePoints{Total: 5, Done: 5},
	}}
	if s := MilestoneCounts(map[string]*model.Issue{}, r); s.Points == nil || *s.Points != (model.MilestonePoints{Total: 5, Done: 5}) {
		t.Errorf("points = %+v, want the snapshot's 5/5", s.Points)
	}
	r.Stats.Points = nil
	if s := MilestoneCounts(map[string]*model.Issue{}, r); s.Points != nil {
		t.Errorf("a snapshot without points has none, got %+v", s.Points)
	}
}

func TestComputeMilestoneProgress_CountsLeavesAndBurnsDown(t *testing.T) {
	day := func(d int, h int) time.Time { return time.Date(2026, 10, d, h, 0, 0, 0, time.UTC) }
	started := day(1, 9)
	rid := "ms-a"
	events := []model.Event{
		relEvent(rid, model.EventTypeMilestoneCreate, day(1, 8), model.MilestoneCreatePayload{Number: 1}),
		relEvent(rid, model.EventTypeMilestoneUpdate, started, model.MilestoneUpdatePayload{StartedAt: &started}),
		{ID: "x-epic", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "epic", MilestoneID: rid}, CreatedAt: day(1, 10)},
		{ID: "x-1", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "1", ParentID: "x-epic", MilestoneID: rid}, CreatedAt: day(1, 11)},
		{ID: "x-2", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "2", MilestoneID: rid}, CreatedAt: day(1, 12)},
		{ID: "x-1", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: sp("DONE")}, CreatedAt: day(2, 10)},
		// Scope added mid-milestone.
		{ID: "x-3", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "3", MilestoneID: rid}, CreatedAt: day(3, 10)},
		// Canceled and duplicate issues leave the scope.
		{ID: "x-4", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "4", MilestoneID: rid}, CreatedAt: day(3, 11)},
		{ID: "x-4", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: sp("CANCELED")}, CreatedAt: day(4, 10)},
		{ID: "x-2", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: sp("DUPLICATE")}, CreatedAt: day(4, 11)},
	}
	r := ProjectMilestones(events)[0]
	p := ComputeMilestoneProgress(events, r, day(4, 20))

	if p.Total != 2 || p.Done != 1 || p.Counts["DONE"] != 1 || p.Counts["BACKLOG"] != 1 || len(p.Counts) != 2 {
		t.Errorf("counts = %+v total %d done %d; the epic, canceled and duplicate issues must not count", p.Counts, p.Total, p.Done)
	}
	want := []MilestoneDay{
		{Date: "2026-10-01", Scope: 2, Open: 2, Completed: 0},
		{Date: "2026-10-02", Scope: 2, Open: 1, Completed: 1},
		{Date: "2026-10-03", Scope: 4, Open: 3, Completed: 1},
		{Date: "2026-10-04", Scope: 2, Open: 1, Completed: 1},
	}
	if len(p.Days) != len(want) {
		t.Fatalf("days = %+v", p.Days)
	}
	for i := range want {
		if p.Days[i] != want[i] {
			t.Errorf("day %d = %+v, want %+v", i, p.Days[i], want[i])
		}
	}
}

func TestComputeMilestoneProgress_NeverCurrentHasNoBurndown(t *testing.T) {
	events := []model.Event{
		relEvent("ms-a", model.EventTypeMilestoneCreate, time.Now(), model.MilestoneCreatePayload{Number: 1}),
	}
	p := ComputeMilestoneProgress(events, ProjectMilestones(events)[0], time.Now())
	if len(p.Days) != 0 {
		t.Errorf("no started_at, no burndown: %+v", p.Days)
	}
}

func TestComputeMilestoneProgress_CompletedUsesSnapshot(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	addIssueIn(t, tr, "d1", "DONE", a.ID)
	addIssueIn(t, tr, "d2", "DONE", a.ID)
	tr.CompleteMilestone(a.ID, model.MilestoneRemove, "")

	// Simulate `xpo archive`: the done issues' events are gone.
	var kept []model.Event
	for _, e := range mustReadEvents(t) {
		if model.IsMilestoneEvent(e.Type) {
			kept = append(kept, e)
		}
	}
	r := ProjectMilestones(kept)[0]
	p := ComputeMilestoneProgress(kept, r, time.Now())
	if p.Total != 2 || p.Done != 2 {
		t.Errorf("COMPLETED progress must come from the snapshot: %+v", p)
	}
}

func TestBuildTimeline_NamesTheMilestone(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "1.5.0")
	iss := addIssueIn(t, tr, "x", "", "")
	tr.UpdateIssue(iss.ID, model.UpdatePayload{MilestoneID: sp(a.ID)}, "update")

	events := mustReadEvents(t)
	entries := BuildTimeline(events, ProjectIssues(events), 0, "issue_event")
	for _, e := range entries {
		if model.IsMilestoneEvent(model.EventType(e.EventType)) {
			t.Errorf("milestone events are not issue activity: %+v", e)
		}
	}
	if len(entries) == 0 || entries[0].Milestone != "1.5.0" {
		t.Errorf("latest entry should name the milestone: %+v", entries)
	}
}

// --- one feed entry per bulk operation ---

func payloadOf[T any](t *testing.T, evt model.Event) T {
	t.Helper()
	var p T
	decodePayload(evt.Payload, &p)
	return p
}

func TestCompleteMilestone_MovesAreCausedAndCounted(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "1.5")
	b := mustCreateMilestone(t, tr, "1.6")
	addIssueIn(t, tr, "o1", "", a.ID)
	addIssueIn(t, tr, "o2", "", a.ID)
	tr.CompleteMilestone(a.ID, "", "")

	events := mustReadEvents(t)
	var moves int
	var complete model.MilestoneCompletePayload
	for _, e := range events {
		if e.Type == model.EventTypeUpdate && e.Cause == a.ID {
			moves++
		}
		if e.Type == model.EventTypeMilestoneComplete {
			complete = payloadOf[model.MilestoneCompletePayload](t, e)
		}
	}
	if moves != 2 || complete.Moved != 2 || complete.MovedTo != b.ID {
		t.Errorf("moves=%d payload=%+v", moves, complete)
	}

	entries := BuildTimeline(events, ProjectIssues(events), 0, "issue_event")
	if len(entries) == 0 || entries[0].EventType != string(model.EventTypeMilestoneComplete) || entries[0].Milestone != "1.5" || entries[0].MovedTo != "1.6" {
		t.Fatalf("the feed leads with one milestone entry: %+v", entries)
	}
	for _, e := range entries {
		if e.EventType == string(model.EventTypeUpdate) && e.Cause != "" {
			t.Errorf("caused moves stay out of the feed: %+v", e)
		}
	}
}

func TestDeleteMilestone_MovesAreCausedAndCounted(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "1.5")
	addIssueIn(t, tr, "o1", "DONE", a.ID)
	tr.DeleteMilestone(a.ID, model.MilestoneRemove, "")
	for _, e := range mustReadEvents(t) {
		if e.Type == model.EventTypeMilestoneDelete {
			if p := payloadOf[model.MilestoneDeletePayload](t, e); p.Moved != 1 || p.MovedTo != "" {
				t.Errorf("delete payload = %+v", p)
			}
		}
		if e.Type == model.EventTypeUpdate && payloadOf[model.UpdatePayload](t, e).MilestoneID != nil && *payloadOf[model.UpdatePayload](t, e).MilestoneID == "" && e.Cause != a.ID {
			t.Errorf("the removal is caused by the delete: %+v", e)
		}
	}
}

func TestWithChildren_OneEntryForTheEpic(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	b := mustCreateMilestone(t, tr, "b")
	epic := addIssueIn(t, tr, "epic", "", a.ID)
	c1, _ := tr.AddIssue(model.CreatePayload{Title: "c1", ParentID: epic.ID})
	tr.AddIssue(model.CreatePayload{Title: "c2", ParentID: epic.ID})

	if _, err := tr.UpdateIssueOpts(epic.ID, model.UpdatePayload{MilestoneID: sp(b.ID)}, "update", UpdateOptions{WithChildren: true}); err != nil {
		t.Fatal(err)
	}
	events := mustReadEvents(t)
	var epicUpdate *model.UpdatePayload
	for _, e := range events {
		if e.Type != model.EventTypeUpdate {
			continue
		}
		p := payloadOf[model.UpdatePayload](t, e)
		switch e.ID {
		case epic.ID:
			epicUpdate = &p
		case c1.ID:
			if e.Cause != epic.ID {
				t.Errorf("child move must be caused by the epic: %+v", e)
			}
		}
	}
	if epicUpdate == nil || epicUpdate.Children == nil || epicUpdate.Children.Count != 2 || epicUpdate.Children.MilestoneID != b.ID {
		t.Fatalf("epic update = %+v", epicUpdate)
	}
	feed := BuildTimeline(events, ProjectIssues(events), 0, "issue_event")
	if feed[0].IssueID != epic.ID || feed[0].Milestone != "b" {
		t.Errorf("one feed entry for the epic: %+v", feed[0])
	}
	if feed[1].EventType == string(model.EventTypeUpdate) {
		t.Errorf("no per-child entries: %+v", feed[1])
	}
}

func TestWithChildren_EpicAlreadyThereStillGetsAnEntry(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "a")
	b := mustCreateMilestone(t, tr, "b")
	epic := addIssueIn(t, tr, "epic", "", b.ID)
	child := addIssueIn(t, tr, "child", "", a.ID)
	tr.UpdateIssue(child.ID, model.UpdatePayload{ParentID: sp(epic.ID)}, "update")

	if _, err := tr.UpdateIssueOpts(epic.ID, model.UpdatePayload{MilestoneID: sp(b.ID)}, "update", UpdateOptions{WithChildren: true}); err != nil {
		t.Fatal(err)
	}
	events := mustReadEvents(t)
	last := events[len(events)-2] // the epic's update, then the caused child move
	p := payloadOf[model.UpdatePayload](t, last)
	if last.ID != epic.ID || p.MilestoneID != nil || p.Children == nil || p.Children.Count != 1 {
		t.Errorf("epic update = %+v %+v", last, p)
	}
	if readAllIssues(t)[child.ID].MilestoneID != b.ID {
		t.Error("the child moved")
	}
}

func TestBuildInbox_MilestoneIsTeamWideAndMovesAreQuiet(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateMilestone(t, tr, "1.5")
	mustCreateMilestone(t, tr, "1.6")
	mine := addIssueIn(t, tr, "mine", "", a.ID)
	tr.UpdateIssue(mine.ID, model.UpdatePayload{Assignee: sp("Me <me@example.com>")}, "update")
	tr.CompleteMilestone(a.ID, "", "")

	events := mustReadEvents(t)
	items := BuildInbox(events, ProjectIssues(events), "Me <me@example.com>", time.Time{})
	var milestone, moves int
	for _, it := range items {
		switch {
		case it.Type == model.EventTypeMilestoneComplete:
			milestone++
			if it.IssueTitle != "1.5" {
				t.Errorf("milestone entry title = %q", it.IssueTitle)
			}
		case it.IssueID == mine.ID && it.Type == model.EventTypeUpdate:
			if p := payloadOf[model.UpdatePayload](t, model.Event{Payload: it.Payload}); p.MilestoneID != nil {
				moves++
			}
		}
	}
	if milestone != 1 || moves != 0 {
		t.Errorf("inbox: %d milestone entries, %d move entries; want 1 and 0", milestone, moves)
	}
	other := BuildInbox(events, ProjectIssues(events), "Other <o@example.com>", time.Time{})
	found := false
	for _, it := range other {
		found = found || it.Type == model.EventTypeMilestoneComplete
	}
	if !found {
		t.Error("a milestone is team-wide news, like a merge")
	}
}
