package exponential

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// addLinked creates A and B with `A <kind> B` stored on A.
func addLinked(t *testing.T, tr *LocalTransport, kind model.DependencyKind) (a, b *model.Issue) {
	t.Helper()
	b, err := tr.AddIssue(model.CreatePayload{Title: "B"})
	if err != nil {
		t.Fatal(err)
	}
	a, err = tr.AddIssue(model.CreatePayload{
		Title:        "A",
		Dependencies: []model.Dependency{{TargetID: b.ID, Kind: kind}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return a, b
}

func TestProjectIssues_DerivesInverseLinks(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	issues := readAllIssues(t)
	want := model.Dependency{SourceID: b.ID, TargetID: a.ID, Kind: model.DependencyBlockedBy, Derived: true}
	if links := issues[b.ID].Links; len(links) != 1 || links[0] != want {
		t.Errorf("B links = %+v, want [%+v]", links, want)
	}
	if deps := issues[b.ID].Dependencies; len(deps) != 0 {
		t.Errorf("B owned deps = %+v, want none", deps)
	}
	wantA := model.Dependency{SourceID: a.ID, TargetID: b.ID, Kind: model.DependencyBlocks}
	if links := issues[a.ID].Links; len(links) != 1 || links[0] != wantA {
		t.Errorf("A links = %+v, want [%+v]", links, wantA)
	}
}

func TestProjectIssues_RelatesToIsSymmetric(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, "relates_to")

	links := readAllIssues(t)[b.ID].Links
	if len(links) != 1 || links[0].TargetID != a.ID || links[0].Kind != "relates_to" || !links[0].Derived {
		t.Errorf("B links = %+v, want derived relates_to %s", links, a.ID)
	}
}

func TestProjectIssues_DedupesBothSides(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)
	// Record the same relationship from B's side by writing the event directly,
	// the way pre-bidirectional data looks.
	if err := tr.appendEvent(model.Event{ID: b.ID, Type: model.EventTypeUpdate, Payload: model.UpdatePayload{
		Dependencies: []model.Dependency{{SourceID: b.ID, TargetID: a.ID, Kind: model.DependencyBlockedBy}},
	}}); err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	for id, issue := range map[string]*model.Issue{a.ID: issues[a.ID], b.ID: issues[b.ID]} {
		if len(issue.Links) != 1 || issue.Links[0].Derived {
			t.Errorf("%s links = %+v, want one owned entry", id, issue.Links)
		}
	}
}

func TestProjectIssues_SkipsLinksFromDeletedIssues(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)
	if err := tr.DeleteIssue(a.ID, "gone", false); err != nil {
		t.Fatal(err)
	}
	if links := readAllIssues(t)[b.ID].Links; len(links) != 0 {
		t.Errorf("B links = %+v, want none", links)
	}
}

func TestUpdateIssue_DerivedBlocksGatesStart(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	_, err := tr.UpdateIssue(b.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err == nil || !strings.Contains(err.Error(), a.ID) {
		t.Fatalf("expected start of B to be blocked by %s, got %v", a.ID, err)
	}

	if _, err := tr.UpdateIssue(a.ID, model.UpdatePayload{Status: sp("DONE")}, "update"); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.UpdateIssue(b.ID, model.UpdatePayload{Status: sp("DOING")}, "update"); err != nil {
		t.Errorf("B should start once A is DONE, got %v", err)
	}
}

func TestUpdateIssue_EmptyLinksRemovesDerived(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	if _, err := tr.UpdateIssue(b.ID, model.UpdatePayload{Dependencies: []model.Dependency{}}, "update"); err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	if deps := issues[a.ID].Dependencies; len(deps) != 0 {
		t.Errorf("A deps = %+v, want the blocks row removed", deps)
	}
	if links := issues[b.ID].Links; len(links) != 0 {
		t.Errorf("B links = %+v, want none", links)
	}
}

func TestUpdateIssue_RemovingLinkRemovesBothSides(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)
	if err := tr.appendEvent(model.Event{ID: b.ID, Type: model.EventTypeUpdate, Payload: model.UpdatePayload{
		Dependencies: []model.Dependency{{SourceID: b.ID, TargetID: a.ID, Kind: model.DependencyBlockedBy}},
	}}); err != nil {
		t.Fatal(err)
	}

	if _, err := tr.UpdateIssue(b.ID, model.UpdatePayload{Dependencies: []model.Dependency{}}, "update"); err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	if len(issues[a.ID].Links) != 0 || len(issues[b.ID].Links) != 0 {
		t.Errorf("links remain: A=%+v B=%+v", issues[a.ID].Links, issues[b.ID].Links)
	}
}

func TestUpdateIssue_SendingCombinedLinksBackIsNoop(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)
	c, _ := tr.AddIssue(model.CreatePayload{Title: "C"})
	if _, err := tr.UpdateIssue(b.ID, model.UpdatePayload{
		Dependencies: append(readAllIssues(t)[b.ID].Links, model.Dependency{TargetID: c.ID, Kind: model.DependencyDependsOn}),
	}, "update"); err != nil {
		t.Fatal(err)
	}
	beforeA, beforeB := countEvents(t, a.ID), countEvents(t, b.ID)

	links := readAllIssues(t)[b.ID].Links
	if len(links) != 2 {
		t.Fatalf("B links = %+v, want 2", links)
	}
	if _, err := tr.UpdateIssue(b.ID, model.UpdatePayload{Dependencies: links}, "update"); err != nil {
		t.Fatal(err)
	}
	if countEvents(t, a.ID) != beforeA || countEvents(t, b.ID) != beforeB {
		t.Errorf("expected no new events when sending combined links back")
	}
}

func TestUpdateIssue_AddingLinkKeepsDerived(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)
	c, _ := tr.AddIssue(model.CreatePayload{Title: "C"})

	links := append(readAllIssues(t)[b.ID].Links, model.Dependency{TargetID: c.ID, Kind: model.DependencyDependsOn})
	if _, err := tr.UpdateIssue(b.ID, model.UpdatePayload{Dependencies: links}, "update"); err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	if deps := issues[a.ID].Dependencies; len(deps) != 1 {
		t.Errorf("A deps = %+v, want blocks row kept", deps)
	}
	owned := issues[b.ID].Dependencies
	if len(owned) != 1 || owned[0].TargetID != c.ID {
		t.Errorf("B owned deps = %+v, want only depends_on %s", owned, c.ID)
	}
}

func TestUpdateIssue_DerivedFlagNotPersisted(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})

	if _, err := tr.UpdateIssue(a.ID, model.UpdatePayload{
		Dependencies: []model.Dependency{{TargetID: b.ID, Kind: model.DependencyBlocks, Derived: true}},
	}, "update"); err != nil {
		t.Fatal(err)
	}
	events, _ := storage.ReadEvents()
	for _, evt := range events {
		raw, _ := json.Marshal(evt.Payload)
		if strings.Contains(string(raw), "derived") {
			t.Errorf("event payload persisted derived flag: %s", raw)
		}
	}
	if links := readAllIssues(t)[b.ID].Links; len(links) != 1 || !links[0].Derived {
		t.Errorf("B links = %+v, want derived blocked_by", links)
	}
}

func TestHasUnresolvedBlockers_Derived(t *testing.T) {
	for _, kind := range []model.DependencyKind{model.DependencyBlocks, model.DependencyDependencyOf} {
		t.Run(string(kind), func(t *testing.T) {
			tr := setupLocalTransport(t)
			_, b := addLinked(t, tr, kind)
			c := &Client{Config: tr.Config, Transport: tr, local: tr}
			if !hasUnresolvedBlockers(readAllIssues(t)[b.ID], c) {
				t.Errorf("B should be blocked by A via derived %s", model.InverseKind(kind))
			}
		})
	}
}

func TestAPIIssueToModel_SplitsDerivedLinks(t *testing.T) {
	issue := apiIssueToModel(apiIssue{ID: "b", Dependencies: []apiDependency{
		{SourceID: "b", TargetID: "c", Kind: "depends_on"},
		{SourceID: "b", TargetID: "a", Kind: "blocked_by", Derived: true},
	}})
	if len(issue.Links) != 2 {
		t.Errorf("links = %+v, want 2", issue.Links)
	}
	if len(issue.Dependencies) != 1 || issue.Dependencies[0].TargetID != "c" {
		t.Errorf("owned deps = %+v, want only depends_on c", issue.Dependencies)
	}
}
