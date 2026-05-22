package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/palarix/beats/internal/model"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// AddOptions contains parameters for creating an issue.
type AddOptions struct {
	Title        string
	Description  string
	ParentID     string
	Estimate     int
	Assignee     string
	Dependencies []model.Dependency
	Labels       []string
}

// AddIssue creates a new issue and persists it.
func (c *Client) AddIssue(opts AddOptions) (*model.Issue, error) {
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

	// Get User
	user := c.GetUser()

	payload := model.CreatePayload{
		Title:        opts.Title,
		Description:  opts.Description,
		ParentID:     opts.ParentID,
		Estimate:     opts.Estimate,
		Assignee:     opts.Assignee,
		Dependencies: opts.Dependencies,
		Labels:       opts.Labels,
	}

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
		commitMsg := fmt.Sprintf("beats: create %s - %s", id, opts.Title)
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	issue := &model.Issue{
		ID:           id,
		Status:       model.StatusBacklog,
		Title:        opts.Title,
		Description:  opts.Description,
		ParentID:     opts.ParentID,
		Estimate:     opts.Estimate,
		Assignee:     opts.Assignee,
		Dependencies: opts.Dependencies,
		Labels:       opts.Labels,
		CreatedAt:    event.CreatedAt,
		CreatedBy:    user,
	}
	return issue, nil
}
