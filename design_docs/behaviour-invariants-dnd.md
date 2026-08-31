# Backlog Drag-and-Drop: Behaviour Invariants

This document specifies the expected DnD outcomes for every dragged-issue x
drop-target combination, separately for **Nested** and **Flat** mode.

**Prerequisites:** DnD is only enabled when sort is set to **Manual**.

---

## Terminology

| Term | Meaning |
|------|---------|
| **Top-level issue** | No `parent_id` |
| **Child issue** | Has `parent_id` |
| **Ghost parent** | (Nested only) Parent rendered in a status group it doesn't belong to, providing context for its real children in that group. Dimmed, non-draggable. |
| **Ghost child** | (Nested only) Child rendered under a real parent in a status group the child doesn't belong to. Dimmed, non-draggable. |
| **Same group** | Dragged issue's status matches the target row's status group |
| **Cross group** | Dragged issue's status differs from the target row's status group |
| **Indicator** | The blue drop line between rows |
| **Nest highlight** | The ring highlight on a parent row (Alt-drag only) |

### Modifier keys

| Key | Effect |
|-----|--------|
| **None** | Reorder within same group only |
| **Cmd/Ctrl** | Allow cross-group drops (status change) |
| **Alt/Option** | Allow reparenting (nest under target / unparent / adopt into sibling group) |

---

## Nested Mode

### Row types

- **Real rows:** full opacity, draggable, valid drop targets
- **Ghost parents:** dimmed (50% opacity), non-draggable, not drop targets (use `ghost:` prefixed DnD ID)
- **Ghost children:** dimmed (50% opacity), non-draggable, not drop targets

### Scenarios

#### 1. Top-level issue dragged

| Drop target | Modifiers | Result |
|-------------|-----------|--------|
| Same-group top-level row | None | Reorder: `sort_order` computed between adjacent top-level rows |
| Same-group child row | None | Redirect indicator to after the parent's subtree |
| Cross-group header | Cmd/Ctrl | Status change + append at end of target group |
| Cross-group top-level row | Cmd/Ctrl | Status change + reorder at indicator position among target group's top-level |
| Top-level row (same group) | Alt | Nest highlight: set `parent_id` to target, append at end of target's children, change status to target's group |
| Cross-group top-level row | Alt | Nest highlight: set `parent_id`, append at end, change status |

#### 2. Top-level parent (has children) dragged

Same as (1) but with **cascade prompt**: if the parent has children in a
different status than the target, the "Update sub-issues?" modal appears before
applying the drop. User chooses "Just this issue" (parent only) or "Update all"
(parent + children). The pending `sort_order` and `status` are applied after
confirmation.

#### 3. Child issue dragged (nested under parent, depth > 0)

| Drop target | Modifiers | Result |
|-------------|-----------|--------|
| Sibling row (same parent) | None | Reorder among siblings: `sort_order` computed between adjacent siblings |
| Parent row (own parent) | None | Indicator redirects to first child position |
| Non-sibling row | None | No indicator (blocked) |
| Cross-group header | Cmd/Ctrl | Status change only, `sort_order` preserved (sibling position maintained) |
| Cross-group sibling | Cmd/Ctrl + Alt | Status change + reorder among siblings in target group |
| Row in another parent's subtree | Alt | Reparent: set `parent_id` to target's parent, sort at indicator position |

#### 4. Ghost rows

| Action | Result |
|--------|--------|
| Attempt to drag ghost parent | Blocked (`canDrag = false`) |
| Attempt to drag ghost child | Blocked (`canDrag = false`), no text selection (`select-none`) |
| Hover over ghost during drag | No indicator, no nest highlight (ghost DnD IDs don't match any issue lookup) |

---

## Flat Mode

### Row types

All rows at depth 0. Children display a breadcrumb (`Parent Title > Child Title`).
No ghost rows. No expand/collapse chevrons on parents.

Children are grouped by parent in the visual order: parent row, then its children
sorted by `sort_order`. Orphan child groups (parent in a different status) are
clustered by parent sort position.

### Scenarios

#### 1. Top-level issue dragged

| Drop target | Modifiers | Result |
|-------------|-----------|--------|
| Another top-level row (same group) | None | Reorder among top-level: `sort_order` between adjacent top-level rows |
| A child row (same group) | None | No indicator (not a sibling) |
| Cross-group header | Cmd/Ctrl | Status change + append at end |
| Another top-level row | Alt | Unparent has no effect (already top-level); otherwise same as no-modifier |

#### 2. Top-level parent (has children) dragged

Same as (1) but with **cascade prompt** when crossing status groups.

#### 3. Child issue dragged

| Drop target | Modifiers | Result |
|-------------|-----------|--------|
| Sibling row (same `parent_id`) | None | Reorder among siblings: `sort_order` between adjacent siblings of same parent |
| Own parent row | None | Indicator at "below parent" position |
| Non-sibling top-level row | None | No indicator (blocked — different parent or no parent) |
| Non-sibling child row | None | No indicator (blocked — different parent) |
| Cross-group header | Cmd/Ctrl | Status change only, `sort_order` preserved |
| Child of a different parent | Alt | Reparent: set `parent_id` to target's parent, sort at indicator position among new siblings |
| Top-level row | Alt | Unparent: clear `parent_id`, sort among top-level at indicator position |

#### 4. Parent row with children dragged within same group

Reorder among top-level rows. Children follow the parent in the visual list
(grouped by parent) but their `sort_order` is independent — only the parent's
`sort_order` changes.

---

## Cross-cutting behaviours (both modes)

### Status change on reparent

When Alt-dropping onto a row in a different status group (nest target or
indicator with Alt), the dragged issue's status changes to match the target
group. This applies to both the nest-highlight path and the indicator path.

### Cascade prompt

Any drop that changes a parent's status triggers the "Update sub-issues?" modal
if the parent has children in a different status (excluding DONE children).
The modal offers:

- **Abort** — cancel the drop entirely
- **Just this issue** — apply status + sort_order to parent only
- **Update all** — apply status to parent and all non-DONE children

### Sort order computation

Sort keys are computed using `fractional-indexing` (`generateKeyBetween`).
The key is generated between the two adjacent issues at the insert position.
This produces keys that sort lexicographically between neighbours without
rewriting other issues' keys.

### Drag batch

When dragging a parent in nested mode, same-status children are included in the
drag batch (dimmed alongside the parent). The batch is visual only — only the
dragged parent's fields are updated on drop.

### DnD ID uniqueness

Ghost rows use `ghost:<issue-id>` as their DnD ID to avoid collisions with
the real row for the same issue. Ghost IDs don't match any issue lookup in
`handleDndOver`/`handleDndMove`, so hovering over a ghost clears the drop state.
