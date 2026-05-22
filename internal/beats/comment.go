package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/palarix/beats/internal/model"
)

// AddComment adds a comment to an issue.
func (c *Client) AddComment(issueID, text string) error {
	// Validate Issue Exists
	issue, err := c.GetIssue(issueID)
	if err != nil {
		return err
	}
	issueID = issue.ID

	// Generate Comment ID (unique)
	commentID := fmt.Sprintf("cmt-%s", uuid.New().String())

	// Get User
	user := c.GetUser()

	event := model.Event{
		ID:   issueID, // Event ID must be the Issue ID for projection to work
		Type: model.EventTypeComment,
		Payload: model.CommentPayload{
			ID:   commentID,
			Text: text,
		},
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	// Append Event
	if err := c.appendEvent(event); err != nil {
		return fmt.Errorf("error appending event: %w", err)
	}

	// Auto-commit
	if c.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: comment on %s", issueID)
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return nil
}
