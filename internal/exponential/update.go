package exponential

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/sortorder"
	"github.com/palarix/exponential/internal/storage"
)

// applyUpdate builds and appends update events (including cascading
// automations) without creating a git commit.  Callers that need to
// bundle the update into a larger commit (e.g. MergeIssue) use this
// directly; standalone updates go through UpdateIssue which adds the
// git commit step.
func (t *LocalTransport) applyUpdate(id string, payload model.UpdatePayload) ([]string, error) {
	// 1. Read and Project State
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	issues := ProjectIssues(events)

	targetIssue, err := t.resolveIssue(issues, id)
	if err != nil {
		return nil, err
	}
	id = targetIssue.ID // Use the resolved ID

	user := t.GetUser()
	timestamp := time.Now().UTC()
	var eventsToAppend []model.Event
	var messages []string

	// Guard: reject self-referencing parent
	if payload.ParentID != nil && *payload.ParentID == id {
		return nil, fmt.Errorf("cannot set issue %s as its own parent", id)
	}

	// 2. Prepare Primary Update
	primaryEvent := model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   payload,
		CreatedAt: timestamp,
		CreatedBy: user,
	}
	eventsToAppend = append(eventsToAppend, primaryEvent)
	messages = append(messages, fmt.Sprintf("Updated %s", id))

	// Auto-assign sort_order when status changes and no explicit sort_order is set
	if payload.Status != nil && payload.SortOrder == nil {
		newStatus := model.IssueStatus(*payload.Status)
		if newStatus != targetIssue.Status {
			var keys []string
			for _, iss := range issues {
				if iss.Status == newStatus && iss.SortOrder != "" {
					keys = append(keys, iss.SortOrder)
				}
			}
			sort.Strings(keys)
			lastKey := ""
			if len(keys) > 0 {
				lastKey = keys[len(keys)-1]
			}
			if key, err := sortorder.GenerateKeyBetween(lastKey, ""); err == nil {
				payload.SortOrder = &key
				primaryEvent.Payload = payload
			}
		}
	}

	// 3. Logic & Side Effects based on Status Change
	if payload.Status != nil {
		newStatus := model.IssueStatus(*payload.Status)

		// --- Check blocked_by dependencies before starting ---
		if newStatus == model.StatusDoing {
			for _, dep := range targetIssue.Dependencies {
				if dep.Kind == model.DependencyBlockedBy {
					blocker, bExists := issues[dep.TargetID]
					if bExists && blocker.Status != model.StatusDone && !blocker.Deleted {
						return nil, fmt.Errorf("cannot start issue %s: blocked by incomplete issue %s", id, blocker.ID)
					}
				}
			}
		}

		// --- First-start trigger: parent auto-starts when any child starts ---
		if t.Config.Automations.FirstStart && targetIssue.ParentID != "" {
			if newStatus == model.StatusDoing || newStatus == model.StatusBlocked {
				parent, pExists := issues[targetIssue.ParentID]
				if pExists && !parent.Deleted &&
					(parent.Status == model.StatusBacklog || parent.Status == model.StatusPlanned) {
					parentStatus := string(model.StatusDoing)
					parentEvent := model.Event{
						ID:        parent.ID,
						Type:      model.EventTypeUpdate,
						Payload:   model.UpdatePayload{Status: &parentStatus},
						CreatedAt: timestamp,
						CreatedBy: user,
					}
					eventsToAppend = append(eventsToAppend, parentEvent)
					messages = append(messages, fmt.Sprintf("Auto-started parent %s", parent.ID))
				}
			}
		}

		// --- Last-completed trigger: parent auto-closes when last child is done ---
		if t.Config.Automations.LastCompleted && targetIssue.ParentID != "" && newStatus == model.StatusDone {
			parent, pExists := issues[targetIssue.ParentID]
			if pExists && !parent.Deleted && parent.Status != model.StatusDone {
				allDone := true
				for _, sibling := range issues {
					if sibling.ParentID != parent.ID || sibling.Deleted || sibling.ID == id {
						continue
					}
					if sibling.Status != model.StatusDone {
						allDone = false
						break
					}
				}
				if allDone {
					parentStatus := string(model.StatusDone)
					parentEvent := model.Event{
						ID:        parent.ID,
						Type:      model.EventTypeUpdate,
						Payload:   model.UpdatePayload{Status: &parentStatus},
						CreatedAt: timestamp,
						CreatedBy: user,
					}
					eventsToAppend = append(eventsToAppend, parentEvent)
					messages = append(messages, fmt.Sprintf("Auto-completed parent %s", parent.ID))
				}
			}
		}
	}

	// 4. Append Events
	for _, evt := range eventsToAppend {
		if err := t.appendEvent(evt); err != nil {
			return nil, fmt.Errorf("failed to append event for %s: %w", evt.ID, err)
		}
	}

	return messages, nil
}

// UpdateIssue updates an issue, handles side effects, and optionally
// creates a git commit when AutoCommit is enabled.
func (t *LocalTransport) UpdateIssue(id string, payload model.UpdatePayload, action string) ([]string, error) {
	messages, err := t.applyUpdate(id, payload)
	if err != nil {
		return nil, err
	}

	if t.Config.AutoCommit {
		commitMsg := fmt.Sprintf("xpo: %s %s", action, id)
		if len(messages) > 1 {
			commitMsg += " (with cascading updates)"
		}
		_ = exec.Command("git", "add", ".xpo/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return messages, nil
}

// GenerateUpdateTemplate generates text content for editing an issue.
func GenerateUpdateTemplate(i *model.Issue) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Title: %s\n", i.Title))
	sb.WriteString(fmt.Sprintf("Status: %s\n", i.Status))
	sb.WriteString(fmt.Sprintf("Parent: %s\n", i.ParentID))
	sb.WriteString(fmt.Sprintf("Estimate: %d\n", i.Estimate))
	sb.WriteString(fmt.Sprintf("Assignee: %s\n", i.Assignee))
	if len(i.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(i.Labels, ", ")))
	}
	sb.WriteString("\n") // Double newline separates headers from description
	sb.WriteString(i.Description)

	return sb.String()
}

// ParseUpdateContent parses edited text content into an UpdatePayload.
func ParseUpdateContent(content string, original *model.Issue) (*model.UpdatePayload, error) {
	// Split into Headers and Body
	parts := strings.SplitN(content, "\n\n", 2)

	headerBlock := parts[0]
	descriptionBlock := ""
	if len(parts) > 1 {
		descriptionBlock = parts[1]
	}

	// Parse Headers
	meta := make(map[string]string)
	headerLines := strings.Split(headerBlock, "\n")
	for _, line := range headerLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kp := strings.SplitN(line, ":", 2)
		if len(kp) == 2 {
			key := strings.TrimSpace(strings.ToLower(kp[0]))
			val := strings.TrimSpace(kp[1])
			meta[key] = val
		}
	}

	payload := &model.UpdatePayload{}

	if val, ok := meta["title"]; ok {
		if val != original.Title {
			payload.Title = &val
		}
	}

	if val, ok := meta["status"]; ok {
		if val != string(original.Status) {
			statusVal := val
			payload.Status = &statusVal
		}
	}
	if val, ok := meta["parent"]; ok {
		if val != original.ParentID {
			valCopy := val
			payload.ParentID = &valCopy
		}
	}
	if val, ok := meta["estimate"]; ok {
		est, err := strconv.Atoi(val)
		if err == nil {
			if est != original.Estimate {
				payload.Estimate = &est
			}
		}
	}
	if val, ok := meta["assignee"]; ok {
		if val != original.Assignee {
			valCopy := val
			payload.Assignee = &valCopy
		}
	}
	if val, ok := meta["labels"]; ok {
		labels := strings.Split(val, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}
		// Filter out empty strings
		var filtered []string
		for _, l := range labels {
			if l != "" {
				filtered = append(filtered, l)
			}
		}
		labels = filtered

		changed := false
		if len(labels) != len(original.Labels) {
			changed = true
		} else {
			for i, l := range labels {
				if l != original.Labels[i] {
					changed = true
					break
				}
			}
		}

		if changed {
			payload.Labels = labels
		}
	}

	// Handle Description
	newDesc := strings.TrimSpace(descriptionBlock)
	if newDesc != original.Description {
		payload.Description = &newDesc
	}

	return payload, nil
}
