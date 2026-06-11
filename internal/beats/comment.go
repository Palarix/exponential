package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/palarix/beats/internal/model"
)

// AddComment adds a comment to an issue.
func (t *LocalTransport) AddComment(issueID, text string) error {
	issue, err := t.GetIssue(issueID)
	if err != nil {
		return err
	}
	issueID = issue.ID

	commentID := fmt.Sprintf("cmt-%s", uuid.New().String())
	user := t.GetUser()

	event := model.Event{
		ID:   issueID,
		Type: model.EventTypeComment,
		Payload: model.CommentPayload{
			ID:   commentID,
			Text: text,
		},
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := t.appendEvent(event); err != nil {
		return fmt.Errorf("error appending event: %w", err)
	}

	if t.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: comment on %s", issueID)
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return nil
}
