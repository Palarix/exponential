# Walkthrough: Hide expand chevron on parents with no visible children

## Problem

The backlog's nested mode had two chevron issues:

1. Parents whose children were all filtered out (e.g., in a tab where only certain statuses show) displayed a clickable expand chevron that expanded to nothing.
2. Ghost parents had a clickable chevron that, when clicked, toggled the *real* parent's expand/collapse state in another status group — a confusing cross-group side effect.

## What changed

### New `hasVisibleChildren` field on `RowItem` (`useBacklogRows.ts`)

The existing `hasChildren` field reflects whether an issue has *any* children across all statuses. It can't be repurposed because the progress counter (`3/5`) and estimate badge both need the full count. A new `hasVisibleChildren` field tracks whether children actually render in the current status group after ghost and status filtering.

The key change: `allVisual` (the filtered/sorted list of children that will actually render) is now computed *before* the parent row is pushed into the result array, so the parent row can carry `hasVisibleChildren: allVisual.length > 0`. Previously `allVisual` was computed after the push.

### Three-state chevron logic (`BacklogIssueRow.tsx`)

The chevron column now has three branches instead of two:

- **Clickable chevron** — real (non-ghost) parent with visible children. Rotates between right/down on toggle. This is the only interactive state.
- **Static chevron** — ghost parents (always points down, since they're always expanded) and empty parents (points right). Same SVG, same dimensions, wrapped in a `<span>` instead of a `<button>`.
- **Spacer** — non-parents. Preserves horizontal alignment.

### Keyboard and DnD consistency (`Backlog.tsx`)

`ArrowRight`/`ArrowLeft` keyboard shortcuts and the DnD drop-target logic both referenced `hasChildren` to decide whether to expand/collapse or nest under a parent. These now use `hasVisibleChildren` to stay consistent with the visual state.

### Ghost opacity bump

Ghost rows moved from `opacity-50` to `opacity-65` for better readability while still being visually distinct from real rows.
