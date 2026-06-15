package beats

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/palarix/beats/internal/model"
)

// InboxItem is a single event surfaced in a user's personal inbox feed.
type InboxItem struct {
	IssueID    string          `json:"issue_id"`
	IssueTitle string          `json:"issue_title"`
	Type       model.EventType `json:"type"`
	Payload    interface{}     `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
	CreatedBy  string          `json:"created_by"`
}

// emailOf extracts a lowercased email from a "Name <email>" or bare-email
// identity string, or "" if none is present.
func emailOf(identity string) string {
	identity = strings.TrimSpace(identity)
	if i := strings.LastIndex(identity, "<"); i != -1 {
		if j := strings.LastIndex(identity, ">"); j > i {
			return strings.ToLower(strings.TrimSpace(identity[i+1 : j]))
		}
	}
	if strings.Contains(identity, "@") {
		return strings.ToLower(identity)
	}
	return ""
}

// identityMatches reports whether two actor strings refer to the same person.
// Both are typically "Name <email>" but may be partial. Email is compared
// when both sides expose one; otherwise a case-insensitive exact match.
func identityMatches(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if ea, eb := emailOf(a), emailOf(b); ea != "" && eb != "" {
		return ea == eb
	}
	return strings.EqualFold(a, b)
}

// BuildInbox filters the full event log down to events relevant to `me` that
// occurred after `since`, newest-first. Relevance = events on issues where
// the user is the creator, current assignee, or has commented — plus all
// MERGE events (team-wide visibility). The user's own actions are excluded so
// the inbox shows what others did. A zero `since` returns all matching events.
func BuildInbox(events []model.Event, issues map[string]*model.Issue, me string, since time.Time) []InboxItem {
	// Determine the set of issues the user participates in.
	participates := make(map[string]bool, len(issues))
	titles := make(map[string]string, len(issues))
	for id, issue := range issues {
		titles[id] = issue.Title
		if identityMatches(issue.CreatedBy, me) || identityMatches(issue.Assignee, me) {
			participates[id] = true
			continue
		}
		for _, c := range issue.Comments {
			if identityMatches(c.CreatedBy, me) {
				participates[id] = true
				break
			}
		}
	}
	// Also count any issue the user touched via updates (status transitions,
	// title edits, etc.) — these don't appear in the projected issue state.
	for _, evt := range events {
		if !participates[evt.ID] && identityMatches(evt.CreatedBy, me) {
			participates[evt.ID] = true
		}
	}

	var items []InboxItem
	for _, evt := range events {
		if !since.IsZero() && !evt.CreatedAt.After(since) {
			continue
		}
		if identityMatches(evt.CreatedBy, me) {
			continue // don't notify about your own actions
		}
		if !isMeaningfulInboxEvent(evt) {
			continue // skip bookkeeping noise (e.g. sort-order-only updates)
		}
		if !participates[evt.ID] && evt.Type != model.EventTypeMerge {
			continue
		}
		items = append(items, InboxItem{
			IssueID:    evt.ID,
			IssueTitle: titles[evt.ID],
			Type:       evt.Type,
			Payload:    evt.Payload,
			CreatedAt:  evt.CreatedAt,
			CreatedBy:  evt.CreatedBy,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items
}

// isMeaningfulInboxEvent reports whether an event is worth surfacing in the
// inbox. CREATE, COMMENT, and MERGE always are; UPDATE events that only carry
// a sort_order change are bookkeeping noise (e.g. drag-reorder, migrations)
// and are skipped. Works for both typed and map[string]interface{} payloads.
func isMeaningfulInboxEvent(evt model.Event) bool {
	switch evt.Type {
	case model.EventTypeCreate, model.EventTypeComment, model.EventTypeMerge:
		return true
	case model.EventTypeUpdate:
		var p model.UpdatePayload
		decodePayload(evt.Payload, &p)
		sortOrderOnly := p.SortOrder != nil &&
			p.Title == nil && p.Description == nil && p.Status == nil &&
			p.ParentID == nil && p.Estimate == nil && p.Priority == nil &&
			p.Assignee == nil && p.CycleID == nil &&
			len(p.Dependencies) == 0 && len(p.Labels) == 0
		return !sortOrderOnly
	default:
		return false
	}
}

// actorName returns the display name from a "Name <email>" identity, or the
// raw string if it has no email portion.
func actorName(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.LastIndex(raw, " <"); i != -1 {
		return strings.TrimSpace(raw[:i])
	}
	return raw
}

// decodePayload re-decodes an event payload (typically map[string]interface{}
// after JSON round-tripping) into a typed struct, mirroring projection.go.
func decodePayload(payload interface{}, target interface{}) {
	b, _ := json.Marshal(payload)
	_ = json.Unmarshal(b, target)
}

func commentSnippet(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	const max = 60
	if len(text) > max {
		return strings.TrimSpace(text[:max]) + "…"
	}
	return text
}

// FormatInboxItem renders a single inbox item as a one-line human sentence
// from the perspective of `me` (used to render "to you").
func FormatInboxItem(item InboxItem, me string) string {
	who := actorName(item.CreatedBy)
	id := item.IssueID

	switch item.Type {
	case model.EventTypeCreate:
		var p model.CreatePayload
		decodePayload(item.Payload, &p)
		if p.Assignee != "" && identityMatches(p.Assignee, me) {
			return fmt.Sprintf("%s created %s and assigned it to you", who, id)
		}
		return fmt.Sprintf("%s created %s", who, id)

	case model.EventTypeUpdate:
		var p model.UpdatePayload
		decodePayload(item.Payload, &p)
		if p.Assignee != nil {
			switch {
			case identityMatches(*p.Assignee, me):
				return fmt.Sprintf("%s assigned %s to you", who, id)
			case strings.TrimSpace(*p.Assignee) == "":
				return fmt.Sprintf("%s unassigned %s", who, id)
			default:
				return fmt.Sprintf("%s assigned %s to %s", who, id, actorName(*p.Assignee))
			}
		}
		if p.Status != nil {
			return fmt.Sprintf("%s changed %s status to %s", who, id, *p.Status)
		}
		return fmt.Sprintf("%s updated %s", who, id)

	case model.EventTypeComment:
		var p model.CommentPayload
		decodePayload(item.Payload, &p)
		if snippet := commentSnippet(p.Text); snippet != "" {
			return fmt.Sprintf("%s commented on %s %q", who, id, snippet)
		}
		return fmt.Sprintf("%s commented on %s", who, id)

	case model.EventTypeMerge:
		var p model.MergePayload
		decodePayload(item.Payload, &p)
		if p.Strategy != "" {
			return fmt.Sprintf("%s merged %s via %s", who, id, p.Strategy)
		}
		return fmt.Sprintf("%s merged %s", who, id)

	case model.EventTypeDelete:
		return fmt.Sprintf("%s deleted %s", who, id)

	default:
		return fmt.Sprintf("%s updated %s", who, id)
	}
}
