package exponential

import (
	"fmt"
	"sort"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/sortorder"
	"github.com/palarix/exponential/internal/storage"
	"github.com/palarix/exponential/internal/version"
)

// RunMigrations runs all pending migrations up to the current DataModelVersion.
func RunMigrations(currentVersion int) (int, error) {
	if currentVersion >= version.DataModelVersion {
		return currentVersion, nil
	}
	if currentVersion < 2 {
		return 0, fmt.Errorf("data model v%d is not compatible with v%d; please run `xpo init` to create a fresh database", currentVersion, version.DataModelVersion)
	}

	if currentVersion == 2 {
		if err := migrateV2ToV3(); err != nil {
			return currentVersion, fmt.Errorf("v2→v3 migration failed (your data has not been modified): %w", err)
		}
		if err := UpdateConfigVersion(3); err != nil {
			return currentVersion, fmt.Errorf("migration succeeded but failed to update config version — re-run to retry: %w", err)
		}
		currentVersion = 3
	}

	return currentVersion, nil
}

// migrateV2ToV3 re-keys all issues with globally consistent sort_order values.
// Walks issues in visual order (status group by status group, preserving
// within-group sort_order), then assigns fresh keys in sequence.
func migrateV2ToV3() error {
	events, err := storage.ReadEvents()
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}
	issues := ProjectIssues(events)

	ordered := buildGlobalOrder(issues)
	if len(ordered) == 0 {
		return nil
	}

	keys, err := sortorder.GenerateNKeysBetween("", "", len(ordered))
	if err != nil {
		return fmt.Errorf("generating keys: %w", err)
	}

	timestamp := time.Now().UTC()
	for i, issue := range ordered {
		if issue.SortOrder == keys[i] {
			continue
		}
		newKey := keys[i]
		evt := model.Event{
			ID:        issue.ID,
			Type:      model.EventTypeUpdate,
			Payload:   model.UpdatePayload{SortOrder: &newKey},
			CreatedAt: timestamp,
			CreatedBy: "xpo migrate <system>",
		}
		if err := storage.AppendEvent(evt); err != nil {
			return fmt.Errorf("writing sort_order for %s: %w", issue.ID, err)
		}
	}

	return nil
}

var statusOrder = []model.IssueStatus{
	model.StatusBacklog,
	model.StatusPlanned,
	model.StatusDoing,
	model.StatusBlocked,
	model.StatusDone,
}

// buildGlobalOrder returns all issues in visual order: status group by
// status group, sorted by sort_order within each group.
func buildGlobalOrder(issues map[string]*model.Issue) []*model.Issue {
	byStatus := make(map[model.IssueStatus][]*model.Issue)
	for _, issue := range issues {
		byStatus[issue.Status] = append(byStatus[issue.Status], issue)
	}

	for _, group := range byStatus {
		sort.Slice(group, func(i, j int) bool {
			ki, kj := group[i].SortOrder, group[j].SortOrder
			if ki != kj {
				return ki < kj
			}
			if !group[i].CreatedAt.Equal(group[j].CreatedAt) {
				return group[i].CreatedAt.Before(group[j].CreatedAt)
			}
			return group[i].ID < group[j].ID
		})
	}

	var result []*model.Issue
	for _, status := range statusOrder {
		result = append(result, byStatus[status]...)
	}
	return result
}
