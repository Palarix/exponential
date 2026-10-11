package exponential

import (
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

// appendLegacyLinks stores deps on id directly, bypassing validation, to
// model a multi-link pair recorded before the one-link rule.
func appendLegacyLinks(t *testing.T, tr *LocalTransport, id string, deps ...model.Dependency) {
	t.Helper()
	for i := range deps {
		deps[i].SourceID = id
	}
	if err := tr.appendEvent(model.Event{ID: id, Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Dependencies: deps}}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateIssue_RejectsTwoKindsToSameTarget(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})

	_, err := tr.UpdateIssue(a.ID, model.UpdatePayload{Dependencies: []model.Dependency{
		{TargetID: b.ID, Kind: model.DependencyBlocks},
		{TargetID: b.ID, Kind: "relates_to"},
	}}, "update")
	if err == nil || !strings.Contains(err.Error(), "only one link is allowed between two issues") {
		t.Fatalf("err = %v, want one-link error", err)
	}
	if links := readAllIssues(t)[a.ID].Links; len(links) != 0 {
		t.Errorf("A links = %+v, want nothing written", links)
	}
}

func TestUpdateIssue_ChangesKindViaFullSet(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	// Replace "B blocked_by A" (derived on B) with "B relates_to A".
	if _, err := tr.UpdateIssue(b.ID, model.UpdatePayload{Dependencies: []model.Dependency{
		{TargetID: a.ID, Kind: "relates_to"},
	}}, "update"); err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	if deps := issues[a.ID].Dependencies; len(deps) != 0 {
		t.Errorf("A deps = %+v, want blocks row removed", deps)
	}
	if links := issues[b.ID].Links; len(links) != 1 || links[0].Kind != "relates_to" {
		t.Errorf("B links = %+v, want only relates_to", links)
	}
}

func TestUpdateIssue_KeepsUntouchedLegacyMultiLinkPair(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})
	c, _ := tr.AddIssue(model.CreatePayload{Title: "C"})
	appendLegacyLinks(t, tr, a.ID, model.Dependency{TargetID: b.ID, Kind: model.DependencyBlocks})
	appendLegacyLinks(t, tr, b.ID, model.Dependency{TargetID: a.ID, Kind: "relates_to"})

	links := readAllIssues(t)[a.ID].Links
	if len(links) != 2 {
		t.Fatalf("setup: A links = %+v, want 2", links)
	}
	desired := append(links, model.Dependency{TargetID: c.ID, Kind: model.DependencyDependsOn})
	if _, err := tr.UpdateIssue(a.ID, model.UpdatePayload{Dependencies: desired}, "update"); err != nil {
		t.Fatalf("unrelated update rejected: %v", err)
	}
	if got := readAllIssues(t)[a.ID].Links; len(got) != 3 {
		t.Errorf("A links = %+v, want legacy pair kept plus C", got)
	}
}

func TestUpdateIssue_RejectsAddingToLegacyMultiLinkPair(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})
	appendLegacyLinks(t, tr, a.ID,
		model.Dependency{TargetID: b.ID, Kind: model.DependencyBlocks},
		model.Dependency{TargetID: b.ID, Kind: "relates_to"})

	desired := append(readAllIssues(t)[a.ID].Links, model.Dependency{TargetID: b.ID, Kind: model.DependencyDuplicates})
	if _, err := tr.UpdateIssue(a.ID, model.UpdatePayload{Dependencies: desired}, "update"); err == nil {
		t.Fatal("expected one-link error")
	}
}

func TestAddIssue_RejectsTwoLinksToSameTarget(t *testing.T) {
	tr := setupLocalTransport(t)
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})

	_, err := tr.AddIssue(model.CreatePayload{Title: "A", Dependencies: []model.Dependency{
		{TargetID: b.ID, Kind: model.DependencyBlocks},
		{TargetID: b.ID, Kind: model.DependencyBlocks},
	}})
	if err == nil || !strings.Contains(err.Error(), "only one link") {
		t.Fatalf("err = %v, want one-link error", err)
	}
	if n := len(readAllIssues(t)); n != 1 {
		t.Errorf("issues = %d, want the rejected issue not created", n)
	}
}

func TestLink_AddsLink(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})

	added, replaced, _, err := localClient(tr).Link(a.ID, b.ID, "blocks", false)
	if err != nil {
		t.Fatal(err)
	}
	if added.SourceID != a.ID || added.TargetID != b.ID || added.Kind != model.DependencyBlocks || len(replaced) != 0 {
		t.Errorf("added = %+v replaced = %+v", added, replaced)
	}
	if deps := readAllIssues(t)[a.ID].Dependencies; len(deps) != 1 || deps[0].TargetID != b.ID {
		t.Errorf("A deps = %+v", deps)
	}
}

func TestLink_RefusesDifferentKindWithoutOverwrite(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	_, _, _, err := localClient(tr).Link(a.ID, b.ID, "relates_to", false)
	want := a.ID + " is already linked to " + b.ID + " (blocks); set overwrite to replace it with relates_to"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if links := readAllIssues(t)[a.ID].Links; len(links) != 1 || links[0].Kind != model.DependencyBlocks {
		t.Errorf("A links = %+v, want unchanged", links)
	}
}

func TestLink_OverwriteReplacesOwnedLink(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	_, replaced, _, err := localClient(tr).Link(a.ID, b.ID, "relates_to", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(replaced) != 1 || replaced[0].Kind != model.DependencyBlocks {
		t.Errorf("replaced = %+v, want blocks", replaced)
	}
	if links := readAllIssues(t)[a.ID].Links; len(links) != 1 || links[0].Kind != "relates_to" {
		t.Errorf("A links = %+v, want only relates_to", links)
	}
}

func TestLink_OverwriteReplacesLinkStoredOnTarget(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks) // stored on A

	_, replaced, _, err := localClient(tr).Link(b.ID, a.ID, "relates_to", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(replaced) != 1 || replaced[0].Kind != model.DependencyBlockedBy {
		t.Errorf("replaced = %+v, want blocked_by (as seen from B)", replaced)
	}
	issues := readAllIssues(t)
	if deps := issues[a.ID].Dependencies; len(deps) != 0 {
		t.Errorf("A deps = %+v, want blocks row removed", deps)
	}
	if links := issues[b.ID].Links; len(links) != 1 || links[0].Kind != "relates_to" {
		t.Errorf("B links = %+v, want only relates_to", links)
	}
}

func TestLink_OverwriteCollapsesLegacyPair(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})
	appendLegacyLinks(t, tr, a.ID,
		model.Dependency{TargetID: b.ID, Kind: model.DependencyBlocks},
		model.Dependency{TargetID: b.ID, Kind: "relates_to"})

	_, replaced, _, err := localClient(tr).Link(a.ID, b.ID, "blocks", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(replaced) != 1 || replaced[0].Kind != "relates_to" {
		t.Errorf("replaced = %+v, want relates_to", replaced)
	}
	if links := readAllIssues(t)[a.ID].Links; len(links) != 1 || links[0].Kind != model.DependencyBlocks {
		t.Errorf("A links = %+v, want only blocks", links)
	}
}

func TestLink_Errors(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)
	c := localClient(tr)

	cases := []struct {
		name, source, target, kind string
		overwrite                  bool
		want                       string
	}{
		{"missing args", a.ID, "", "blocks", false, "'source' and 'target' are required"},
		{"bad kind", a.ID, b.ID, "garbage", false, `invalid link type "garbage"`},
		{"unknown target", a.ID, "zzzzzz", "blocks", false, "target:"},
		{"self", a.ID, a.ID, "blocks", false, "cannot link an issue to itself"},
		{"identical", a.ID, b.ID, "blocks", false, "link blocks " + b.ID + " already exists on " + a.ID},
		{"identical with overwrite", a.ID, b.ID, "blocks", true, "already exists"},
		{"identical via inverse", b.ID, a.ID, "blocked_by", false, "link blocked_by " + a.ID + " already exists on " + b.ID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := c.Link(tc.source, tc.target, tc.kind, tc.overwrite)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}
