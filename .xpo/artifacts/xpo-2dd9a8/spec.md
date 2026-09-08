# Spec: Hide expand chevron on parents with no visible children

## What

In nested backlog mode, parents whose children are all filtered out still displayed a clickable expand chevron. Ghost parents also had clickable chevrons that toggled the real parent's expand state in another status group.

## Why

The chevron promises expandable content. An empty expand is confusing, and ghost parent toggles affecting real parents elsewhere is a side-effect leak.

## How

### Approach: `hasVisibleChildren` field + static chevrons

`hasChildren` (all children across statuses) is still needed for progress counters and estimate badges. A new `hasVisibleChildren` field tracks whether children actually render after ghost/status filtering.

### Chevron states

| State | Chevron | Clickable |
|---|---|---|
| Real parent, visible children | Rotates on expand/collapse | Yes |
| Ghost parent (always expanded) | Points down (static) | No |
| Parent, no visible children | Points right (static) | No |
| Non-parent | Spacer | — |

### Changes

**`useBacklogRows.ts`**
- Added `hasVisibleChildren: boolean` to RowItem
- Moved `allVisual` computation before parent row push; set `hasVisibleChildren: allVisual.length > 0`
- Ghost child rows: `hasVisibleChildren: false`
- Flat mode rows: mirrors `hasChildren` (chevrons not shown in flat mode)

**`BacklogIssueRow.tsx`**
- Clickable chevron gated on `hasVisibleChildren && !isGhostParent`
- Static chevron for ghost parents and empty parents (rotated when `hasVisibleChildren`)
- Ghost row opacity changed from 50% to 65% for readability

**`Backlog.tsx`**
- Keyboard nav (`ArrowRight`/`ArrowLeft`) and DnD drop-target logic switched to `hasVisibleChildren`

## Acceptance Criteria

- [x] Real parents with no visible children show a static right-pointing chevron
- [x] Ghost parents show a static down-pointing chevron, not clickable
- [x] Real parents with visible children have a clickable toggle chevron
- [x] Layout alignment preserved across all states
- [x] Progress counter and estimate badge still use full children count
- [x] Ghost row opacity at 65%
- [x] `make test` passes
