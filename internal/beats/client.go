package beats

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

// GitCommit stages and commits the issues.db file.
func GitCommit(msg string) {
	_ = exec.Command("git", "add", ".beats/issues.db").Run()
	_ = exec.Command("git", "commit", "-m", msg).Run()
}

// Client manages the interaction with the beats issue tracker.
type Client struct {
	Config   *config.Config
	Collapse bool
	// UserOverride, when non-empty, is returned by GetUser instead of the
	// configured user. The MCP server uses this to record the calling
	// agent's identity on writes.
	UserOverride string
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		Config: cfg,
	}
}

func (c *Client) appendEvent(evt model.Event) error {
	if c.Collapse {
		return storage.AppendEventCollapsed(evt)
	}
	return storage.AppendEvent(evt)
}

// GetIssue retrieves an issue by ID, resolving short IDs if necessary.
func (c *Client) GetIssue(id string) (*model.Issue, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	return c.resolveIssue(issues, id)
}

// resolveIssue attempts to find an issue by ID using exact match, prefix match, or short hash match.
func (c *Client) resolveIssue(issues map[string]*model.Issue, id string) (*model.Issue, error) {
	// 1. Exact Match
	if issue, exists := issues[id]; exists {
		return issue, nil
	}

	// 2. Prefix Match (if configured)
	if c.Config.Prefix != "" && !strings.HasPrefix(id, c.Config.Prefix) {
		prefixedID := c.Config.Prefix + id
		if issue, exists := issues[prefixedID]; exists {
			return issue, nil
		}
	}

	// 3. Short Hash Match (suffix)
	// If the ID is a short hash (e.g. from git or nanoid), try to match likely candidates
	var matches []*model.Issue
	for _, issue := range issues {
		// Check if the issue ID ends with the provided short ID
		// or if the provided ID is a substring of the Issue ID (safer to check suffix for nanoid?)
		// Nanoids are random, so suffix/prefix doesn't strictly matter like Git SHAs,
		// but users might type the last few chars.
		// Let's assume users might type the *unique* part.
		// Since we use `prefix-nanoid`, checking if `issue.ID` contains `id` is a good start.

		if strings.Contains(issue.ID, id) {
			matches = append(matches, issue)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		// Ambiguous
		return nil, fmt.Errorf("issue ID '%s' is ambiguous (matches %d issues)", id, len(matches))
	}

	return nil, fmt.Errorf("issue %s not found", id)
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

	if issue, err := c.resolveIssue(issues, id); err == nil {
		return issue, findChildren(issue.ID, issues), false, nil
	}

	// 2. Check Archive
	archivedEvents, err := storage.ReadArchivedEvents()
	if err != nil {
		return nil, nil, false, fmt.Errorf("issue not found (and error reading archive: %w)", err)
	}
	archivedIssues := ProjectIssues(archivedEvents)

	if issue, err := c.resolveIssue(archivedIssues, id); err == nil {
		return issue, findChildren(issue.ID, archivedIssues), true, nil
	}

	return nil, nil, false, fmt.Errorf("issue %s not found", id)
}
