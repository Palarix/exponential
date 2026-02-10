package beats

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

// UpdateIssue updates an issue and handles side effects.
func (c *Client) UpdateIssue(id string, payload model.UpdatePayload, action string) ([]string, error) {
	// 1. Read and Project State
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	targetIssue, err := c.resolveIssue(issues, id)
	if err != nil {
		return nil, err
	}
	id = targetIssue.ID // Use the resolved ID

	user := c.GetUser()
	timestamp := time.Now().UTC()
	var eventsToAppend []model.Event
	var messages []string

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

		// --- Cannot complete parent with incomplete children (unless auto-close is on) ---
		if newStatus == model.StatusDone && !c.Config.Automations.AutoCloseSubIssues {
			hasIncompleteChildren := false
			for _, child := range issues {
				if child.ParentID == id && child.Status != model.StatusDone && !child.Deleted {
					hasIncompleteChildren = true
					break
				}
			}
			if hasIncompleteChildren {
				return nil, fmt.Errorf("cannot complete issue %s: it has unfinished sub-issues", id)
			}
		}

		// --- Automation: auto-close sub-issues when parent is closed ---
		if newStatus == model.StatusDone && c.Config.Automations.AutoCloseSubIssues {
			for _, child := range issues {
				if child.ParentID == id && child.Status != model.StatusDone && !child.Deleted {
					childStatus := string(model.StatusDone)
					childPayload := model.UpdatePayload{Status: &childStatus}
					childEvent := model.Event{
						ID:        child.ID,
						Type:      model.EventTypeUpdate,
						Payload:   childPayload,
						CreatedAt: timestamp,
						CreatedBy: user,
					}
					eventsToAppend = append(eventsToAppend, childEvent)
					messages = append(messages, fmt.Sprintf("Auto-closed sub-issue %s", child.ID))
				}
			}
		}

		// --- Automation: auto-progress sub-issues when parent is progressed ---
		if c.Config.Automations.AutoProgressSubIssues {
			for _, child := range issues {
				if child.ParentID != id || child.Deleted {
					continue
				}
				// backlog -> todo: If parent moves to PLANNED, move BACKLOG children to PLANNED
				if newStatus == model.StatusPlanned && child.Status == model.StatusBacklog {
					childStatus := string(model.StatusPlanned)
					childPayload := model.UpdatePayload{Status: &childStatus}
					childEvent := model.Event{
						ID:        child.ID,
						Type:      model.EventTypeUpdate,
						Payload:   childPayload,
						CreatedAt: timestamp,
						CreatedBy: user,
					}
					eventsToAppend = append(eventsToAppend, childEvent)
					messages = append(messages, fmt.Sprintf("Auto-progressed sub-issue %s to PLANNED", child.ID))
				}
				// todo -> doing: If parent moves to DOING, move PLANNED children to DOING
				if newStatus == model.StatusDoing && child.Status == model.StatusPlanned {
					childStatus := string(model.StatusDoing)
					childPayload := model.UpdatePayload{Status: &childStatus}
					childEvent := model.Event{
						ID:        child.ID,
						Type:      model.EventTypeUpdate,
						Payload:   childPayload,
						CreatedAt: timestamp,
						CreatedBy: user,
					}
					eventsToAppend = append(eventsToAppend, childEvent)
					messages = append(messages, fmt.Sprintf("Auto-progressed sub-issue %s to DOING", child.ID))
				}
			}
		}
	}

	// 4. Commit Changes
	for _, evt := range eventsToAppend {
		if err := storage.AppendEvent(evt); err != nil {
			return nil, fmt.Errorf("error appending event for %s: %w", evt.ID, err)
		}
	}

	// 5. Git Commit (if enabled)
	if c.Config.AutoCommit {
		commitMsg := fmt.Sprintf("beats: %s %s", action, id)
		if len(eventsToAppend) > 1 {
			commitMsg += " (with cascading updates)"
		}
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
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
