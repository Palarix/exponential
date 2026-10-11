package exponential

import (
	"fmt"
	"strings"

	"github.com/palarix/exponential/internal/model"
)

// Link adds one relationship from source to target. Two issues may hold at
// most one link, whichever side stores it: if they are already linked by a
// different kind, Link refuses unless overwrite is set, in which case the
// existing link is removed from whichever issue stores it. It returns the
// added link and the links it replaced, both as seen from source.
func (c *Client) Link(source, target, kind string, overwrite bool) (model.Dependency, []model.Dependency, []string, error) {
	if source == "" || target == "" {
		return model.Dependency{}, nil, nil, fmt.Errorf("'source' and 'target' are required")
	}
	k := model.NormalizeDependencyKind(kind)
	if k == "" {
		return model.Dependency{}, nil, nil, fmt.Errorf("invalid link type %q", kind)
	}
	src, err := c.GetIssue(source)
	if err != nil {
		return model.Dependency{}, nil, nil, fmt.Errorf("source: %w", err)
	}
	tgt, err := c.GetIssue(target)
	if err != nil {
		return model.Dependency{}, nil, nil, fmt.Errorf("target: %w", err)
	}
	if src.ID == tgt.ID {
		return model.Dependency{}, nil, nil, fmt.Errorf("cannot link an issue to itself")
	}
	added := model.Dependency{SourceID: src.ID, TargetID: tgt.ID, Kind: model.DependencyKind(k)}

	// Links includes derived inverses, so this sees a link stored on target too.
	var kept, replaced []model.Dependency
	identical := false
	for _, l := range src.Links {
		switch {
		case l.TargetID != tgt.ID:
			kept = append(kept, l)
		case l.Kind == added.Kind:
			identical = true
		default:
			replaced = append(replaced, l)
		}
	}
	if identical && len(replaced) == 0 {
		return model.Dependency{}, nil, nil, fmt.Errorf("link %s %s already exists on %s", k, tgt.ID, src.ID)
	}
	if len(replaced) > 0 && !overwrite {
		kinds := make([]string, len(replaced))
		for i, l := range replaced {
			kinds[i] = string(l.Kind)
		}
		return model.Dependency{}, nil, nil, fmt.Errorf("%s is already linked to %s (%s); set overwrite to replace it with %s",
			src.ID, tgt.ID, strings.Join(kinds, ", "), k)
	}

	msgs, err := c.UpdateIssue(src.ID, model.UpdatePayload{Dependencies: append(kept, added)}, "link")
	if err != nil {
		return model.Dependency{}, nil, nil, err
	}
	return added, replaced, msgs, nil
}
