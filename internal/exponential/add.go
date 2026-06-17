package exponential

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/palarix/exponential/internal/model"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// AddIssue creates a new issue and persists it. If payload.Status is empty,
// the issue defaults to BACKLOG. Side-effect rules (auto-progress, blocked_by
// checks) are not run on create — they apply only to subsequent updates.
func (t *LocalTransport) AddIssue(payload model.CreatePayload) (*model.Issue, error) {
	alphabet := "0123456789abcdef"
	id, err := gonanoid.Generate(alphabet, 6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate issue ID: %w", err)
	}
	prefix := "issue-"
	if t.Config.Prefix != "" {
		prefix = t.Config.Prefix
	}
	id = prefix + id

	for i := range payload.Dependencies {
		if payload.Dependencies[i].SourceID == "" {
			payload.Dependencies[i].SourceID = id
		}
	}

	user := t.GetUser()

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeCreate,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := t.appendEvent(event); err != nil {
		return nil, fmt.Errorf("failed to append event: %w", err)
	}

	if t.Config.AutoCommit {
		commitMsg := fmt.Sprintf("xpo: create %s - %s", id, payload.Title)
		_ = exec.Command("git", "add", ".xpo/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	status := model.IssueStatus(payload.Status)
	if status == "" {
		status = model.StatusBacklog
	}

	issue := &model.Issue{
		ID:           id,
		Status:       status,
		Title:        payload.Title,
		Description:  payload.Description,
		ParentID:     payload.ParentID,
		Estimate:     payload.Estimate,
		Priority:     payload.Priority,
		SortOrder:    payload.SortOrder,
		Assignee:     payload.Assignee,
		Dependencies: payload.Dependencies,
		Labels:       payload.Labels,
		CreatedAt:    event.CreatedAt,
		CreatedBy:    user,
	}
	return issue, nil
}
