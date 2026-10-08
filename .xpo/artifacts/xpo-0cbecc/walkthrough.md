# Walkthrough: Ghost parents are collapsible in the Backlog

## What was built and why

In nested hierarchy mode, a **ghost parent** is a faded parent row injected into a status group so that sub-issues living in that group still appear nested under their parent. xpo-2dd9a8 made ghosts "always expanded" with a static down chevron. Users saw a chevron, clicked it, and nothing happened, so it read as "parents are no longer collapsible". "Collapse all" also skipped ghosts.

Ghost parents now behave exactly like real parents. One subtlety drove a second change: search and filters run *before* row building, so a matching child whose parent doesn't match is rendered under a ghost parent. "Always expanded" had silently guaranteed search hits were visible. Once ghosts honour the persisted collapsed set, that guarantee would break, so filtering mode now ignores the persisted state.

## How the pieces fit together

### Row building: `useBacklogRows.ts`
- The body of the hook moved into a pure, exported `buildBacklogRows(...)`. `useBacklogRows` is now a `useMemo` wrapper around it. This exists purely so row building can be unit-tested without a DOM. Most of the diff is re-indentation.
- In `addTree`, children render when `allVisual.length > 0 && expandedNodes.has(issue.id)`. The `isGhost ||` short-circuit is gone.

### Indicator: `backlog-row-utils.ts` + `BacklogIssueRow.tsx`
- `nodeIndicator()` lost its `"expanded"` state; any nested parent with visible children returns `"toggle"`. `isGhostParent` stays in the argument type to document that ghost-ness no longer matters.
- `BacklogIssueRow` lost the static-chevron `<span>` branch.

### Expanded state: `expandedNodeIds()` + `Backlog.tsx`
- `expandedNodeIds({ parentIds, persistedCollapsed, transientCollapsed, filtering })` returns every parent minus whichever collapsed set applies: the persisted one normally, the transient one while filtering.
- `Backlog.tsx` derives `filtering = query !== "" || hasActiveFilters(filters)` and `filterKey = JSON.stringify([query, filters])`.
- `transientNodes` state is `{ key, collapsed }`. The effective transient set is `collapsed` only when `key === filterKey`, otherwise empty. That's how "reset when the query changes" happens: by comparison during render, with no `useEffect` and no extra render.
- `toggleNode`, `expandAllNodes` and `collapseAllNodes` branch on `filtering`. In filtering mode they write `transientNodes` and never touch `exponential-backlog-nodes-collapsed` in localStorage. Keyboard ←/→ and the "Expand all" / "Collapse all" menu go through the same callbacks via refs, so they inherit this automatically.
- Drag and drop: removed `|| overRow.isGhostParent` from the "drop below an expanded parent" check. It was dead code, because `overRow` is looked up with `!r.isGhostParent && !r.isGhostChild`; ghost rows are routed to the `ghost:` group-drop branch instead.

## Key decisions
- **Collapsed state is shared per issue ID.** Collapsing a ghost collapses the real parent in its own group, and vice versa. The user's intent is "hide epic X's children", not "hide this row", and per-ID state survives status changes. The "leak" xpo-2dd9a8 worried about was a ghost toggle changing a real parent elsewhere while the ghost itself stayed expanded; that inconsistency is gone now that ghosts honour the state.
- **Filtering never hides matches by default.** This matches common filtered-tree behaviour. It also fixes a smaller pre-existing case: a real parent that matches the search, was collapsed earlier, and hid a matching child.
- **Collapses while filtering are temporary.** The alternative (writing to the persisted set) would quietly change the user's normal view from inside a search.
- **Story points raised 2 → 3** for the filtering-mode work.

## Non-obvious things for future readers
- `filteredIssues` in `Backlog.tsx` is not memoized, so `useBacklogRows`' memo recomputes every render. That was true before this change too; `buildBacklogRows` now also rebuilds `childrenByParent` per call, which is O(n).
- Any new code path that toggles nodes must go through `toggleNode` / `expandAllNodes` / `collapseAllNodes`, or it will bypass filtering mode.

## Acceptance Criteria
- [x] Clicking a ghost parent's chevron collapses/expands its children; the chevron rotates. *Evidence: `nodeIndicator` returns `toggle` for ghosts and the row uses the same button as real parents; confirmed in manual testing.*
- [x] → / ← expand/collapse a focused ghost parent. *Evidence: the handlers key off `hasVisibleChildren` + `expandedNodes`, which ghost rows now honour; confirmed in manual testing.*
- [x] "Collapse all" / "Expand all" also affect ghost parents. *Evidence: they write the shared per-ID set that ghost rows now read.*
- [x] Ghost parents default to expanded. *Evidence: `expandedNodeIds` starts from all parent IDs; test "shows ghost-parent children when the parent is expanded".*
- [x] Real parents' behaviour is unchanged (outside filtering mode). *Evidence: same persisted-set path as before; existing tests pass.*
- [x] Drag-and-drop "below" a collapsed ghost parent doesn't target its hidden children. *Evidence: ghost rows never become `overRow`; the dead `isGhostParent` clause was removed.*
- [x] With epic X collapsed, searching for a child of X shows that child. *Evidence: `expandedNodeIds` test "ignores the persisted collapsed set while filtering" plus `buildBacklogRows` test "renders a search hit under an expanded ghost parent"; confirmed in manual testing.*
- [x] Same for an active filter. *Evidence: `filtering` includes `hasActiveFilters(filters)`.*
- [x] Toggling while filtering does not change `exponential-backlog-nodes-collapsed`. *Evidence: the filtering branches return before any localStorage write.*
- [x] Clearing the search/filters restores the previously persisted collapsed state. *Evidence: the persisted set is never modified in filtering mode; test "ignores the transient collapsed set when not filtering"; confirmed in manual testing.*
- [x] Changing the query resets transient toggles. *Evidence: the transient set applies only when `transientNodes.key === filterKey`.*
- [x] Unit tests: `nodeIndicator` returns `toggle` for ghost parents; row building hides ghost-parent children when collapsed; filtering mode ignores the persisted collapsed set. *Evidence: `backlog-row-utils.test.ts`, `useBacklogRows.test.ts`.*
- [x] `make test` passes. *Evidence: 385 frontend tests, Go tests and lint pass; `make frontend` builds.*
