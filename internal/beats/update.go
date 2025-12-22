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
	if payload.Status != nil {
		newStatus := model.IssueStatus(*payload.Status)

		// --- Scenario: Start Child -> Auto-Start Parent ---
		if newStatus == model.StatusDoing && targetIssue.ParentID != "" {
			parent, pExists := issues[targetIssue.ParentID]
			if pExists && parent.Status != model.StatusDoing && parent.Status != model.StatusDone {
				// Auto-start parent
				pStatus := string(model.StatusDoing)
				parentEvent := model.Event{
					ID:        parent.ID,
					Type:      model.EventTypeUpdate,
					Payload:   model.UpdatePayload{Status: &pStatus},
					CreatedAt: timestamp, // Logical simultaneity
					CreatedBy: user,
				}
				eventsToAppend = append(eventsToAppend, parentEvent)
				messages = append(messages, fmt.Sprintf("Auto-started parent epic %s", parent.ID))
			}
		}

		// --- Scenario: Manual Complete Epic -> Validation ---
		if newStatus == model.StatusDone {
			// Check if this issue is a parent with incomplete children
			hasIncompleteChildren := false
			for _, child := range issues {
				if child.ParentID == id && child.Status != model.StatusDone {
					hasIncompleteChildren = true
					break
				}
			}
			if hasIncompleteChildren {
				return nil, fmt.Errorf("incorrect status: cannot complete epic %s because it has unfinished child tasks", id)
			}
		}

		// --- Scenario: Complete Child -> Auto-Complete Parent ---
		if newStatus == model.StatusDone && targetIssue.ParentID != "" {
			parent, pExists := issues[targetIssue.ParentID]
			if pExists && parent.Status != model.StatusDone {
				// Check if ALL OTHER children are done
				allSiblingsDone := true
				for _, other := range issues {
					if other.ParentID == parent.ID && other.ID != id { // Skip self (we are becoming done)
						if other.Status != model.StatusDone {
							allSiblingsDone = false
							break
						}
					}
				}

				if allSiblingsDone {
					// Auto-complete parent
					pStatus := string(model.StatusDone)
					parentEvent := model.Event{
						ID:        parent.ID,
						Type:      model.EventTypeUpdate,
						Payload:   model.UpdatePayload{Status: &pStatus},
						CreatedAt: timestamp,
						CreatedBy: user,
					}
					eventsToAppend = append(eventsToAppend, parentEvent)
					messages = append(messages, fmt.Sprintf("Auto-completed parent epic %s (all children done)", parent.ID))
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
		// We ignore error here but print if we could?
		// Client logic usually shouldn't print. Maybe return warning or log?
		// For now we just exec.
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
	sb.WriteString(fmt.Sprintf("Blocked By: %s\n", i.BlockedBy))
	sb.WriteString(fmt.Sprintf("Block Reason: %s\n", i.BlockReason))
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

	// Handle Description
	newDesc := strings.TrimSpace(descriptionBlock)
	if newDesc != original.Description {
		payload.Description = &newDesc
	}

	return payload, nil
}
