package beats

import (
	"fmt"
	"os/exec"

	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
)

// GitCommit stages and commits the issues.db file.
func GitCommit(msg string) {
	_ = exec.Command("git", "add", ".beats/issues.db").Run()
	_ = exec.Command("git", "commit", "-m", msg).Run()
}

// Client manages the interaction with the beats issue tracker.
type Client struct {
	Config *config.Config
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		Config: cfg,
	}
}

// GetIssue retrieves an issue by ID.
func (c *Client) GetIssue(id string) (*model.Issue, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	issue, exists := issues[id]
	if !exists {
		return nil, fmt.Errorf("issue %s not found", id)
	}
	return issue, nil
}

// FindIssue retrieves an issue by ID, checking the active store first, then the archive.
// Returns the issue, a list of its children, a boolean indicating if it is archived, and any error.
func (c *Client) FindIssue(id string) (*model.Issue, []*model.Issue, bool, error) {
	// Helper to find children
	findChildren := func(targetID string, allIssues map[string]*model.Issue) []*model.Issue {
		var children []*model.Issue
		for _, i := range allIssues {
			if i.ParentID == targetID {
				children = append(children, i)
			}
		}
		return children
	}

	// 1. Check Active
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, nil, false, fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	if issue, exists := issues[id]; exists {
		return issue, findChildren(id, issues), false, nil
	}

	// 2. Check Archive
	archivedEvents, err := storage.ReadArchivedEvents()
	if err != nil {
		return nil, nil, false, fmt.Errorf("issue not found (and error reading archive: %w)", err)
	}
	archivedIssues := ProjectIssues(archivedEvents)

	if issue, exists := archivedIssues[id]; exists {
		return issue, findChildren(id, archivedIssues), true, nil
	}

	return nil, nil, false, fmt.Errorf("issue %s not found", id)
}
