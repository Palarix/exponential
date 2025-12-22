package beats

import (
	"sort"
	"strings"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

// CheckDuplicates searches for issues with similar titles.
func (c *Client) CheckDuplicates(title string) ([]*model.Issue, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, err
	}

	issuesMap := ProjectIssues(events)
	var issues []*model.Issue
	for _, i := range issuesMap {
		issues = append(issues, i)
	}
	// Sort by creation time desc (newest first)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].CreatedAt.After(issues[j].CreatedAt)
	})

	titleTokens := tokenize(title)
	var duplicates []*model.Issue

	for _, issue := range issues {
		issueTokens := tokenize(issue.Title)

		intersection := 0
		for t := range titleTokens {
			if issueTokens[t] {
				intersection++
			}
		}

		// Criteria:
		// 1. Strict Subset: intersection == len(titleTokens) (all new words exist in old)
		// 2. High Overlap: intersection >= 75% of min length

		minLen := len(titleTokens)
		if len(issueTokens) < minLen {
			minLen = len(issueTokens)
		}

		if minLen > 0 {
			ratio := float64(intersection) / float64(minLen)
			if ratio >= 0.75 {
				duplicates = append(duplicates, issue)
			}
		}
	}

	return duplicates, nil
}

func tokenize(s string) map[string]bool {
	tokens := make(map[string]bool)
	fields := strings.Fields(strings.ToLower(s))
	for _, f := range fields {
		f = strings.Trim(f, "(),.:;!?")
		if len(f) > 0 {
			tokens[f] = true
		}
	}
	return tokens
}
