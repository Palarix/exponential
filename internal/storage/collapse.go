package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/palarix/exponential/internal/model"
)

func readCommittedBytes(path string) ([]byte, error) {
	out, err := exec.Command("git", "-C", HubRoot(), "show", "HEAD:.xpo/issues.db").Output()
	if err != nil {
		return nil, nil
	}
	return out, nil
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	s := strings.TrimRight(string(data), "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func AppendEventCollapsed(event model.Event) error {
	path := filepath.Join(XpoDir(), "issues.db")

	committedBytes, err := readCommittedBytes(path)
	if err != nil {
		return err
	}

	committedCount := countLines(committedBytes)

	events, err := ReadEvents()
	if err != nil {
		return err
	}

	uncommitted := events[committedCount:]

	merged := false
	for i := len(uncommitted) - 1; i >= 0; i-- {
		existing := uncommitted[i]
		if existing.ID != event.ID || existing.Type != event.Type {
			continue
		}

		switch event.Type {
		case model.EventTypeUpdate:
			merged = mergeUpdatePayloads(&uncommitted[i], event)
		case model.EventTypeComment:
			merged = mergeCommentPayloads(&uncommitted[i], event)
		}
		if merged {
			break
		}
	}

	if !merged {
		uncommitted = append(uncommitted, event)
	}

	committedState := projectCommittedState(events[:committedCount])
	uncommitted = pruneNoopUpdates(uncommitted, committedState)

	return rewriteFile(path, committedBytes, uncommitted)
}

func mergeUpdatePayloads(existing *model.Event, incoming model.Event) bool {
	oldBytes, _ := json.Marshal(existing.Payload)
	newBytes, _ := json.Marshal(incoming.Payload)

	var oldP, newP model.UpdatePayload
	json.Unmarshal(oldBytes, &oldP)
	json.Unmarshal(newBytes, &newP)

	if newP.Title != nil {
		oldP.Title = newP.Title
	}
	if newP.Description != nil {
		oldP.Description = newP.Description
	}
	if newP.Status != nil {
		oldP.Status = newP.Status
	}
	if newP.ParentID != nil {
		oldP.ParentID = newP.ParentID
	}
	if newP.Estimate != nil {
		oldP.Estimate = newP.Estimate
	}
	if newP.Priority != nil {
		oldP.Priority = newP.Priority
	}
	if newP.SortOrder != nil {
		oldP.SortOrder = newP.SortOrder
	}
	if newP.Assignee != nil {
		oldP.Assignee = newP.Assignee
	}
	if newP.CycleID != nil {
		oldP.CycleID = newP.CycleID
	}
	if newP.Labels != nil {
		oldP.Labels = newP.Labels
	}
	if newP.Dependencies != nil {
		oldP.Dependencies = newP.Dependencies
	}

	existing.Payload = oldP
	existing.CreatedAt = incoming.CreatedAt
	return true
}

func mergeCommentPayloads(existing *model.Event, incoming model.Event) bool {
	oldBytes, _ := json.Marshal(existing.Payload)
	newBytes, _ := json.Marshal(incoming.Payload)

	var oldP, newP model.CommentPayload
	json.Unmarshal(oldBytes, &oldP)
	json.Unmarshal(newBytes, &newP)

	if oldP.ID != newP.ID {
		return false
	}

	existing.Payload = newP
	existing.CreatedAt = incoming.CreatedAt
	return true
}

func projectCommittedState(events []model.Event) map[string]*model.Issue {
	issues := make(map[string]*model.Issue)
	for _, evt := range events {
		switch evt.Type {
		case model.EventTypeCreate:
			b, _ := json.Marshal(evt.Payload)
			var p model.CreatePayload
			json.Unmarshal(b, &p)
			status := model.IssueStatus(p.Status)
			if status == "" {
				status = model.StatusBacklog
			}
			issues[evt.ID] = &model.Issue{
				ID:           evt.ID,
				Title:        p.Title,
				Description:  p.Description,
				Status:       status,
				ParentID:     p.ParentID,
				Estimate:     p.Estimate,
				Priority:     p.Priority,
				SortOrder:    p.SortOrder,
				Assignee:     p.Assignee,
				CycleID:      p.CycleID,
				Labels:       p.Labels,
				Dependencies: p.Dependencies,
			}
		case model.EventTypeUpdate:
			issue, ok := issues[evt.ID]
			if !ok {
				continue
			}
			b, _ := json.Marshal(evt.Payload)
			var p model.UpdatePayload
			json.Unmarshal(b, &p)
			if p.Title != nil {
				issue.Title = *p.Title
			}
			if p.Description != nil {
				issue.Description = *p.Description
			}
			if p.Status != nil {
				issue.Status = model.IssueStatus(*p.Status)
			}
			if p.ParentID != nil {
				issue.ParentID = *p.ParentID
			}
			if p.Estimate != nil {
				issue.Estimate = *p.Estimate
			}
			if p.Priority != nil {
				issue.Priority = *p.Priority
			}
			if p.SortOrder != nil {
				issue.SortOrder = *p.SortOrder
			}
			if p.Assignee != nil {
				issue.Assignee = *p.Assignee
			}
			if p.CycleID != nil {
				issue.CycleID = *p.CycleID
			}
			if p.Labels != nil {
				issue.Labels = p.Labels
			}
			if p.Dependencies != nil {
				issue.Dependencies = p.Dependencies
			}
		case model.EventTypeDelete:
			if issue, ok := issues[evt.ID]; ok {
				issue.Deleted = true
			}
		}
	}
	return issues
}

func pruneNoopUpdates(uncommitted []model.Event, committedState map[string]*model.Issue) []model.Event {
	var result []model.Event
	for _, evt := range uncommitted {
		if evt.Type != model.EventTypeUpdate {
			result = append(result, evt)
			continue
		}

		issue, exists := committedState[evt.ID]
		if !exists {
			result = append(result, evt)
			continue
		}

		b, _ := json.Marshal(evt.Payload)
		var p model.UpdatePayload
		json.Unmarshal(b, &p)

		if p.Title != nil && *p.Title == issue.Title {
			p.Title = nil
		}
		if p.Description != nil && *p.Description == issue.Description {
			p.Description = nil
		}
		if p.Status != nil && *p.Status == string(issue.Status) {
			p.Status = nil
		}
		if p.ParentID != nil && *p.ParentID == issue.ParentID {
			p.ParentID = nil
		}
		if p.Estimate != nil && *p.Estimate == issue.Estimate {
			p.Estimate = nil
		}
		if p.Priority != nil && *p.Priority == issue.Priority {
			p.Priority = nil
		}
		if p.SortOrder != nil && *p.SortOrder == issue.SortOrder {
			p.SortOrder = nil
		}
		if p.Assignee != nil && *p.Assignee == issue.Assignee {
			p.Assignee = nil
		}
		if p.CycleID != nil && *p.CycleID == issue.CycleID {
			p.CycleID = nil
		}
		if p.Labels != nil && strSlicesEqual(p.Labels, issue.Labels) {
			p.Labels = nil
		}
		if p.Dependencies != nil && depsEqual(p.Dependencies, issue.Dependencies) {
			p.Dependencies = nil
		}

		if p.Title == nil && p.Description == nil && p.Status == nil &&
			p.ParentID == nil && p.Estimate == nil && p.Priority == nil &&
			p.SortOrder == nil && p.Assignee == nil && p.CycleID == nil &&
			p.Labels == nil && p.Dependencies == nil {
			continue
		}

		evt.Payload = p
		result = append(result, evt)
	}
	return result
}

func strSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func depsEqual(a, b []model.Dependency) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func rewriteFile(path string, committedRaw []byte, uncommitted []model.Event) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)

	if len(committedRaw) > 0 {
		w.Write(committedRaw)
	}

	for _, evt := range uncommitted {
		bytes, err := json.Marshal(evt)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		w.Write(bytes)
		w.WriteString("\n")
	}

	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()

	return os.Rename(tmp, path)
}
