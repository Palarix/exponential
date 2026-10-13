package exponential

import (
	"fmt"
	"sort"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/sortorder"
)

// Position is a resolved before/after request: the sort_order key that
// places an issue directly above or below Target.
type Position struct {
	Key    string
	Target string
	Above  bool
}

// Unchanged is the message for an update whose position was already in place.
func (p *Position) Unchanged(id string) string {
	where := "after"
	if p.Above {
		where = "before"
	}
	return fmt.Sprintf("%s is already %s %s", id, where, p.Target)
}

// PositionKey resolves a before/after request for issue selfID (empty for a
// new issue) into a sort_order key. It returns nil when neither is set. The
// neighbor comes from the global order of all issues, so the issue lands
// next to the target in every filtered view (xpo-b92fb0).
func (c *Client) PositionKey(selfID, before, after string) (*Position, error) {
	if before != "" && after != "" {
		return nil, fmt.Errorf("'before' and 'after' are mutually exclusive")
	}
	ref := before
	if ref == "" {
		ref = after
	}
	if ref == "" {
		return nil, nil
	}
	target, err := c.GetIssue(ref)
	if err != nil {
		return nil, fmt.Errorf("position target: %w", err)
	}
	if target.ID == selfID {
		return nil, fmt.Errorf("cannot position issue %s relative to itself", selfID)
	}
	issues, err := c.ListIssues(FilterOptions{All: true})
	if err != nil {
		return nil, err
	}
	key, err := keyBeside(issues, selfID, target.ID, after != "")
	if err != nil {
		return nil, err
	}
	return &Position{Key: key, Target: target.ID, Above: before != ""}, nil
}

// keyBeside returns a key directly below (or above) targetID among issues,
// ignoring selfID. If selfID already sits there, its current key is returned
// so the update is a no-op. Keys equal to the target's are skipped: legacy
// data can hold duplicates, and GenerateKeyBetween rejects equal bounds.
func keyBeside(issues []*model.Issue, selfID, targetID string, below bool) (string, error) {
	var targetKey, selfKey string
	var keys []string
	for _, i := range issues {
		switch i.ID {
		case targetID:
			targetKey = i.SortOrder
		case selfID:
			selfKey = i.SortOrder
			continue
		}
		if i.SortOrder != "" {
			keys = append(keys, i.SortOrder)
		}
	}
	if targetKey == "" {
		return "", fmt.Errorf("issue %s has no sort order", targetID)
	}
	sort.Strings(keys)

	if below {
		neighbor := ""
		if n := sort.SearchStrings(keys, targetKey+"\x00"); n < len(keys) {
			neighbor = keys[n]
		}
		if selfKey > targetKey && (neighbor == "" || selfKey < neighbor) {
			return selfKey, nil
		}
		return sortorder.GenerateKeyBetween(targetKey, neighbor)
	}

	neighbor := ""
	if n := sort.SearchStrings(keys, targetKey); n > 0 {
		neighbor = keys[n-1]
	}
	if selfKey != "" && selfKey < targetKey && selfKey > neighbor {
		return selfKey, nil
	}
	return sortorder.GenerateKeyBetween(neighbor, targetKey)
}
