package jsonio

import (
	"encoding/json"

	"github.com/palarix/exponential/internal/identity"
	"github.com/palarix/exponential/internal/model"
)

// FieldChange is one field an event changed. Scalar fields use From and To;
// set fields (labels, dependencies) use Added and Removed. An empty value is
// left out, so a cleared field has only From.
type FieldChange struct {
	From    any `json:"from,omitempty"`
	To      any `json:"to,omitempty"`
	Added   any `json:"added,omitempty"`
	Removed any `json:"removed,omitempty"`
}

// issueState is the part of an issue that event changes are reported for.
type issueState struct {
	title, description, status, parentID string
	assignee, cycleID, milestoneID       string
	estimate, priority                   int
	labels                               []string
	deps                                 []model.Dependency
}

// eventChanges replays one issue's events in order and returns, per event,
// the fields it changed. Events that changed nothing shown get nil.
func eventChanges(es []model.Event) []map[string]FieldChange {
	out := make([]map[string]FieldChange, len(es))
	var st *issueState
	for i, e := range es {
		switch e.Type {
		case model.EventTypeCreate:
			var p model.CreatePayload
			decodePayload(e.Payload, &p)
			status := p.Status
			if status == "" {
				status = string(model.StatusBacklog)
			}
			assignee, _ := identity.ResolveAssignment(p.Assignee, e.CreatedBy, e.OnBehalfOf)
			st = &issueState{
				title: p.Title, description: p.Description, status: status,
				parentID: p.ParentID, assignee: assignee, cycleID: p.CycleID,
				milestoneID: p.MilestoneID, estimate: p.Estimate, priority: p.Priority,
				labels: p.Labels, deps: p.Dependencies,
			}
		case model.EventTypeUpdate:
			if st == nil {
				continue
			}
			var p model.UpdatePayload
			decodePayload(e.Payload, &p)
			if p.Assignee != nil {
				// Resolve like the projection does, so values match show's assignee.
				a, _ := identity.ResolveAssignment(*p.Assignee, e.CreatedBy, e.OnBehalfOf)
				p.Assignee = &a
			}
			out[i] = st.applyUpdate(p)
		case model.EventTypeMerge:
			if st == nil {
				continue
			}
			c := map[string]FieldChange{}
			scalar(c, "status", &st.status, strPtr(string(model.StatusDone)))
			out[i] = nonEmpty(c)
		}
	}
	return out
}

func (st *issueState) applyUpdate(p model.UpdatePayload) map[string]FieldChange {
	c := map[string]FieldChange{}
	scalar(c, "title", &st.title, p.Title)
	scalar(c, "description", &st.description, p.Description)
	scalar(c, "status", &st.status, p.Status)
	scalar(c, "parent_id", &st.parentID, p.ParentID)
	scalar(c, "story_points", &st.estimate, p.Estimate)
	scalar(c, "priority", &st.priority, p.Priority)
	scalar(c, "assignee", &st.assignee, p.Assignee)
	scalar(c, "cycle_id", &st.cycleID, p.CycleID)
	scalar(c, "milestone_id", &st.milestoneID, p.MilestoneID)
	if p.Labels != nil {
		added, removed := setDiff(st.labels, p.Labels, func(s string) string { return s })
		if len(added) > 0 || len(removed) > 0 {
			c["labels"] = FieldChange{Added: nilIfEmpty(added), Removed: nilIfEmpty(removed)}
		}
		st.labels = p.Labels
	}
	if p.Dependencies != nil {
		key := func(d model.Dependency) string { return d.TargetID + "\x00" + string(d.Kind) }
		added, removed := setDiff(st.deps, p.Dependencies, key)
		if len(added) > 0 || len(removed) > 0 {
			c["dependencies"] = FieldChange{Added: nilIfEmpty(added), Removed: nilIfEmpty(removed)}
		}
		st.deps = p.Dependencies
	}
	return nonEmpty(c)
}

// scalar records a from/to change when next is set and differs from *cur,
// then applies it.
func scalar[T comparable](c map[string]FieldChange, key string, cur *T, next *T) {
	if next == nil || *next == *cur {
		return
	}
	var zero T
	var fc FieldChange
	if *cur != zero {
		fc.From = *cur
	}
	if *next != zero {
		fc.To = *next
	}
	c[key] = fc
	*cur = *next
}

// setDiff returns the items of next not in prev (added) and of prev not in
// next (removed), each in its list's order.
func setDiff[T any](prev, next []T, key func(T) string) (added, removed []T) {
	inPrev := make(map[string]bool, len(prev))
	for _, v := range prev {
		inPrev[key(v)] = true
	}
	inNext := make(map[string]bool, len(next))
	for _, v := range next {
		inNext[key(v)] = true
		if !inPrev[key(v)] {
			added = append(added, v)
		}
	}
	for _, v := range prev {
		if !inNext[key(v)] {
			removed = append(removed, v)
		}
	}
	return added, removed
}

// nilIfEmpty turns an empty slice into an untyped nil so omitempty drops it.
func nilIfEmpty[T any](s []T) any {
	if len(s) == 0 {
		return nil
	}
	return s
}

func nonEmpty(c map[string]FieldChange) map[string]FieldChange {
	if len(c) == 0 {
		return nil
	}
	return c
}

func decodePayload(payload interface{}, dst interface{}) {
	b, _ := json.Marshal(payload)
	_ = json.Unmarshal(b, dst)
}

func strPtr(s string) *string { return &s }
