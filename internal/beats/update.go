package beats

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
)

// UpdateIssue updates an issue and handles side effects.
func (c *Client) UpdateIssue(id string, payload model.UpdatePayload, action string) ([]string, error) {
	// 1. Read and Project State
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}
	issues := ProjectIssues(events)

	targetIssue, exists := issues[id]
	if !exists {
		return nil, fmt.Errorf("issue %s not found", id)
	}

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
	// 3. Logic & Side Effects based on Status Change
	if payload.Status != nil {
		newStatus := model.IssueStatus(*payload.Status)

		// --- Scenario: Start Task -> Check Blocking ---
		if newStatus == model.StatusDoing {
			// Check existing blockers
			for _, dep := range targetIssue.Dependencies {
				if dep.Kind == model.DependencyBlockedBy {
					blocker, bExists := issues[dep.TargetID]
					if bExists && blocker.Status != model.StatusDone && !blocker.Deleted {
						return nil, fmt.Errorf("cannot start issue %s: blocked by incomplete issue %s", id, blocker.ID)
					}
				}
			}
			// Check new blocker from payload (compatibility)
			if payload.BlockedBy != nil && *payload.BlockedBy != "" {
				blocker, bExists := issues[*payload.BlockedBy]
				if bExists && blocker.Status != model.StatusDone && !blocker.Deleted {
					return nil, fmt.Errorf("cannot start issue %s: blocked by incomplete issue %s", id, blocker.ID)
				}
			}
		}

		// --- Scenario: Manual Complete Epic -> Validation ---
		if newStatus == model.StatusDone && targetIssue.Kind == "EPIC" {
			// Check if this issue is a parent with incomplete children
			// We need to look up children. The `issues` map contains all.
			hasIncompleteChildren := false
			for _, child := range issues {
				if child.ParentID == id && child.Status != model.StatusDone && !child.Deleted {
					hasIncompleteChildren = true
					break
				}
			}
			if hasIncompleteChildren {
				return nil, fmt.Errorf("incorrect status: cannot complete epic %s because it has unfinished child tasks", id)
			}
		}

		// Previous Auto-Start / Auto-Complete logic removed.
		// Epic state is now derived on projection.
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
		// We ignore error here but print if we could?
		// Client logic usually shouldn't print. Maybe return warning or log?
		// For now we just exec.
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", commitMsg).Run()
	}

	return messages, nil
}

// GenerateUpdateTemplate generates text content for editing an issue.
// GenerateUpdateTemplate generates text content for editing an issue.
func GenerateUpdateTemplate(i *model.Issue) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Title: %s\n", i.Title))
	sb.WriteString(fmt.Sprintf("Status: %s\n", i.Status))
	sb.WriteString(fmt.Sprintf("Parent: %s\n", i.ParentID))
	sb.WriteString(fmt.Sprintf("Estimate: %d\n", i.Estimate))
	sb.WriteString(fmt.Sprintf("Blocked By: %s\n", i.BlockedBy))
	sb.WriteString(fmt.Sprintf("Block Reason: %s\n", i.BlockReason))
	// Add new fields as comments or read-only for now
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
		// Expect "Key: Value"
		kp := strings.SplitN(line, ":", 2)
		if len(kp) == 2 {
			key := strings.TrimSpace(strings.ToLower(kp[0]))
			val := strings.TrimSpace(kp[1])
			meta[key] = val
		}
	}

	payload := &model.UpdatePayload{}

	// Map headers to payload
	if val, ok := meta["title"]; ok {
		if val != original.Title {
			payload.Title = &val
		}
	}

	if val, ok := meta["status"]; ok {
		if val != string(original.Status) {
			statusVal := val // Need addressable string
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
	if val, ok := meta["blocked by"]; ok {
		if val != original.BlockedBy {
			valCopy := val
			payload.BlockedBy = &valCopy
		}
	}
	if val, ok := meta["block reason"]; ok {
		if val != original.BlockReason {
			valCopy := val
			payload.BlockReason = &valCopy
		}
	}
	if val, ok := meta["labels"]; ok {
		// Simple comma separated parser
		labels := strings.Split(val, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}
		// TODO: Compare with original to see if changed?
		// For now always update if present in template?
		// Actually, if simply re-saving unchanged list, it creates event.
		// Ideally we check equality.
		changed := false
		if len(labels) != len(original.Labels) {
			changed = true
		} else {
			// Compare content (order matters for simple equality check)
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
