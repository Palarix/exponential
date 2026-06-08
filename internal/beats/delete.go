package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
)

// DeleteIssue deletes an issue by appending a delete event.
// When cascade is true, children are also deleted; otherwise their ParentID is cleared.
func (c *Client) DeleteIssue(id string, reason string, cascade ...bool) error {
	// 1. Verify existence (Active only)
	events, err := storage.ReadEvents()
	if err != nil {
		return fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	targetIssue, err := c.resolveIssue(issues, id)
	if err != nil {
		return err
	}
	id = targetIssue.ID // Use resolved ID

	user := c.GetUser()

	doCascade := len(cascade) > 0 && cascade[0]

	// 2. Create delete event
	payload := model.DeletePayload{
		Reason:  reason,
		Cascade: doCascade,
	}

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeDelete,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := c.appendEvent(event); err != nil {
		return fmt.Errorf("error appending event: %w", err)
	}

	// 3. Handle children: cascade-delete or unparent
	for _, issue := range issues {
		if issue.ParentID != id {
			continue
		}
		if doCascade {
			deleteEvent := model.Event{
				ID:        issue.ID,
				Type:      model.EventTypeDelete,
				Payload:   model.DeletePayload{Reason: fmt.Sprintf("cascade from %s", id)},
				CreatedAt: time.Now().UTC(),
				CreatedBy: user,
			}
			if err := c.appendEvent(deleteEvent); err != nil {
				return fmt.Errorf("error cascade-deleting child %s: %w", issue.ID, err)
			}
		} else {
			emptyParent := ""
			unparentEvent := model.Event{
				ID:        issue.ID,
				Type:      model.EventTypeUpdate,
				Payload:   model.UpdatePayload{ParentID: &emptyParent},
				CreatedAt: time.Now().UTC(),
				CreatedBy: user,
			}
			if err := c.appendEvent(unparentEvent); err != nil {
				return fmt.Errorf("error unparenting child %s: %w", issue.ID, err)
			}
		}
	}

	// 4. Autocommit
	if c.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: delete %s", id)
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return nil
}
