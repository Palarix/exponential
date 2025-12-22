package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

// LogWork logs work (burned points) on an issue.
func (c *Client) LogWork(id string, amount int) error {
	// 1. Verify existence
	events, err := storage.ReadEvents()
	if err != nil {
		return fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	if _, exists := issues[id]; !exists {
		return fmt.Errorf("issue %s not found", id)
	}

	user := c.GetUser()

	// 2. Create worklog event
	payload := model.WorkLogPayload{
		Amount: amount,
	}

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeWorkLog,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := storage.AppendEvent(event); err != nil {
		return fmt.Errorf("error appending event: %w", err)
	}

	// 3. Autocommit
	if c.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: log %s %d", id, amount)
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return nil
}
