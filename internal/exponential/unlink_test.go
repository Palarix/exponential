package exponential

import (
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func localClient(tr *LocalTransport) *Client {
	return &Client{Config: tr.Config, Transport: tr, local: tr}
}

func TestUnlink_RemovesOwnedLinkKeepsOthers(t *testing.T) {
	tr := setupLocalTransport(t)
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})
	c, _ := tr.AddIssue(model.CreatePayload{Title: "C"})
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A", Dependencies: []model.Dependency{
		{TargetID: b.ID, Kind: model.DependencyDependsOn},
		{TargetID: c.ID, Kind: "relates_to"},
	}})

	dep, _, err := localClient(tr).Unlink(a.ID, b.ID, "depends-on")
	if err != nil {
		t.Fatal(err)
	}
	if dep.TargetID != b.ID || dep.Kind != model.DependencyDependsOn {
		t.Errorf("removed = %+v, want depends_on %s", dep, b.ID)
	}
	links := readAllIssues(t)[a.ID].Links
	if len(links) != 1 || links[0].TargetID != c.ID {
		t.Errorf("A links = %+v, want only relates_to %s", links, c.ID)
	}
}

func TestUnlink_RemovesLastLink(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	if _, _, err := localClient(tr).Unlink(a.ID, b.ID, "blocks"); err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	if len(issues[a.ID].Links) != 0 || len(issues[b.ID].Links) != 0 {
		t.Errorf("links remain: A=%+v B=%+v", issues[a.ID].Links, issues[b.ID].Links)
	}
}

func TestUnlink_RemovesLinkStoredOnTarget(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)

	if _, _, err := localClient(tr).Unlink(b.ID, a.ID, "blocked_by"); err != nil {
		t.Fatal(err)
	}
	if deps := readAllIssues(t)[a.ID].Dependencies; len(deps) != 0 {
		t.Errorf("A deps = %+v, want blocks row removed", deps)
	}
}

func TestUnlink_RemovesRowWithEmptySourceID(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "A"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "B"})
	// Pre-xpo-080fc4 shape: an update stored without source_id.
	if err := tr.appendEvent(model.Event{ID: a.ID, Type: model.EventTypeUpdate, Payload: model.UpdatePayload{
		Dependencies: []model.Dependency{{TargetID: b.ID, Kind: model.DependencyDependsOn}},
	}}); err != nil {
		t.Fatal(err)
	}

	if _, _, err := localClient(tr).Unlink(a.ID, b.ID, "depends_on"); err != nil {
		t.Fatal(err)
	}
	if links := readAllIssues(t)[a.ID].Links; len(links) != 0 {
		t.Errorf("A links = %+v, want none", links)
	}
}

func TestUnlink_TargetNoLongerResolves(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyDependsOn)
	if err := tr.DeleteIssue(b.ID, "gone", false); err != nil {
		t.Fatal(err)
	}

	// Unique suffix of the vanished target's ID.
	suffix := b.ID[len(b.ID)-4:]
	if _, _, err := localClient(tr).Unlink(a.ID, suffix, "depends_on"); err != nil {
		t.Fatal(err)
	}
	if links := readAllIssues(t)[a.ID].Links; len(links) != 0 {
		t.Errorf("A links = %+v, want none", links)
	}
}

func TestUnlink_Errors(t *testing.T) {
	tr := setupLocalTransport(t)
	a, b := addLinked(t, tr, model.DependencyBlocks)
	c := localClient(tr)

	cases := []struct {
		name, source, target, kind, want string
	}{
		{"missing args", a.ID, "", "blocks", "'source' and 'target' are required"},
		{"bad kind", a.ID, b.ID, "garbage", `invalid link type "garbage"`},
		{"no such link", a.ID, b.ID, "relates_to", "no link relates_to"},
		{"unknown target", a.ID, "zzzzzz", "blocks", "target"},
		{"self", a.ID, a.ID, "blocks", "no link blocks"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := c.Unlink(tc.source, tc.target, tc.kind)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}
