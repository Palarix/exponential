package exponential

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
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
		hub := storage.HubRoot()
		commitMsg := fmt.Sprintf("xpo: comment on %s", issueID)
		_ = exec.Command("git", "-C", hub, "add", ".xpo/issues.db").Run()
		_ = exec.Command("git", "-C", hub, "commit", "-m", commitMsg).Run()
	}

	return nil
}
