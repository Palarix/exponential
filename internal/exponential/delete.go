package exponential

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// DeleteIssue deletes an issue by appending a delete event.
// When cascade is true, children are also deleted; otherwise their ParentID is cleared.
func (t *LocalTransport) DeleteIssue(id string, reason string, cascade bool) error {
	events, err := storage.ReadEvents()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}
	issues := ProjectIssues(events)

	targetIssue, err := t.resolveIssue(issues, id)
	if err != nil {
		return err
	}
	id = targetIssue.ID

	user := t.GetUser()

	payload := model.DeletePayload{
		Reason:  reason,
		Cascade: cascade,
	}

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeDelete,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := t.appendEvent(event); err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}

	for _, issue := range issues {
		if issue.ParentID != id {
			continue
		}
		if cascade {
			deleteEvent := model.Event{
				ID:        issue.ID,
				Type:      model.EventTypeDelete,
				Payload:   model.DeletePayload{Reason: fmt.Sprintf("cascade from %s", id)},
				CreatedAt: time.Now().UTC(),
				CreatedBy: user,
			}
			if err := t.appendEvent(deleteEvent); err != nil {
				return fmt.Errorf("failed to cascade-delete child %s: %w", issue.ID, err)
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
			if err := t.appendEvent(unparentEvent); err != nil {
				return fmt.Errorf("failed to unparent child %s: %w", issue.ID, err)
			}
		}
	}

	if t.Config.AutoCommit {
		hub := storage.HubRoot()
		commitMsg := fmt.Sprintf("xpo: delete %s", id)
		_ = exec.Command("git", "-C", hub, "add", ".xpo/issues.db").Run()
		_ = exec.Command("git", "-C", hub, "commit", "-m", commitMsg).Run()
	}

	return nil
}
