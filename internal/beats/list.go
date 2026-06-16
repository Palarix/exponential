package beats

import (
	"fmt"
	"strings"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/palarix/beats/internal/ui"
)

// FilterOptions contains parameters for filtering issues.
type FilterOptions struct {
	Statuses []string
	Since    string
	Before   string
	Match    string
	Mine     bool
	ParentID string
	Label    string
	Assignee string
	CycleID  string
	All      bool
	Archived bool
}

// ListIssues retrieves and filters issues based on options.
func (t *LocalTransport) ListIssues(opts FilterOptions) ([]*model.Issue, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	if opts.Archived {
		archivedEvents, err := storage.ReadArchivedEvents()
		if err != nil {
			return nil, fmt.Errorf("failed to read archived events: %w", err)
		}
		events = append(events, archivedEvents...)
	}

	issuesMap := ProjectIssuesWithConfig(events, t.Config)
	issues := SortIssues(issuesMap)

	return FilterIssues(issues, opts, t.GetUser()), nil
}

// FilterIssues applies filtering logic to a list of issues.
func FilterIssues(issues []*model.Issue, opts FilterOptions, currentUser string) []*model.Issue {
	var filtered []*model.Issue

	validStatuses := make(map[string]bool)
	for _, s := range opts.Statuses {
		validStatuses[s] = true
	}

	var sinceTime, beforeTime time.Time
	if opts.Since != "" {
		if t, err := ParseTimeFilter(opts.Since); err == nil {
			sinceTime = t
		}
	}
	if opts.Before != "" {
		if t, err := ParseTimeFilter(opts.Before); err == nil {
			beforeTime = t
		}
	}

	recentDoneIDs := make(map[string]bool)
	if !opts.All {
		recentDoneIDs = ui.GetRecentDoneIDs(issues, 3)
	}

	matchQuery := strings.ToLower(opts.Match)
	labelQuery := strings.ToLower(opts.Label)
	assigneeQuery := strings.ToLower(opts.Assignee)

	for _, i := range issues {
		if len(validStatuses) > 0 {
			if !validStatuses[string(i.Status)] {
				continue
			}
		}

		if !sinceTime.IsZero() && i.UpdatedAt.Before(sinceTime) {
			continue
		}
		if !beforeTime.IsZero() && i.UpdatedAt.After(beforeTime) {
			continue
		}

		if matchQuery != "" {
			matchFound := false
			fields := []string{
				i.ID,
				string(i.Status),
				i.Title,
				i.ParentID,
				i.CreatedBy,
				i.Assignee,
			}
			fields = append(fields, i.Labels...)
			for _, f := range fields {
				if strings.Contains(strings.ToLower(f), matchQuery) {
					matchFound = true
					break
				}
			}
			if !matchFound {
				continue
			}
		}

		if opts.Mine {
			if !strings.Contains(i.CreatedBy, currentUser) && !strings.Contains(currentUser, i.CreatedBy) {
				userEmail := ui.ExtractEmail(currentUser)
				issueEmail := ui.ExtractEmail(i.CreatedBy)
				if userEmail == "" || issueEmail == "" || userEmail != issueEmail {
					continue
				}
			}
		}

		if opts.ParentID != "" {
			if i.ParentID != opts.ParentID {
				continue
			}
		}

		if labelQuery != "" {
			labelFound := false
			for _, l := range i.Labels {
				if strings.Contains(strings.ToLower(l), labelQuery) {
					labelFound = true
					break
				}
			}
			if !labelFound {
				continue
			}
		}

		if assigneeQuery != "" {
			if !strings.Contains(strings.ToLower(i.Assignee), assigneeQuery) {
				continue
			}
		}

		if opts.CycleID != "" {
			if i.EffectiveCycleID != opts.CycleID {
				continue
			}
		}

		if i.Status == model.StatusDone {
			show := false
			if opts.All {
				show = true
			} else if validStatuses[string(model.StatusDone)] {
				show = true
			} else if recentDoneIDs[i.ID] {
				show = true
			}

			if !show {
				continue
			}
		}

		filtered = append(filtered, i)
	}

	return filtered
}

// ParseTimeFilter parses a time string (duration or date) into a time.Time.
func ParseTimeFilter(input string) (time.Time, error) {
	if len(input) > 1 {
		lastChar := input[len(input)-1]
		if lastChar == 'd' || lastChar == 'w' {
			valStr := input[:len(input)-1]
			modifiedInput := valStr + "h"
			if d, err := time.ParseDuration(modifiedInput); err == nil {
				var factor int64
				if lastChar == 'd' {
					factor = 24
				} else {
					factor = 24 * 7
				}
				finalDuration := d * time.Duration(factor)
				return time.Now().Add(-finalDuration), nil
			}
		}
	}

	if d, err := time.ParseDuration(input); err == nil {
		return time.Now().Add(-d), nil
	}
	if t, err := time.Parse("2006-01-02", input); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, input)
}
