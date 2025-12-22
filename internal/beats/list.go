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
	EpicID   string
	All      bool
	Archived bool
}

// ListIssues retrieves and filters issues based on options.
func (c *Client) ListIssues(opts FilterOptions) ([]*model.Issue, error) {
	// 1. Read Events
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}

	if opts.Archived {
		archivedEvents, err := storage.ReadArchivedEvents()
		if err != nil {
			return nil, fmt.Errorf("error reading archived events: %w", err)
		}
		events = append(events, archivedEvents...)
	}

	// 2. Project
	issuesMap := ProjectIssues(events)
	issues := SortIssues(issuesMap)

	// 3. Filter
	return c.FilterIssues(issues, opts), nil
}

// FilterIssues applies filtering logic to a list of issues.
func (c *Client) FilterIssues(issues []*model.Issue, opts FilterOptions) []*model.Issue {
	var filtered []*model.Issue

	// Parse Statuses
	validStatuses := make(map[string]bool)
	for _, s := range opts.Statuses {
		validStatuses[s] = true
	}

	// Parse Time
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

	// Get Recent Done (needed for default visibility logic)
	recentDoneIDs := make(map[string]bool)
	if !opts.All {
		recentDoneIDs = ui.GetRecentDoneIDs(issues, 3)
	}

	matchQuery := strings.ToLower(opts.Match)
	currentUser := c.GetUser() // Needed for Mine filter

	for _, i := range issues {
		// Status Filter
		if len(validStatuses) > 0 {
			if !validStatuses[string(i.Status)] {
				continue
			}
		}

		// Time Filter
		if !sinceTime.IsZero() && i.UpdatedAt.Before(sinceTime) {
			continue
		}
		if !beforeTime.IsZero() && i.UpdatedAt.After(beforeTime) {
			continue
		}

		// Match Filter
		if matchQuery != "" {
			matchFound := false
			fields := []string{
				i.ID,
				i.Kind,
				string(i.Status),
				i.Title,
				i.ParentID,
				i.CreatedBy,
			}
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

		// Mine Filter
		if opts.Mine {
			if !strings.Contains(i.CreatedBy, currentUser) && !strings.Contains(currentUser, i.CreatedBy) {
				// Fallback: Check emails
				userEmail := ui.ExtractEmail(currentUser)
				issueEmail := ui.ExtractEmail(i.CreatedBy)
				if userEmail == "" || issueEmail == "" || userEmail != issueEmail {
					continue
				}
			}
		}

		// Epic Filter
		if opts.EpicID != "" {
			if i.ParentID != opts.EpicID {
				continue
			}
		}

		// Hide DONE unless explicit or recent
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
	// Handle custom suffixes for days (d) and weeks (w)
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

	// Try parsing as standard duration (relative to now)
	if d, err := time.ParseDuration(input); err == nil {
		return time.Now().Add(-d), nil
	}
	// Try parsing as date
	if t, err := time.Parse("2006-01-02", input); err == nil {
		return t, nil
	}
	// Try RFC3339
	return time.Parse(time.RFC3339, input)
}
