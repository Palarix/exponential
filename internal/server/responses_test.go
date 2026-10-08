package server

import (
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestIssueToResponse_IncludesDerivedLinks(t *testing.T) {
	resp := issueToResponse(&model.Issue{
		ID: "b",
		Links: []model.Dependency{
			{SourceID: "b", TargetID: "a", Kind: model.DependencyBlockedBy, Derived: true},
		},
	})
	want := DependencyResponse{SourceID: "b", TargetID: "a", Kind: "blocked_by", Derived: true}
	if len(resp.Dependencies) != 1 || resp.Dependencies[0] != want {
		t.Errorf("dependencies = %+v, want [%+v]", resp.Dependencies, want)
	}
}
