# Parent issues with no visible children show a blank slot instead of the chevron

## What
In the Backlog (nested hierarchy mode), a parent row whose children are all hidden from the current view renders a static, unrotated chevron (`BacklogIssueRow.tsx`, non-button branch). It looks like a collapsed, expandable node but clicking does nothing. Render the blank spacer instead, the same one used for issues without children.

## Why
The chevron promises an affordance that doesn't exist. The row still shows the `done/total` sub-progress, so the "this is a parent" signal isn't lost.

## When does this happen
`hasChildren && !hasVisibleChildren` in nested mode:
- all children are terminal and ghosts are hidden (`showGhosts = false`)
- all children sit in another status group that's visible on the same tab (they render there under a ghost parent)
- ghost-child rows that are themselves parents (`hasVisibleChildren` is always `false` for them)

## How
- Extract a pure helper `nodeIndicator({ hasChildren, hasVisibleChildren, isGhostParent, hierarchyMode })` in a new `backlog-row-utils.ts` returning:
  - `"toggle"` — nested, has visible children, not a ghost parent → clickable chevron (unchanged)
  - `"expanded"` — nested, ghost parent with visible children → static rotated chevron (unchanged)
  - `"none"` — everything else (no children, no visible children, flat mode) → blank `w-4` spacer
- `BacklogIssueRow` switches on the result.

## Decisions
- **Blank instead of an icon.** First tried lucide `SquareDashedText`; on review the user preferred a blank slot. No tooltip, since there's nothing to hover.

## Acceptance Criteria
- [ ] Nested mode: a parent whose children are all hidden from the view shows a blank slot, not a chevron
- [ ] Clickable chevron for parents with visible children is unchanged
- [ ] Ghost parents with visible children still show the static expanded chevron
- [ ] Row alignment unchanged (blank slot has the same width as the chevron slot)
- [ ] Unit tests cover all `nodeIndicator` outcomes
- [ ] `make test` passes
