package exponential

import (
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func mustReadEvents(t *testing.T) []model.Event {
	t.Helper()
	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func readReleases(t *testing.T) []*model.Release {
	t.Helper()
	return ProjectReleases(mustReadEvents(t))
}

func mustCreateRelease(t *testing.T, tr *LocalTransport, label string) *model.Release {
	t.Helper()
	r, err := tr.CreateRelease(ReleaseInput{Label: &label})
	if err != nil {
		t.Fatalf("create release %q: %v", label, err)
	}
	return r
}

func releaseByNumber(t *testing.T, n int) *model.Release {
	t.Helper()
	for _, r := range readReleases(t) {
		if r.Number == n {
			return r
		}
	}
	t.Fatalf("no release #%d", n)
	return nil
}

func openDisplays(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, r := range OpenReleases(readReleases(t)) {
		out = append(out, r.Display())
	}
	return out
}

func relEvent(id string, typ model.EventType, at time.Time, payload interface{}) model.Event {
	return model.Event{ID: id, Type: typ, Payload: payload, CreatedAt: at, CreatedBy: "T <t@x>"}
}

// --- projection ---

func TestProjectReleases_DuplicateNumberGoesToLaterRelease(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	// File order after a union merge need not match created_at.
	events := []model.Event{
		relEvent("rel-b", model.EventTypeReleaseCreate, t0.Add(2*time.Hour), model.ReleaseCreatePayload{Number: 1, SortOrder: "a1"}),
		relEvent("rel-a", model.EventTypeReleaseCreate, t0.Add(time.Hour), model.ReleaseCreatePayload{Number: 1, SortOrder: "a0"}),
		relEvent("rel-c", model.EventTypeReleaseCreate, t0.Add(3*time.Hour), model.ReleaseCreatePayload{Number: 2, SortOrder: "a2"}),
	}
	got := map[string]int{}
	for _, r := range ProjectReleases(events) {
		got[r.ID] = r.Number
	}
	if got["rel-a"] != 1 || got["rel-c"] != 2 || got["rel-b"] != 3 {
		t.Errorf("numbers = %v, want rel-a=1 rel-c=2 rel-b=3", got)
	}
}

func TestProjectReleases_StatesAndFields(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	goal := "ship it"
	events := []model.Event{
		relEvent("rel-a", model.EventTypeReleaseCreate, t0, model.ReleaseCreatePayload{Number: 1, Label: "1.0"}),
		relEvent("rel-a", model.EventTypeReleaseUpdate, t0.Add(time.Minute), model.ReleaseUpdatePayload{Goal: &goal}),
		relEvent("rel-a", model.EventTypeReleaseComplete, t0.Add(time.Hour), model.ReleaseCompletePayload{Stats: model.ReleaseStats{Counts: map[string]int{"DONE": 2}}}),
		relEvent("rel-b", model.EventTypeReleaseCreate, t0, model.ReleaseCreatePayload{Number: 2}),
		relEvent("rel-b", model.EventTypeReleaseDelete, t0.Add(time.Hour), model.ReleaseDeletePayload{Issues: model.ReleaseRemove}),
	}
	rs := ProjectReleases(events)
	if len(rs) != 2 {
		t.Fatalf("got %d releases, want 2 (deleted ones included)", len(rs))
	}
	a, b := rs[0], rs[1]
	if a.State != model.ReleaseReleased || a.Goal != "ship it" || a.ReleasedAt == nil || a.Stats == nil || a.Stats.Counts["DONE"] != 2 {
		t.Errorf("release a = %+v", a)
	}
	if !b.Deleted || b.Display() != "#2" {
		t.Errorf("release b = %+v", b)
	}
	if len(OpenReleases(rs)) != 0 {
		t.Error("released and deleted releases are not open")
	}
}

func TestProjectIssues_IgnoresReleaseEvents(t *testing.T) {
	events := []model.Event{
		relEvent("rel-a", model.EventTypeReleaseCreate, time.Now(), model.ReleaseCreatePayload{Number: 1, Label: "1.0"}),
		{ID: "x-1", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "a", ReleaseID: "rel-a"}, CreatedAt: time.Now()},
	}
	issues := ProjectIssues(events)
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues["x-1"].ReleaseID != "rel-a" || issues["x-1"].ReleaseName != "1.0" {
		t.Errorf("issue release = %q / %q", issues["x-1"].ReleaseID, issues["x-1"].ReleaseName)
	}
}

// --- Current / Next and placeholders ---

func TestCurrentNext_PlaceholdersFillMissingSlots(t *testing.T) {
	cur, next := CurrentNext(nil)
	if !cur.Placeholder || cur.Number != 1 || !next.Placeholder || next.Number != 2 {
		t.Errorf("empty board: current=%+v next=%+v", cur, next)
	}
	if cur.Locked || !next.Locked {
		t.Errorf("Next is locked while Current is a placeholder: current=%+v next=%+v", cur, next)
	}

	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "1.5")
	cur, next = CurrentNext(readReleases(t))
	if cur.Placeholder || cur.Label != "1.5" || !next.Placeholder || next.Number != 2 || next.Locked {
		t.Errorf("one release: current=%+v next=%+v", cur, next)
	}
}

func TestCreateRelease_NumbersAreSequentialAndNeverReused(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	b := mustCreateRelease(t, tr, "b")
	if a.Number != 1 || b.Number != 2 {
		t.Fatalf("numbers = %d, %d", a.Number, b.Number)
	}
	if _, err := tr.DeleteRelease(b.ID, model.ReleaseRemove, ""); err != nil {
		t.Fatal(err)
	}
	c := mustCreateRelease(t, tr, "c")
	if c.Number != 3 {
		t.Errorf("number after delete = %d, want 3", c.Number)
	}
	if !strings.HasPrefix(c.ID, model.ReleaseIDPrefix) {
		t.Errorf("id %q lacks the rel- prefix", c.ID)
	}
}

func TestCreateRelease_DuplicateOpenLabelFails(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "1.5")
	label := "1.5"
	if _, err := tr.CreateRelease(ReleaseInput{Label: &label}); err == nil {
		t.Fatal("expected duplicate label error")
	}
	upper := "Team Mode"
	mustCreateRelease(t, tr, "team mode")
	if _, err := tr.CreateRelease(ReleaseInput{Label: &upper}); err == nil {
		t.Error("labels are unique regardless of case")
	}
}

func TestCreateRelease_LabelOfReleasedReleaseCanBeReused(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "1.5")
	if _, _, err := tr.CompleteRelease(a.ID, model.ReleaseRemove, ""); err != nil {
		t.Fatal(err)
	}
	mustCreateRelease(t, tr, "1.5")
}

func TestCreateRelease_RejectsLabelsThatLookLikeRefs(t *testing.T) {
	tr := setupLocalTransport(t)
	for _, l := range []string{"#7", "#beta", "@next", "None", "rel-abc"} {
		label := l
		if _, err := tr.CreateRelease(ReleaseInput{Label: &label}); err == nil {
			t.Errorf("label %q should be rejected", l)
		}
	}
}

func TestCreateRelease_NumericLabelsAreLabels(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "first")
	year := mustCreateRelease(t, tr, "2026")
	seven := mustCreateRelease(t, tr, "R1")
	rs := readReleases(t)
	if r, err := ResolveRelease(rs, "2026"); err != nil || r.ID != year.ID {
		t.Errorf("2026 = %+v, %v", r, err)
	}
	if r, err := ResolveRelease(rs, "r1"); err != nil || r.ID != seven.ID {
		t.Errorf("R1 is a label lookup: %+v, %v", r, err)
	}
	if r, err := ResolveRelease(rs, "#1"); err != nil || r.Label != "first" {
		t.Errorf("#1 is release number 1: %+v, %v", r, err)
	}
}

func TestCreateRelease_RejectsBadTargetDate(t *testing.T) {
	tr := setupLocalTransport(t)
	d := "next tuesday"
	if _, err := tr.CreateRelease(ReleaseInput{TargetDate: &d}); err == nil {
		t.Error("target date must be YYYY-MM-DD")
	}
}

func TestStartedAt_SetWhenSavedReleaseBecomesCurrent(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	b := mustCreateRelease(t, tr, "b")
	if releaseByNumber(t, a.Number).StartedAt == nil {
		t.Error("the first release becomes Current and gets started_at")
	}
	if releaseByNumber(t, b.Number).StartedAt != nil {
		t.Error("Next has no started_at")
	}
}

func TestUpdateRelease_PlaceholderIsSavedOnFirstWrite(t *testing.T) {
	tr := setupLocalTransport(t)
	goal := "first goal"
	r, err := tr.UpdateRelease("@current", ReleaseInput{Goal: &goal})
	if err != nil {
		t.Fatal(err)
	}
	if r.Placeholder || r.Number != 1 || r.Goal != "first goal" {
		t.Errorf("saved = %+v", r)
	}
	if len(readReleases(t)) != 1 {
		t.Errorf("want 1 stored release")
	}
	if releaseByNumber(t, 1).StartedAt == nil {
		t.Error("a placeholder saved as Current gets started_at")
	}
}

func TestLockedNext_RejectsWritesUntilCurrentIsSaved(t *testing.T) {
	tr := setupLocalTransport(t)
	label := "later"
	if _, err := tr.UpdateRelease("@next", ReleaseInput{Label: &label}); err == nil || !strings.Contains(err.Error(), "save Current first") {
		t.Errorf("update locked Next: err = %v", err)
	}
	if _, err := tr.UpdateRelease("#2", ReleaseInput{Label: &label}); err == nil {
		t.Error("the locked Next by number must fail too")
	}
	if _, err := tr.AddIssue(model.CreatePayload{Title: "x", ReleaseID: "@next"}); err == nil {
		t.Error("assigning to a locked Next must fail")
	}
	if len(readReleases(t)) != 0 || len(readAllIssues(t)) != 0 {
		t.Error("a refused write stores nothing")
	}

	// Once Current is saved, Next unlocks.
	mustCreateRelease(t, tr, "now")
	r, err := tr.UpdateRelease("@next", ReleaseInput{Label: &label})
	if err != nil || r.Number != 2 || r.Label != "later" {
		t.Errorf("unlocked Next: %+v, %v", r, err)
	}
}

func TestUpdateRelease_ChangesFields(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	label, goal, date := "1.6", "a goal", "2026-12-01"
	r, err := tr.UpdateRelease("#1", ReleaseInput{Label: &label, Goal: &goal, TargetDate: &date})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != a.ID || r.Label != "1.6" || r.Goal != "a goal" || r.TargetDate != "2026-12-01" {
		t.Errorf("updated = %+v", r)
	}
	mustCreateRelease(t, tr, "b")
	dup := "b"
	if _, err := tr.UpdateRelease(a.ID, ReleaseInput{Label: &dup}); err == nil {
		t.Error("renaming to another open release's label must fail")
	}
	same := "1.6"
	if _, err := tr.UpdateRelease(a.ID, ReleaseInput{Label: &same}); err != nil {
		t.Errorf("keeping its own label: %v", err)
	}
}

func TestPlaceholders_CannotBeCompletedReorderedOrDeleted(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "a")
	if _, _, err := tr.CompleteRelease("@next", model.ReleaseMoveToNext, ""); err == nil {
		t.Error("complete placeholder should fail")
	}
	if _, err := tr.DeleteRelease("@next", model.ReleaseMoveToNext, ""); err == nil {
		t.Error("delete placeholder should fail")
	}
	if _, err := tr.ReorderRelease("@next", "a", ""); err == nil {
		t.Error("reorder placeholder should fail")
	}
	if _, err := tr.ReorderRelease("a", "@next", ""); err == nil {
		t.Error("reorder relative to a placeholder should fail")
	}
	if len(readReleases(t)) != 1 {
		t.Error("failed writes must not save the placeholder")
	}
}

// --- reorder ---

func TestReorderRelease_MovesAndKeepsStartedAt(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "a")
	mustCreateRelease(t, tr, "b")
	mustCreateRelease(t, tr, "c")

	if _, err := tr.ReorderRelease("c", "a", ""); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(openDisplays(t), ","); got != "c,a,b" {
		t.Fatalf("order = %s, want c,a,b", got)
	}
	startedA := releaseByNumber(t, 1).StartedAt
	if startedA == nil || releaseByNumber(t, 3).StartedAt == nil {
		t.Error("c became Current and gets started_at; a keeps its own")
	}

	if _, err := tr.ReorderRelease("c", "", "b"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(openDisplays(t), ","); got != "a,b,c" {
		t.Fatalf("order = %s, want a,b,c", got)
	}
	if got := releaseByNumber(t, 1).StartedAt; got == nil || !got.Equal(*startedA) {
		t.Error("a reorder must not reset started_at")
	}
}

func TestReorderRelease_NeedsExactlyOneAnchor(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "a")
	mustCreateRelease(t, tr, "b")
	if _, err := tr.ReorderRelease("a", "", ""); err == nil {
		t.Error("missing anchor should fail")
	}
	if _, err := tr.ReorderRelease("a", "b", "b"); err == nil {
		t.Error("both anchors should fail")
	}
}

// --- complete ---

func addIssueIn(t *testing.T, tr *LocalTransport, title, status, release string) *model.Issue {
	t.Helper()
	iss, err := tr.AddIssue(model.CreatePayload{Title: title, Status: status, ReleaseID: release})
	if err != nil {
		t.Fatal(err)
	}
	return iss
}

func TestCompleteRelease_MovesOpenIssuesToNextAndKeepsDone(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	b := mustCreateRelease(t, tr, "b")
	done := addIssueIn(t, tr, "done", "DONE", a.ID)
	open := addIssueIn(t, tr, "open", "DOING", a.ID)
	canceled := addIssueIn(t, tr, "canceled", "CANCELED", a.ID)

	r, msgs, err := tr.CompleteRelease("a", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.State != model.ReleaseReleased || len(msgs) == 0 {
		t.Errorf("completed = %+v, msgs %v", r, msgs)
	}
	issues := readAllIssues(t)
	if issues[done.ID].ReleaseID != a.ID || issues[canceled.ID].ReleaseID != a.ID {
		t.Error("done and canceled issues stay in the completed release")
	}
	if issues[open.ID].ReleaseID != b.ID {
		t.Errorf("open issue moved to %q, want Next %q", issues[open.ID].ReleaseID, b.ID)
	}
	got := releaseByNumber(t, 1)
	if got.Stats == nil || got.Stats.Counts["DONE"] != 1 || len(got.Stats.Counts) != 1 || len(got.Stats.LeafIDs) != 1 || got.Stats.LeafIDs[0] != done.ID {
		t.Errorf("the snapshot holds only the scope (canceled left out): %+v", got.Stats)
	}
	if _, total, doneN := ReleaseCounts(readAllIssues(t), got); total != 1 || doneN != 1 {
		t.Errorf("a release whose remaining issues are done reads as complete: %d/%d", doneN, total)
	}
	if releaseByNumber(t, 2).StartedAt == nil {
		t.Error("Next became Current and gets started_at")
	}

	if _, _, err := tr.CompleteRelease(a.ID, "", ""); err == nil {
		t.Error("a RELEASED release cannot be completed again")
	}
	if _, err := tr.DeleteRelease(a.ID, "", ""); err == nil {
		t.Error("a RELEASED release cannot be deleted")
	}
	if _, err := tr.ReorderRelease(a.ID, "b", ""); err == nil {
		t.Error("a RELEASED release cannot be reordered")
	}
}

func TestCompleteRelease_RemoveClearsOpenIssues(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	open := addIssueIn(t, tr, "open", "PLANNED", a.ID)
	if _, _, err := tr.CompleteRelease(a.ID, model.ReleaseRemove, ""); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[open.ID].ReleaseID != "" {
		t.Error("remove clears the release of open issues")
	}
	if len(readReleases(t)) != 1 {
		t.Error("remove must not save a placeholder")
	}
}

func TestCompleteRelease_MoveToPlaceholderNextSavesIt(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	open := addIssueIn(t, tr, "open", "BACKLOG", a.ID)
	if _, _, err := tr.CompleteRelease(a.ID, model.ReleaseMoveToNext, ""); err != nil {
		t.Fatal(err)
	}
	saved := releaseByNumber(t, 2)
	if readAllIssues(t)[open.ID].ReleaseID != saved.ID {
		t.Error("open issue should move to the saved Next")
	}
	cur, _ := CurrentNext(readReleases(t))
	if cur.ID != saved.ID || saved.StartedAt == nil {
		t.Errorf("saved Next becomes Current with started_at: %+v", saved)
	}
}

func TestCompleteRelease_InvalidMode(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	if _, _, err := tr.CompleteRelease(a.ID, "shred", ""); err == nil {
		t.Error("unknown mode should fail")
	}
}

// --- delete ---

func TestDeleteRelease_MovesAllIssuesIncludingDone(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	b := mustCreateRelease(t, tr, "b")
	done := addIssueIn(t, tr, "done", "DONE", a.ID)
	open := addIssueIn(t, tr, "open", "DOING", a.ID)

	if _, err := tr.DeleteRelease("a", "", ""); err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	if issues[done.ID].ReleaseID != b.ID || issues[open.ID].ReleaseID != b.ID {
		t.Error("delete moves every issue, done included, to Next")
	}
	if !releaseByNumber(t, 1).Deleted {
		t.Error("release should be deleted")
	}
	if _, err := ResolveRelease(readReleases(t), "a"); err == nil {
		t.Error("a deleted release no longer resolves")
	}
}

func TestDeleteRelease_NextMovesToTheReleaseAfterIt(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "a")
	b := mustCreateRelease(t, tr, "b")
	c := mustCreateRelease(t, tr, "c")
	iss := addIssueIn(t, tr, "x", "BACKLOG", b.ID)
	if _, err := tr.DeleteRelease(b.ID, model.ReleaseMoveToNext, ""); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[iss.ID].ReleaseID != c.ID {
		t.Error("issues of a deleted Next move to the release after it")
	}
}

func TestDeleteRelease_LastMovesToTheReleaseBeforeIt(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	b := mustCreateRelease(t, tr, "b")
	iss := addIssueIn(t, tr, "x", "BACKLOG", b.ID)
	msgs, err := tr.DeleteRelease(b.ID, model.ReleaseMoveToNext, "")
	if err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[iss.ID].ReleaseID != a.ID {
		t.Error("issues of a deleted last release move to the release before it, not to a new one")
	}
	if _, next := CurrentNext(readReleases(t)); !next.Placeholder {
		t.Errorf("no release is saved for the move: next = %+v (%v)", next, msgs)
	}
}

func TestDeleteRelease_UpcomingMovesToItsNeighbour(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "a")
	mustCreateRelease(t, tr, "b")
	c := mustCreateRelease(t, tr, "c")
	d := mustCreateRelease(t, tr, "d")
	iss := addIssueIn(t, tr, "x", "BACKLOG", c.ID)
	if _, err := tr.DeleteRelease(c.ID, model.ReleaseMoveToNext, ""); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[iss.ID].ReleaseID != d.ID {
		t.Error("issues of a deleted Upcoming release move to the release after it")
	}
}

func TestCompleteRelease_ToMovesToTheGivenRelease(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	mustCreateRelease(t, tr, "b")
	c := mustCreateRelease(t, tr, "c")
	iss := addIssueIn(t, tr, "x", "BACKLOG", a.ID)
	_, msgs, err := tr.CompleteRelease(a.ID, model.ReleaseMoveToNext, "c")
	if err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[iss.ID].ReleaseID != c.ID {
		t.Errorf("open issues move to the release given by to: %v", msgs)
	}
}

func TestDeleteRelease_ToMovesToTheGivenRelease(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	b := mustCreateRelease(t, tr, "b")
	mustCreateRelease(t, tr, "c")
	iss := addIssueIn(t, tr, "x", "DONE", b.ID)
	if _, err := tr.DeleteRelease(b.ID, "", "#1"); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[iss.ID].ReleaseID != a.ID {
		t.Error("all issues move to the release given by to")
	}
}

func TestDeleteRelease_ToPlaceholderSavesIt(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	iss := addIssueIn(t, tr, "x", "BACKLOG", a.ID)
	if _, _, err := tr.CompleteRelease(a.ID, "", "@next"); err != nil {
		t.Fatal(err)
	}
	cur, _ := CurrentNext(readReleases(t))
	if cur.Placeholder || readAllIssues(t)[iss.ID].ReleaseID != cur.ID {
		t.Errorf("the Next placeholder is saved and takes the issues: %+v", cur)
	}
}

func TestReleaseMoveTo_Refusals(t *testing.T) {
	tr := setupLocalTransport(t)
	old := mustCreateRelease(t, tr, "old")
	tr.CompleteRelease(old.ID, model.ReleaseRemove, "")
	a := mustCreateRelease(t, tr, "a")
	mustCreateRelease(t, tr, "b")
	addIssueIn(t, tr, "x", "BACKLOG", a.ID)
	for _, tc := range []struct{ mode, to, why string }{
		{model.ReleaseMoveToNext, "a", "itself"},
		{model.ReleaseMoveToNext, "old", "a released release"},
		{model.ReleaseMoveToNext, "nope", "an unknown release"},
		{model.ReleaseMoveToNext, "none", "no release (use remove)"},
		{model.ReleaseRemove, "b", "to with remove"},
	} {
		if _, err := tr.DeleteRelease(a.ID, tc.mode, tc.to); err == nil {
			t.Errorf("delete moving to %s should fail", tc.why)
		}
		if _, _, err := tr.CompleteRelease(a.ID, tc.mode, tc.to); err == nil {
			t.Errorf("complete moving to %s should fail", tc.why)
		}
	}
}

func TestDeleteRelease_CurrentWithPlaceholderNext(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	iss := addIssueIn(t, tr, "x", "BACKLOG", a.ID)
	if _, err := tr.DeleteRelease(a.ID, model.ReleaseMoveToNext, ""); err != nil {
		t.Fatal(err)
	}
	cur, next := CurrentNext(readReleases(t))
	if cur.Placeholder || cur.Number != 2 || readAllIssues(t)[iss.ID].ReleaseID != cur.ID {
		t.Errorf("the saved placeholder becomes Current: %+v", cur)
	}
	if !next.Placeholder || next.Number != 3 {
		t.Errorf("a new placeholder becomes Next: %+v", next)
	}
}

func TestDeleteRelease_RemoveClearsAllIssues(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "a")
	done := addIssueIn(t, tr, "done", "DONE", a.ID)
	if _, err := tr.DeleteRelease(a.ID, model.ReleaseRemove, ""); err != nil {
		t.Fatal(err)
	}
	if readAllIssues(t)[done.ID].ReleaseID != "" {
		t.Error("remove clears every issue's release")
	}
}

// --- refs ---

func TestResolveRelease_Refs(t *testing.T) {
	tr := setupLocalTransport(t)
	a := mustCreateRelease(t, tr, "1.5")
	b, _ := tr.CreateRelease(ReleaseInput{})
	rs := readReleases(t)

	cases := map[string]string{
		"#1": a.ID, " #1 ": a.ID, "1.5": a.ID, a.ID: a.ID,
		"#2": b.ID, "@current": a.ID, "@NEXT": b.ID,
	}
	for ref, want := range cases {
		r, err := ResolveRelease(rs, ref)
		if err != nil || r == nil || r.ID != want {
			t.Errorf("ResolveRelease(%q) = %+v, %v; want %s", ref, r, err, want)
		}
	}
	for _, ref := range []string{"none", "", " NONE "} {
		if r, err := ResolveRelease(rs, ref); err != nil || r != nil {
			t.Errorf("ResolveRelease(%q) = %+v, %v; want nil", ref, r, err)
		}
	}
	for _, ref := range []string{"#9", "R1", "1", "nope", "rel-000000"} {
		if _, err := ResolveRelease(rs, ref); err == nil {
			t.Errorf("ResolveRelease(%q) should fail", ref)
		}
	}
}

func TestResolveRelease_PlaceholderRefs(t *testing.T) {
	tr := setupLocalTransport(t)
	mustCreateRelease(t, tr, "a")
	rs := readReleases(t)
	for _, ref := range []string{"@next", "#2"} {
		r, err := ResolveRelease(rs, ref)
		if err != nil || r == nil || !r.Placeholder || r.Number != 2 {
			t.Errorf("ResolveRelease(%q) = %+v, %v; want placeholder #2", ref, r, err)
		}
	}
}

func TestResolveRelease_OpenLabelBeatsReleased(t *testing.T) {
	tr := setupLocalTransport(t)
	old := mustCreateRelease(t, tr, "beta")
	tr.CompleteRelease(old.ID, model.ReleaseRemove, "")
	cur := mustCreateRelease(t, tr, "beta")
	r, err := ResolveRelease(readReleases(t), "Beta")
	if err != nil || r.ID != cur.ID {
		t.Errorf("got %+v, %v; want the OPEN release", r, err)
	}
}
