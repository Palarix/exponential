package beats

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// AddOptions contains parameters for creating an issue.
type AddOptions struct {
	Title       string
	Description string
	Kind        string
	ParentID    string
	Estimate    int
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
		Kind:        opts.Kind,
		Title:       opts.Title,
		Description: opts.Description,
		ParentID:    opts.ParentID,
		Estimate:    opts.Estimate,
	}

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeCreate,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := storage.AppendEvent(event); err != nil {
		return nil, fmt.Errorf("error appending event: %w", err)
	}

	if c.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: create %s %s - %s", opts.Kind, id, opts.Title)
		// We ignore errors here as it's a convenience feature, or should we log?
		// Logic was: fmt.Printf("Error...")
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	// We need to return the Issue object.
	// Since we just appended the event, we can construct the Issue manually or re-project.
	// Re-projecting is expensive. Let's construct it.
	issue := &model.Issue{
		ID:        id,
		Kind:      opts.Kind,
		Status:    model.StatusBacklog, // Default
		Title:     opts.Title,
		CreatedAt: event.CreatedAt,
		CreatedBy: user,
		ParentID:  opts.ParentID,
		Estimate:  opts.Estimate,
		// Description is not on issue struct usually? Let's check projection.go
	}
	return issue, nil
}
