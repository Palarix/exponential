package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/palarix/beats/internal/model"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// AddIssue creates a new issue and persists it. If payload.Status is empty,
// the issue defaults to BACKLOG. Side-effect rules (auto-progress, blocked_by
// checks) are not run on create — they apply only to subsequent updates.
func (c *Client) AddIssue(payload model.CreatePayload) (*model.Issue, error) {
	// Generate ID
	alphabet := "0123456789abcdef"
	id, err := gonanoid.Generate(alphabet, 6)
	if err != nil {
		return nil, fmt.Errorf("error generating ID: %w", err)
	}
	prefix := "beats-"
	if c.Config.Prefix != "" {
		prefix = c.Config.Prefix
	}
	id = prefix + id

	// Auto-fill SourceID on dependencies so callers don't need to know the
	// new issue's ID up front.
	for i := range payload.Dependencies {
		if payload.Dependencies[i].SourceID == "" {
			payload.Dependencies[i].SourceID = id
		}
	}

	user := c.GetUser()

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeCreate,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := c.appendEvent(event); err != nil {
		return nil, fmt.Errorf("error appending event: %w", err)
	}

	if c.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: create %s - %s", id, payload.Title)
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
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
