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

func TestIssueToResponse_CommentOnBehalfOf(t *testing.T) {
	resp := issueToResponse(&model.Issue{
		ID: "a",
		Comments: []model.Comment{
			{ID: "c1", Text: "hi", CreatedBy: "claude-code/2.1.263 <agent@mcp>", OnBehalfOf: "Nicolas <nic@x.com>"},
		},
	})
	if len(resp.Comments) != 1 || resp.Comments[0].OnBehalfOf != "Nicolas <nic@x.com>" {
		t.Errorf("comments = %+v, want on_behalf_of carried", resp.Comments)
	}
}
