package beats

import (
	"fmt"
	"sort"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

type ArchiveStats struct {
	ActiveEvents   []model.Event
	ArchivedEvents []model.Event
	DoneCount      int
	DeletedCount   int
	KeptCount      int
	TotalArchive   int
}

// GetArchiveStats calculates which issues would be archived.
func (c *Client) GetArchiveStats(days int, keep int) (*ArchiveStats, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}

	issues := ProjectIssues(events)

	var deletedIDs []string
	var doneIssues []*model.Issue

	for _, i := range issues {
		if i.Deleted {
			deletedIDs = append(deletedIDs, i.ID)
		} else if i.Status == model.StatusDone {
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

	// Filter events
	var activeEvents []model.Event
	var archivedEvents []model.Event

	for _, evt := range events {
		if idsToArchive[evt.ID] {
			archivedEvents = append(archivedEvents, evt)
		} else {
			activeEvents = append(activeEvents, evt)
		}
	}

	return &ArchiveStats{
		ActiveEvents:   activeEvents,
		ArchivedEvents: archivedEvents,
		DoneCount:      len(doneIDsToArchive),
		DeletedCount:   len(deletedIDs),
		KeptCount:      len(doneIssues) - len(doneIDsToArchive),
		TotalArchive:   len(idsToArchive),
	}, nil
}

// PerformArchive executes the archiving operation.
func (c *Client) PerformArchive(stats *ArchiveStats) error {
	if stats == nil {
		return fmt.Errorf("archive stats cannot be nil")
	}
	return storage.ArchiveEvents(stats.ActiveEvents, stats.ArchivedEvents)
}
