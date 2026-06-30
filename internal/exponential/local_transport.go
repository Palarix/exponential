package exponential

import (
	"fmt"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// LocalTransport implements Transport by reading and writing the local
// .xpo/issues.db event log directly.
type LocalTransport struct {
	Config       *config.Config
	Collapse     bool
	UserOverride string
	OnBehalfOf   string
	Source       string
}

func (t *LocalTransport) appendEvent(evt model.Event) error {
	if t.Source != "" {
		evt.Source = t.Source
	}
	if t.OnBehalfOf != "" {
		evt.OnBehalfOf = t.OnBehalfOf
	}
	if t.Collapse {
		return storage.AppendEventCollapsed(evt)
	}
	return storage.AppendEvent(evt)
}

// GetIssue retrieves an issue by ID, resolving short IDs if necessary.
func (t *LocalTransport) GetIssue(id string) (*model.Issue, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	issues := ProjectIssues(events)

	return t.resolveIssue(issues, id)
}

// resolveIssue attempts to find an issue by ID using exact match, prefix match, or short hash match.
func (t *LocalTransport) resolveIssue(issues map[string]*model.Issue, id string) (*model.Issue, error) {
	if issue, exists := issues[id]; exists {
		return issue, nil
	}

	if t.Config.Prefix != "" && !strings.HasPrefix(id, t.Config.Prefix) {
		prefixedID := t.Config.Prefix + id
		if issue, exists := issues[prefixedID]; exists {
			return issue, nil
		}
	}

	var matches []*model.Issue
	for _, issue := range issues {
		if strings.Contains(issue.ID, id) {
			matches = append(matches, issue)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return nil, fmt.Errorf("issue ID '%s' is ambiguous (matches %d issues)", id, len(matches))
	}

	return nil, fmt.Errorf("issue %s not found", id)
}

// GetInbox scans the local event log for events relevant to the current
// user since the given cursor.
func (t *LocalTransport) GetInbox(since time.Time) ([]InboxItem, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	issues := ProjectIssues(events)
	return BuildInbox(events, issues, t.GetUser(), since), nil
}

// FindIssue retrieves an issue by ID, checking the active store first, then the archive.
func (t *LocalTransport) FindIssue(id string) (*model.Issue, []*model.Issue, bool, error) {
	findChildren := func(targetID string, allIssues map[string]*model.Issue) []*model.Issue {
		var children []*model.Issue
		for _, i := range allIssues {
			if i.ParentID == targetID {
				children = append(children, i)
			}
		}
		return children
	}

	events, err := storage.ReadEvents()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to read events: %w", err)
	}
	issues := ProjectIssues(events)

	if issue, err := t.resolveIssue(issues, id); err == nil {
		return issue, findChildren(issue.ID, issues), false, nil
	}

	archivedEvents, err := storage.ReadArchivedEvents()
	if err != nil {
		return nil, nil, false, fmt.Errorf("issue not found (and error reading archive: %w)", err)
	}
	archivedIssues := ProjectIssues(archivedEvents)

	if issue, err := t.resolveIssue(archivedIssues, id); err == nil {
		return issue, findChildren(issue.ID, archivedIssues), true, nil
	}

	return nil, nil, false, fmt.Errorf("issue %s not found", id)
}
