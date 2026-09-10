package exponential

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/sortorder"
	"github.com/palarix/exponential/internal/storage"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// normalizeLabels maps each label to its canonical casing from the project
// config or built-in defaults, and deduplicates the result.
func normalizeLabels(labels []string, cfg *config.Config) []string {
	if len(labels) == 0 {
		return labels
	}
	lookup := make(map[string]string)
	for k := range config.BuiltinLabels {
		lookup[strings.ToLower(k)] = k
	}
	if cfg != nil {
		for k := range cfg.Labels {
			lookup[strings.ToLower(k)] = k
		}
	}
	seen := make(map[string]bool)
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		canonical, ok := lookup[strings.ToLower(l)]
		if !ok {
			canonical = l
		}
		key := strings.ToLower(canonical)
		if !seen[key] {
			seen[key] = true
			out = append(out, canonical)
		}
	}
	return out
}

// AddIssue creates a new issue and persists it. If payload.Status is empty,
// the issue defaults to BACKLOG. Side-effect rules (auto-progress, blocked_by
// checks) are not run on create — they apply only to subsequent updates.
func (t *LocalTransport) AddIssue(payload model.CreatePayload) (*model.Issue, error) {
	alphabet := "0123456789abcdef"
	id, err := gonanoid.Generate(alphabet, 6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate issue ID: %w", err)
	}
	prefix := "issue"
	if t.Config.Prefix != "" {
		prefix = t.Config.Prefix
	}
	id = prefix + "-" + id

	payload.Labels = normalizeLabels(payload.Labels, t.Config)

	for i := range payload.Dependencies {
		if payload.Dependencies[i].SourceID == "" {
			payload.Dependencies[i].SourceID = id
		}
	}

	// Auto-assign sort_order if not provided — append at end of target status group
	if payload.SortOrder == "" {
		targetStatus := model.IssueStatus(payload.Status)
		if targetStatus == "" {
			targetStatus = model.StatusBacklog
		}
		if events, err := storage.ReadEvents(); err == nil {
			issues := ProjectIssues(events)
			var keys []string
			for _, iss := range issues {
				if iss.Status == targetStatus && iss.SortOrder != "" {
					keys = append(keys, iss.SortOrder)
				}
			}
			sort.Strings(keys)
			lastKey := ""
			if len(keys) > 0 {
				lastKey = keys[len(keys)-1]
			}
			if key, err := sortorder.GenerateKeyBetween(lastKey, ""); err == nil {
				payload.SortOrder = key
			}
		}
	}

	user := t.GetUser()

	event := model.Event{
		ID:        id,
		Type:      model.EventTypeCreate,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	if err := t.appendEvent(event); err != nil {
		return nil, fmt.Errorf("failed to append event: %w", err)
	}

	if t.Config.AutoCommit {
		hub := storage.HubRoot()
		commitMsg := fmt.Sprintf("xpo: create %s - %s", id, payload.Title)
		_ = exec.Command("git", "-C", hub, "add", ".xpo/issues.db").Run()
		_ = exec.Command("git", "-C", hub, "commit", "-m", commitMsg).Run()
	}

	status := model.IssueStatus(payload.Status)
	if status == "" {
		status = model.StatusBacklog
	}

	issue := &model.Issue{
		ID:           id,
		Status:       status,
		Title:        payload.Title,
		Description:  payload.Description,
		ParentID:     payload.ParentID,
		Estimate:     payload.Estimate,
		Priority:     payload.Priority,
		SortOrder:    payload.SortOrder,
		Assignee:     payload.Assignee,
		Dependencies: payload.Dependencies,
		Labels:       payload.Labels,
		CreatedAt:    event.CreatedAt,
		CreatedBy:    user,
	}
	return issue, nil
}
