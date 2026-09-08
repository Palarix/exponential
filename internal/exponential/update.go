package exponential

import (
	"fmt"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/sortorder"
	"github.com/palarix/exponential/internal/storage"
)

// buildUpdate resolves the target issue, prunes unchanged fields, and
// returns the events that should be appended (primary update + cascading
// automations) along with human-readable messages.  It performs no I/O;
// callers are responsible for persisting the returned events.
func (t *LocalTransport) buildUpdate(id string, payload model.UpdatePayload, issues map[string]*model.Issue) ([]model.Event, []string, error) {
	targetIssue, err := t.resolveIssue(issues, id)
	if err != nil {
		return nil, nil, err
	}
	id = targetIssue.ID // Use the resolved ID

	user := t.GetUser()
	timestamp := time.Now().UTC()
	var eventsToAppend []model.Event
	var messages []string

	// Guard: reject self-referencing parent
	if payload.ParentID != nil && *payload.ParentID == id {
		return nil, nil, fmt.Errorf("cannot set issue %s as its own parent", id)
	}

	payload.Labels = normalizeLabels(payload.Labels, t.Config)

	// Prune fields that already match current state so redundant updates are no-ops.
	pruneUnchangedFields(&payload, targetIssue)
	if payloadEmpty(payload) {
		return nil, nil, nil
	}

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
			}
		}
	}

	// Prepare Primary Update
	primaryEvent := model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   payload,
		CreatedAt: timestamp,
		CreatedBy: user,
	}
	eventsToAppend = append(eventsToAppend, primaryEvent)
	messages = append(messages, fmt.Sprintf("Updated %s", id))

	// Cascading side effects based on status change
	if payload.Status != nil {
		newStatus := model.IssueStatus(*payload.Status)

		// --- Check blocked_by dependencies before starting ---
		if newStatus == model.StatusDoing {
			for _, dep := range targetIssue.Dependencies {
				if dep.Kind == model.DependencyBlockedBy {
					blocker, bExists := issues[dep.TargetID]
					if bExists && !model.IsTerminal(blocker.Status) && !blocker.Deleted {
						return nil, nil, fmt.Errorf("cannot start issue %s: blocked by incomplete issue %s", id, blocker.ID)
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

		// --- Last-completed trigger: parent auto-closes when last child reaches a terminal status ---
		if t.Config.Automations.LastCompleted && targetIssue.ParentID != "" && model.IsTerminal(newStatus) {
			parent, pExists := issues[targetIssue.ParentID]
			if pExists && !parent.Deleted && !model.IsTerminal(parent.Status) {
				allDone := true
				for _, sibling := range issues {
					if sibling.ParentID != parent.ID || sibling.Deleted || sibling.ID == id {
						continue
					}
					if !model.IsTerminal(sibling.Status) {
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

	return eventsToAppend, messages, nil
}

// applyUpdate reads current state, builds update events via buildUpdate,
// and appends them to storage.  Does not create a git commit.
func (t *LocalTransport) applyUpdate(id string, payload model.UpdatePayload) ([]string, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	issues := ProjectIssues(events)

	toAppend, messages, err := t.buildUpdate(id, payload, issues)
	if err != nil {
		return nil, err
	}

	for _, evt := range toAppend {
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
		hub := storage.HubRoot()
		commitMsg := fmt.Sprintf("xpo: %s %s", action, id)
		if len(messages) > 1 {
			commitMsg += " (with cascading updates)"
		}
		_ = exec.Command("git", "-C", hub, "add", ".xpo/issues.db").Run()
		_ = exec.Command("git", "-C", hub, "commit", "-m", commitMsg).Run()
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

// pruneUnchangedFields nils out payload fields that already match the
// current issue state so redundant updates produce no events.  Fields
// are matched by name between UpdatePayload and Issue using reflection,
// so new fields are handled automatically.
func pruneUnchangedFields(p *model.UpdatePayload, issue *model.Issue) {
	pv := reflect.ValueOf(p).Elem()
	iv := reflect.ValueOf(issue).Elem()
	pt := pv.Type()

	for i := 0; i < pt.NumField(); i++ {
		pf := pv.Field(i)
		name := pt.Field(i).Name
		issueField := iv.FieldByName(name)
		if !issueField.IsValid() {
			continue
		}

		switch pf.Kind() {
		case reflect.Ptr:
			if pf.IsNil() {
				continue
			}
			if fieldsEqual(pf.Elem(), issueField) {
				pf.Set(reflect.Zero(pf.Type()))
			}
		case reflect.Slice:
			if pf.IsNil() {
				continue
			}
			if fieldsEqual(pf, issueField) {
				pf.Set(reflect.Zero(pf.Type()))
			}
		}
	}
}

func payloadEmpty(p model.UpdatePayload) bool {
	v := reflect.ValueOf(p)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		switch f.Kind() {
		case reflect.Ptr:
			if !f.IsNil() {
				return false
			}
		case reflect.Slice:
			if !f.IsNil() {
				return false
			}
		}
	}
	return true
}

// fieldsEqual compares a payload value against an issue value.  Type
// aliases with the same underlying kind (e.g. string vs IssueStatus)
// are converted before comparing.  Slices are compared as unordered sets.
func fieldsEqual(a, b reflect.Value) bool {
	if a.Type() != b.Type() {
		if a.Type().ConvertibleTo(b.Type()) {
			a = a.Convert(b.Type())
		} else {
			return false
		}
	}
	if a.Kind() == reflect.Slice {
		return unorderedSlicesEqual(a, b)
	}
	return reflect.DeepEqual(a.Interface(), b.Interface())
}

func unorderedSlicesEqual(a, b reflect.Value) bool {
	if a.Len() != b.Len() {
		return false
	}
	if a.Len() == 0 {
		return true
	}
	as := sprintSorted(a)
	bs := sprintSorted(b)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func sprintSorted(v reflect.Value) []string {
	s := make([]string, v.Len())
	for i := range s {
		s[i] = fmt.Sprintf("%v", v.Index(i).Interface())
	}
	sort.Strings(s)
	return s
}
