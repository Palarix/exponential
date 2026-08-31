# Walkthrough: Hierarchy toggle for backlog views with consistent DnD

## Problem

Backlog DnD was broken because the rendering mixed nested children (depth > 0 under visible parents) with breadcrumbed children (depth 0 with parent in a different status group). Every DnD function had to branch on whether depth 0 meant "real top-level" or "orphaned child", causing four interrelated bugs: breadcrumbed children not draggable, wrong peer set in performDrop, ghost parents unsortable and appended at end, and cross-status sibling interference.

## Solution

A per-view "Show hierarchy" toggle that switches between two internally consistent modes. Each mode has one data model and one DnD code path — the hybrid is eliminated.

## What Changed

### `web/src/components/Backlog/useBacklogRows.ts`

**Flat mode:** All issues at depth 0, grouped by parent. Top-level issues appear in sort order, each followed by its children (sorted by `sort_order`). Orphan child groups (parent in a different status) are clustered by the parent's sort position. Children display a breadcrumb (`Parent Title >`). No ghost rows, no tree recursion.

**Nested mode:** One code path for all tabs (no `isAllTab` branch). For each status group: top-level issues plus ghost parents (interleaved by `sort_order`, not appended). Real parents show all children — real children plus ghost children (different status) for context. Ghost parents only show their real children to avoid duplication. All children (real + ghost) sorted together by `sort_order`.

Added `HierarchyMode` type export and `hierarchyMode` parameter. Added `isGhostChild` to `RowItem`.

### `web/src/components/Backlog/Backlog.tsx`

**Toggle state:** `hierarchyMode` stored in localStorage keyed `exponential-backlog-hierarchy-<tab>`, default `nested`. Synced on tab change.

**View Options dropdown:** Consolidates layout (Flat/Nested with `ListTree` icon), expand/collapse all (nested only), and sort options behind a single `Settings2` icon button. Filter button is now icon-only with tooltip. Both toolbar buttons use `bg-[var(--color-surface-1)]` instead of ghost styling.

**DnD changes:**
- Ghost rows get `id="ghost:<issue-id>"` and `enabled=false` for DnD hooks — avoids duplicate ID registration with dnd-kit
- `canDrag`: blocks ghost parents and ghost children
- `isSiblingPosition`: recognizes depth-0 siblings sharing `parent_id` (flat mode); top-level excludes rows with `parent_id`
- `performDrop` depth-0 branch: detects shared-parent siblings and sorts among them; Alt-drop between children of a different parent reparents (`adoptParent`); Alt-drop on top-level unparents
- Nest-target drop (Alt on parent row): appends at end of new parent's children + changes status to target group
- Group-header drop for children: only changes status, preserves `sort_order`
- Group-header and indicator drops for parents with children: shows cascade "Update sub-issues?" modal with `pendingUpdate` carrying `sort_order` alongside status
- `applyStatusChange` accepts optional `extraFields` to apply pending sort_order from DnD drops
- All issue rows have `select-none` to prevent text selection on ghost drag attempts
- Expand/collapse chevron hidden on parent rows in flat mode

### `web/src/components/Backlog/DndComponents.tsx`

Removed unnecessary inline `transform: translate3d(0,0,0)` from `DragOverlayCard` and unused `CSS` import from `@dnd-kit/utilities`.

### `design_docs/behaviour-invariants-dnd.md`

New design doc specifying expected DnD outcomes for every dragged-issue x drop-target combination in both modes, including modifier keys, ghost handling, cascade prompts, and sort order computation.

## Key Decisions

- **One nested code path for all tabs** rather than separate `isAllTab` logic — status-filtered views and "All Issues" use identical ghost parent/child rendering, just with different visible statuses.
- **Ghost children only under real parents** — showing ghost children under ghost parents would duplicate rows (same child visible as ghost in one group and real in another, both under ghost/real versions of the same parent).
- **Ghost DnD IDs prefixed with `ghost:`** — simplest fix for dnd-kit duplicate ID registration; ghost IDs don't match issue lookups so hover/drop events on ghosts are naturally ignored.
- **Child group-header drops preserve sort_order** — changing a child's status shouldn't scramble its position among siblings.
- **Flat mode groups children by parent** — prevents interleaving of children from different parents, which is confusing when parents have different sort positions.
