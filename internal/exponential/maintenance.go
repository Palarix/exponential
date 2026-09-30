package exponential

import (
	"fmt"
	"sort"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

type ArchiveStats struct {
	IssueIDs     map[string]bool
	DoneCount    int
	DeletedCount int
	KeptCount    int
	TotalArchive int
}

// GetArchiveStats calculates which issues would be archived.
func (c *Client) GetArchiveStats(days int, keep int) (*ArchiveStats, error) {
	if c.local == nil {
		return nil, ErrLocalOnly
	}
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	issues := ProjectIssues(events)

	var deletedIDs []string
	var doneIssues []*model.Issue

	for _, i := range issues {
		if i.Deleted {
			deletedIDs = append(deletedIDs, i.ID)
		} else if model.IsTerminal(i.Status) {
			doneIssues = append(doneIssues, i)
		}
	}

	// Sort DONE issues by UpdatedAt (descending - newest first)
	sort.Slice(doneIssues, func(i, j int) bool {
		return doneIssues[i].UpdatedAt.After(doneIssues[j].UpdatedAt)
	})

	var doneIDsToArchive []string
	cutoff := time.Now().AddDate(0, 0, -days)

	for idx, issue := range doneIssues {
		if idx < keep {
			continue
		}
		if issue.UpdatedAt.Before(cutoff) {
			doneIDsToArchive = append(doneIDsToArchive, issue.ID)
		}
	}

	idsToArchive := make(map[string]bool)
	for _, id := range deletedIDs {
		idsToArchive[id] = true
	}
	for _, id := range doneIDsToArchive {
		idsToArchive[id] = true
	}

	return &ArchiveStats{
		IssueIDs:     idsToArchive,
		DoneCount:    len(doneIDsToArchive),
		DeletedCount: len(deletedIDs),
		KeptCount:    len(doneIssues) - len(doneIDsToArchive),
		TotalArchive: len(idsToArchive),
	}, nil
}

// PerformArchive archives the issues chosen by GetArchiveStats. Issues that
// changed since then and are no longer eligible are skipped and reported
// in the result.
func (c *Client) PerformArchive(stats *ArchiveStats) (storage.ArchiveResult, error) {
	if c.local == nil {
		return storage.ArchiveResult{}, ErrLocalOnly
	}
	if stats == nil {
		return storage.ArchiveResult{}, fmt.Errorf("archive stats cannot be nil")
	}
	return storage.ArchiveEvents(stats.IssueIDs)
}
