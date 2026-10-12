package exponential

import (
	"fmt"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// Issue drafts are unfinished new issues saved from the web UI. They live
// in .xpo/drafts/ on the hub, outside the event log, so only local mode
// supports them.
//
// The owner parameter scopes reads and writes to one user. An empty owner
// means no scoping (auth is off). A draft of another owner reports
// storage.ErrDraftNotFound, so callers cannot tell that it exists.

// ValidateIssueDraftPayload applies the create-payload limits to an issue
// draft. Unlike an issue, an issue draft may have an empty title.
func ValidateIssueDraftPayload(p model.IssueDraftPayload) error {
	if len(p.Title) > MaxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", MaxTitleLen)
	}
	if len(p.Description) > MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum size of %d bytes", MaxDescriptionLen)
	}
	if len(p.Assignee) > MaxAssigneeLen {
		return fmt.Errorf("assignee exceeds maximum length of %d characters", MaxAssigneeLen)
	}
	for _, l := range append(append([]string{}, p.Labels...), p.AdditionalLabels...) {
		if len(l) > MaxLabelLen {
			return fmt.Errorf("label %q exceeds maximum length of %d characters", l, MaxLabelLen)
		}
	}
	if p.Status != "" {
		switch model.IssueStatus(p.Status) {
		case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked,
			model.StatusDone, model.StatusCanceled, model.StatusDuplicate:
		default:
			return fmt.Errorf("invalid status %q", p.Status)
		}
	}
	if p.Estimate < 0 {
		return fmt.Errorf("estimate must not be negative")
	}
	return nil
}

// ListIssueDrafts returns the issue drafts of owner, newest first.
func (c *Client) ListIssueDrafts(owner string) ([]*model.IssueDraft, error) {
	if c.local == nil {
		return nil, ErrLocalOnly
	}
	all, err := storage.ListDrafts()
	if err != nil {
		return nil, err
	}
	if owner == "" {
		return all, nil
	}
	mine := []*model.IssueDraft{}
	for _, d := range all {
		if d.CreatedBy == owner {
			mine = append(mine, d)
		}
	}
	return mine, nil
}

// CreateIssueDraft stores a new issue draft authored by the current user.
func (c *Client) CreateIssueDraft(p model.IssueDraftPayload) (*model.IssueDraft, error) {
	if c.local == nil {
		return nil, ErrLocalOnly
	}
	if err := ValidateIssueDraftPayload(p); err != nil {
		return nil, err
	}
	id, err := storage.NewDraftID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	d := model.IssueDraft{ID: id, CreatedAt: now, UpdatedAt: now, CreatedBy: c.GetUser(), Payload: p}
	if err := storage.WriteDraft(d); err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateIssueDraft replaces the payload of an existing issue draft. It never
// recreates a deleted draft.
func (c *Client) UpdateIssueDraft(id, owner string, p model.IssueDraftPayload) (*model.IssueDraft, error) {
	if c.local == nil {
		return nil, ErrLocalOnly
	}
	if err := ValidateIssueDraftPayload(p); err != nil {
		return nil, err
	}
	d, err := c.ownedDraft(id, owner)
	if err != nil {
		return nil, err
	}
	d.Payload = p
	d.UpdatedAt = time.Now().UTC()
	if err := storage.WriteDraft(*d); err != nil {
		return nil, err
	}
	return d, nil
}

// DeleteIssueDraft removes an issue draft.
func (c *Client) DeleteIssueDraft(id, owner string) error {
	if c.local == nil {
		return ErrLocalOnly
	}
	if _, err := c.ownedDraft(id, owner); err != nil {
		return err
	}
	return storage.DeleteDraft(id)
}

func (c *Client) ownedDraft(id, owner string) (*model.IssueDraft, error) {
	d, err := storage.ReadDraft(id)
	if err != nil {
		return nil, err
	}
	if owner != "" && d.CreatedBy != owner {
		return nil, storage.ErrDraftNotFound
	}
	return d, nil
}
