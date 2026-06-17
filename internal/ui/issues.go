package ui

import (
	"sort"

	"github.com/palarix/exponential/internal/model"
)

// GetRecentDoneIDs returns a map of the most recently updated DONE issue IDs
func GetRecentDoneIDs(issues []*model.Issue, count int) map[string]bool {
	// Filter to only DONE issues
	var doneIssues []*model.Issue
	for _, i := range issues {
		if i.Status == model.StatusDone {
			doneIssues = append(doneIssues, i)
		}
	}

	// Sort by UpdatedAt descending
	sort.Slice(doneIssues, func(i, j int) bool {
		return doneIssues[i].UpdatedAt.After(doneIssues[j].UpdatedAt)
	})

	// Build result map with top N
	result := make(map[string]bool)
	for k := 0; k < count && k < len(doneIssues); k++ {
		result[doneIssues[k].ID] = true
	}

	return result
}

// GetRecentDoneIDsFromMap is a convenience wrapper for map-based issue collections
func GetRecentDoneIDsFromMap(issuesMap map[string]*model.Issue, count int) map[string]bool {
	issues := make([]*model.Issue, 0, len(issuesMap))
	for _, i := range issuesMap {
		issues = append(issues, i)
	}
	return GetRecentDoneIDs(issues, count)
}
