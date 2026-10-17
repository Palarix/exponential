package exponential

import (
	"errors"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func TestCreateIssueDraft_SetsMetadata(t *testing.T) {
	c := setupClient(t)
	d, err := c.CreateIssueDraft(model.IssueDraftPayload{Title: "half done", Labels: []string{"feature"}})
	if err != nil {
		t.Fatal(err)
	}
	if !storage.ValidDraftID(d.ID) {
		t.Errorf("bad id %q", d.ID)
	}
	if d.CreatedBy != "Tester <test@example.com>" {
		t.Errorf("created_by = %q", d.CreatedBy)
	}
	if d.CreatedAt.IsZero() || !d.CreatedAt.Equal(d.UpdatedAt) {
		t.Errorf("timestamps not set: %v / %v", d.CreatedAt, d.UpdatedAt)
	}
	got, err := storage.ReadDraft(d.ID)
	if err != nil || got.Payload.Title != "half done" {
		t.Errorf("draft not persisted: %v %+v", err, got)
	}
}

func TestCreateIssueDraft_AllowsEmptyTitle(t *testing.T) {
	c := setupClient(t)
	if _, err := c.CreateIssueDraft(model.IssueDraftPayload{Description: "only a body"}); err != nil {
		t.Errorf("expected empty title to be allowed, got %v", err)
	}
}

func TestCreateIssueDraft_Validates(t *testing.T) {
	c := setupClient(t)
	cases := map[string]model.IssueDraftPayload{
		"title":    {Title: strings.Repeat("x", MaxTitleLen+1)},
		"desc":     {Description: strings.Repeat("x", MaxDescriptionLen+1)},
		"assignee": {Assignee: strings.Repeat("x", MaxAssigneeLen+1)},
		"label":    {Labels: []string{strings.Repeat("x", MaxLabelLen+1)}},
		"addlabel": {AdditionalLabels: []string{strings.Repeat("x", MaxLabelLen+1)}},
		"status":   {Status: "NOPE"},
		"estimate": {Estimate: -1},
	}
	for name, p := range cases {
		if _, err := c.CreateIssueDraft(p); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}
}

func TestUpdateIssueDraft_KeepsIDAndCreatedAt(t *testing.T) {
	c := setupClient(t)
	d, _ := c.CreateIssueDraft(model.IssueDraftPayload{Title: "v1"})
	u, err := c.UpdateIssueDraft(d.ID, model.IssueDraftPayload{Title: "v2"})
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != d.ID || !u.CreatedAt.Equal(d.CreatedAt) || u.CreatedBy != d.CreatedBy {
		t.Errorf("identity changed: %+v vs %+v", u, d)
	}
	if u.UpdatedAt.Before(d.UpdatedAt) || u.Payload.Title != "v2" {
		t.Errorf("update not applied: %+v", u)
	}
	all, _ := c.ListIssueDrafts()
	if len(all) != 1 {
		t.Errorf("expected 1 draft after update, got %d", len(all))
	}
}

func TestUpdateIssueDraft_MissingDoesNotRecreate(t *testing.T) {
	c := setupClient(t)
	_, err := c.UpdateIssueDraft("draft-deadbeef", model.IssueDraftPayload{Title: "x"})
	if !errors.Is(err, storage.ErrDraftNotFound) {
		t.Errorf("expected ErrDraftNotFound, got %v", err)
	}
	all, _ := c.ListIssueDrafts()
	if len(all) != 0 {
		t.Errorf("update recreated a draft")
	}
}
