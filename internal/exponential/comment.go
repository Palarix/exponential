package exponential

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/palarix/exponential/internal/model"
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
		return fmt.Errorf("failed to append event: %w", err)
	}

	if t.Config.AutoCommit {
		commitMsg := fmt.Sprintf("xpo: comment on %s", issueID)
		_ = exec.Command("git", "add", ".xpo/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return nil
}
