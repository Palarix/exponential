package exponential

import (
	"fmt"
	"strings"

	"github.com/palarix/exponential/internal/model"
)

// Unlink removes one relationship between source and target, whichever
// issue stores it. Links are bidirectional, so dropping the entry from the
// source's combined view and writing it back removes the stored row on
// either side. It returns the removed link as seen from source.
func (c *Client) Unlink(source, target, kind string) (model.Dependency, []string, error) {
	if source == "" || target == "" {
		return model.Dependency{}, nil, fmt.Errorf("'source' and 'target' are required")
	}
	k := model.NormalizeDependencyKind(kind)
	if k == "" {
		return model.Dependency{}, nil, fmt.Errorf("invalid link type %q", kind)
	}
	src, err := c.GetIssue(source)
	if err != nil {
		return model.Dependency{}, nil, fmt.Errorf("source: %w", err)
	}
	targetID, err := c.resolveLinkTarget(src, target)
	if err != nil {
		return model.Dependency{}, nil, err
	}

	remaining := make([]model.Dependency, 0, len(src.Links))
	var removed *model.Dependency
	for _, l := range src.Links {
		if removed == nil && l.TargetID == targetID && string(l.Kind) == k {
			removed = &l
			continue
		}
		remaining = append(remaining, l)
	}
	if removed == nil {
		return model.Dependency{}, nil, fmt.Errorf("no link %s %s on %s", k, targetID, src.ID)
	}

	msgs, err := c.UpdateIssue(src.ID, model.UpdatePayload{Dependencies: remaining}, "unlink")
	if err != nil {
		return model.Dependency{}, nil, err
	}
	return *removed, msgs, nil
}

// resolveLinkTarget resolves target like any issue ID. If no live issue
// matches (it was deleted or purged), it falls back to the source's link
// targets so links to vanished issues stay removable.
func (c *Client) resolveLinkTarget(src *model.Issue, target string) (string, error) {
	tgt, err := c.GetIssue(target)
	if err == nil {
		return tgt.ID, nil
	}
	var matches []string
	seen := map[string]bool{}
	for _, l := range src.Links {
		if seen[l.TargetID] || !strings.Contains(l.TargetID, target) {
			continue
		}
		if l.TargetID == target {
			return target, nil
		}
		seen[l.TargetID] = true
		matches = append(matches, l.TargetID)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return "", fmt.Errorf("target: %w", err)
}
