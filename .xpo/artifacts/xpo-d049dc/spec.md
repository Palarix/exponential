# Spec: Backlog DnD Performance

## What

Drag and drop on the backlog view is sluggish — the drag overlay and drop indicators lag behind the cursor.

## Why

`handleDndMove` fires on every pointer move during a drag. Each invocation calls up to three `setState` calls (`setDropIndicator`, `setDropGroupStatus`, `setDropNestTargetId`). Each state change triggers a **full re-render** of the entire Backlog component (~2500 lines), which:

1. Re-runs the sections IIFE (lines 1719–2312) — iterates all rows to build JSX
2. Re-renders every issue row inline (no `React.memo` boundary)
3. Re-runs `useDraggable`/`useDroppable` hooks for every row — DnD context re-registration
4. Re-evaluates every row's className (including `isDraggedOrBatch`, `isNestTarget`, etc.)

The drop indicator, group highlight, and nest target highlight are purely visual decorations that change at pointer-move frequency but only affect 1–2 DOM elements at a time.

## How

Two-part fix: extract rows into memoized components (structural), and move DnD visual state to refs with direct DOM manipulation (hot-path).

### Part 1: Extract memoized row components

**`BacklogGroupHeader`** — `React.memo` component for the group header row.

**`BacklogIssueRow`** — `React.memo` component for an issue row. Key: **no DnD visual state in props**. Drop indicator, group highlight, and nest target are handled by Part 2.

### Part 2: Ref-based DnD visuals

Replace the three DnD visual states with refs:

```
dropIndicatorRef    (was: dropIndicator useState)
dropGroupStatusRef  (was: dropGroupStatus useState)
dropNestTargetIdRef (was: dropNestTargetId useState)
```

Three module-level DOM helpers (`applyIndicatorDOM`, `applyGroupDOM`, `applyNestDOM`) apply visuals directly:

- **Drop indicator**: CSS pseudo-elements via `data-drop="above|below"` attribute + `--indicator-left` custom property on the row wrapper (`backlog-dnd.css`)
- **Group highlight**: inline `boxShadow`/`background` styles on `[data-group-status]` wrapper + transparent header override
- **Nest target highlight**: inline `boxShadow`/`background` styles on `[data-backlog-row]` element

Wrapper callbacks (`setIndicator`, `setGroup`, `setNest`) read the previous ref value, update the ref, and call the DOM helper.

### Keep `activeId` in React state

`activeId` only changes on drag start/end (not during move), and gates `canDrag`/`isDropTarget` on rows. This is appropriate as React state.

### Memoize the DragOverlay issue lookup

Replace `issues.find(i => i.id === activeId)` inside the render with a `useMemo` keyed on `[activeId, issues]`.

### Ghost row collision fix

Ghost rows (dimmed parent/child rows appearing in a group different from their actual status) were disabled as droppables, creating collision detection dead zones. This caused `closestCenter` to match distant real rows in the wrong group.

Fix:
1. Enable ghost rows as droppables (participate in collision detection)
2. `handleDndOver` detects `ghost:` DnD IDs and shows the group ring for the ghost's group
3. `handleDndMove` early-returns on `ghost:` IDs
4. All `rows.findIndex` lookups that resolve a non-ghost DnD ID filter out ghost rows (`!r.isGhostParent && !r.isGhostChild`) to prevent matching a ghost instance of the same issue before the real row

### Post-drag click suppression

Cancelling a drag (Escape) while holding the mouse button caused a spurious click on mouseup, opening the issue detail view. Fix: `wasDraggingRef` flag set on drag start, cleared with `setTimeout(0)` on drag end/cancel, checked by `onClickCapture` on the list container.

## Acceptance Criteria

- [x] Drag overlay tracks cursor smoothly at 60fps with no perceptible lag
- [x] Drop indicator line, group highlight, and nest target highlight appear/disappear correctly
- [x] All existing DnD behaviors preserved: same-group reorder, Cmd/Ctrl cross-group, Alt nest/reparent, batch drag
- [x] Row components are `React.memo` wrapped — no unnecessary re-renders during drag
- [x] Cross-group drag over ghost parent areas shows the correct group ring
- [x] Escape-cancel does not trigger issue detail view on mouseup
- [x] `make test` passes (lint + tests)
