package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

// DeleteIssue deletes an issue by appending a delete event.
func (c *Client) DeleteIssue(id string, reason string) error {
	// 1. Verify existence (Active only)
	events, err := storage.ReadEvents()
	if err != nil {
		return fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	if _, exists := issues[id]; !exists {
		return fmt.Errorf("issue %s not found", id)
	}

	user := c.GetUser()

	// 2. Create delete event
	payload := model.DeletePayload{
		Reason: reason,
	}

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeDelete,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := storage.AppendEvent(event); err != nil {
		return fmt.Errorf("error appending event: %w", err)
	}

	// 3. Autocommit
	if c.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: delete %s", id)
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return nil
}
