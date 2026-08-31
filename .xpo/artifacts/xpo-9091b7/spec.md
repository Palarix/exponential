# Spec: Hierarchy toggle for backlog views with consistent DnD

## What

Add a per-view "Show hierarchy" toggle to the backlog toolbar that switches between nested and flat rendering. Each mode has a single consistent data model — eliminating the current hybrid (nested + breadcrumbed) that breaks drag-and-drop.

## Why

The current backlog mixes two presentation models in the same list:
- Children whose parent is in the same status group render **nested** (depth > 0)
- Children whose parent is in a different status group render **breadcrumbed** (depth 0 with an italic parent title)

This means `depth === 0` can be either "real top-level" or "orphaned child", and every DnD function must branch on that ambiguity. Four bugs stem directly from it:
1. `isSiblingPosition` rejects breadcrumbed children as drop targets
2. `performDrop` routes breadcrumbed children into the top-level sort path
3. Ghost parents are appended unsorted and non-draggable
4. Sibling sort computation includes invisible cross-status siblings

Patching these individually is fragile — the next nesting feature will hit the same ambiguity. The toggle eliminates the hybrid entirely: nested mode is always a tree, flat mode is always a flat list.

## How

### Toggle state

- Per-view, per-project preference stored in `localStorage`
- Key format: `xpo:<project-prefix>:<view>:hierarchy` where view is `all`, `active`, `backlog`, etc.
- Values: `nested` (default) or `flat`
- Toggle renders as an icon button in the backlog toolbar (tree icon / list icon), with tooltip "Show hierarchy" / "Show flat list"

### Flat mode

Everything at depth 0. No nesting, no ghost rows.

**Row building (`useBacklogRows.ts`):**
- All issues in the group rendered as a single sorted flat list
- Children display a breadcrumb (parent title) as secondary text
- No `addTree` recursion — just iterate the sorted list
- Ghost parents not created (not needed — no nesting to contextualize)

**DnD (`Backlog.tsx`):**
- `isSiblingPosition`: all rows are siblings (same depth, same group) — trivially true for any row in the same status group
- `performDrop`: pure flat reordering — compute `sort_order` between the two adjacent rows in the flat list, update the dragged issue
- `canDrag`: all rows draggable (no ghost exclusion needed)
- Alt-drag reparenting: **disabled** — flat mode is for reordering only; switch to nested mode to manage hierarchy
- Cmd/Ctrl cross-status drag: still works (changes status, computes sort_order in target group's flat list)

### Nested mode

Full parent-child tree with ghost rows for cross-status context.

**Row building (`useBacklogRows.ts`):**

*"All Issues" tab:*
- Build the full tree: every issue with a `parent_id` nests under its parent, regardless of status
- No breadcrumbed children — all children are nested
- Parents without children render as leaf rows

*Status-filtered tabs (Active, Backlog, etc.):*
- Top-level: issues with no `parent_id` that match the status filter
- Ghost parents: issues whose `parent_id` target is in a different status group — the parent is pulled in as a ghost row (dimmed, marked `isGhost: true`). Ghost parents are **interleaved** with top-level issues by `sort_order` (not appended at the end)
- Ghost children: children of a visible parent whose status places them in a different group — rendered nested under the parent as ghost rows (dimmed, non-interactive). This gives context for what else is under an epic without navigating away
- Ghost rows carry a flag (`isGhost: true`) on the row object for styling and DnD exclusion

**DnD (`Backlog.tsx`):**

- `canDrag`: `isDndEnabled && !row.isGhost` — ghost rows excluded from dragging
- `isSiblingPosition`: check `parent_id` match (or both top-level). Since there are no breadcrumbed children, depth reliably indicates nesting. Ghost rows return `false` as drop targets
- `performDrop`: sibling set is `issues.filter(i => i.parent_id === targetParentId && i.status === status)` — filtered to same status to exclude ghost children from sort computation. For ghost parents being reordered: update `sort_order` only, never change status
- Alt-drag reparenting: works as today — sets `parent_id` on the dragged issue
- Drop indicator: skip ghost rows when computing indicator position

**Ghost row styling:**
- Reduced opacity (0.5) or muted text color
- No drag handle shown
- Click still navigates to the issue detail (ghosts are read-only in the list, not invisible)
- Ghost parents auto-expand to show their children (both real and ghost)

### Migration from current behavior

The current ghost-parent logic in status-filtered views is a subset of what nested mode needs — extend it with:
1. Sort-order interleaving (ghost parents sorted among top-level by `sort_order`)
2. Ghost children (new — children of visible parents that are in a different status)
3. "All Issues" always nests (remove the breadcrumb fallback path)

The current breadcrumb rendering in "All Issues" becomes flat mode's only job — move it there and remove it from the nested code path.

## Acceptance Criteria

- [ ] Toggle visible in backlog toolbar for each view (All Issues, Active, Backlog, and any custom status groups)
- [ ] Toggle state persists per view, per project in localStorage
- [ ] Default is nested mode

**Flat mode:**
- [ ] All issues render at depth 0 with breadcrumb for children
- [ ] DnD reorders within the flat list; sort_order persists on refresh
- [ ] Alt-drag nesting disabled; tooltip or visual hint communicates "switch to hierarchy view"
- [ ] Cmd/Ctrl cross-status drag works

**Nested mode:**
- [ ] "All Issues" nests all children under parents (no breadcrumbs)
- [ ] Status-filtered views show ghost parents interleaved by sort_order
- [ ] Status-filtered views show ghost children under visible parents
- [ ] Ghost rows are dimmed, non-draggable, clickable
- [ ] DnD reorders real rows among real siblings; ghost rows excluded from sort computation
- [ ] Alt-drag reparenting works
- [ ] Ghost parents can be dragged to reorder among top-level issues (sort_order only, no status change)

**Regressions:**
- [ ] Existing DnD behavior preserved in nested mode (reorder children, reorder top-level, cross-status drag)
- [ ] Board view unaffected
- [ ] Issue detail view unaffected

## Edge Cases

- Ghost parent with no real children in current group (all children are in other statuses): renders as a ghost with only ghost children — still sortable among top-level
- Issue is both a parent and a child (nested epics): renders at its depth in the tree; ghost logic applies recursively
- Empty state: toggle hidden when no issues exist
- View with zero top-level issues (all children of a single epic): nested mode shows the epic with all children; flat mode shows breadcrumbed children only
