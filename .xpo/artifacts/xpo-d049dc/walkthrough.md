# Walkthrough: Backlog DnD Performance

## What was built

Fixed sluggish drag-and-drop in the backlog view by eliminating React re-renders during the drag hot path, and fixed two related bugs discovered during tophat.

## Architecture

The fix has two structural layers:

### Layer 1: Memoized row components

Extracted the inline row JSX (~450 lines) into two `React.memo` components:

- **`BacklogIssueRow`** — the full issue row including tree guides, priority/status/label/estimate popovers, branch badges, avatars, and all interactive elements. ~300 lines.
- **`BacklogGroupHeader`** — the group section header with expand/collapse, count/points toggle, and inline create button. ~100 lines.

Each component only re-renders when its own props change. During a drag, with DnD visual state moved out of React (Layer 2), rows receive no prop changes and skip re-renders entirely.

Stable callbacks (`handleRowMouseEnter`, `handleOpenPopover`, `handleClosePopover`, `handleToggleStoryPoints`) were added to the parent so the memo boundary holds.

### Layer 2: Ref-based DnD visuals

The three DnD visual states — drop indicator line, group ring highlight, and nest target highlight — were converted from `useState` to `useRef` with direct DOM manipulation.

**How visuals are applied:**

- **Drop indicator**: `data-drop="above|below"` attribute + `--indicator-left` CSS custom property on the row wrapper (`[data-context-issue]`). CSS pseudo-elements in `backlog-dnd.css` render the line. This avoids a persistent floating element and works with the wrapper's existing `position: relative`.
- **Group highlight**: inline `boxShadow` + `background` styles on the `[data-group-status]` wrapper, plus `backgroundColor: transparent` on the `[data-group-header]` element so the wrapper's accent background shows through.
- **Nest target highlight**: inline `boxShadow` + `background` styles on the `[data-backlog-row]` element.

Three module-level helpers (`applyIndicatorDOM`, `applyGroupDOM`, `applyNestDOM`) handle the DOM reads/writes. Component-level wrappers (`setIndicator`, `setGroup`, `setNest`) snapshot the previous ref value, update the ref, and call the helper. The DnD handler callbacks were updated to call these wrappers instead of `setState` — the logic conditions are otherwise unchanged.

`activeId` stays in React state because it only changes on drag start/end and controls `canDrag`/`isDropTarget` props on rows.

## Bugs fixed during tophat

### Ghost row collision dead zones

Ghost rows (dimmed parent/child rows in foreign groups) were disabled as droppables, creating zones where `closestCenter` jumped to a distant real row. When a parent issue existed as both a ghost in Planned and a real row in In Progress, `rows.findIndex` returned the ghost (earlier in the array), making the handlers resolve the wrong group.

**Fix:**
1. Ghost rows are now enabled as droppables (collision detection participation)
2. `handleDndOver` detects `ghost:` DnD IDs and shows the group ring for the ghost's group
3. `handleDndMove` early-returns on `ghost:` IDs
4. All `rows.findIndex` calls filter out ghosts (`!r.isGhostParent && !r.isGhostChild`) when resolving non-ghost DnD IDs

### Post-cancel click

Pressing Escape during a drag cancelled the drag but the subsequent mouseup triggered `onClick`, opening the issue detail view. Fixed with a `wasDraggingRef` flag (set on drag start, cleared via `setTimeout(0)` on end/cancel) and an `onClickCapture` handler on the list container that suppresses clicks while the flag is set.

## Files changed

| File | Change |
|------|--------|
| `BacklogIssueRow.tsx` | New — memoized issue row component |
| `BacklogGroupHeader.tsx` | New — memoized group header component |
| `backlog-dnd.css` | New — drop indicator pseudo-element styles |
| `Backlog.tsx` | Ref-based DnD visuals, stable callbacks, ghost collision fix, click suppression, unused import cleanup |
