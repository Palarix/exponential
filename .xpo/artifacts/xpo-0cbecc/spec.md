# Ghost parents are collapsible in the Backlog

## What
In nested hierarchy mode, a ghost parent (a faded parent row injected into a status group so its real children appear nested) is always expanded and shows a static, non-clickable down chevron. Make ghost parents collapsible exactly like real parents: a clickable chevron that rotates, the same keyboard behaviour (→ / ←), and the same "Expand all" / "Collapse all" handling.

While a search query or filter is active, ignore the persisted collapsed state so matches are never hidden.

## Why
Users see a full chevron on these rows and expect to click it. Nothing happens, so it reads as "parents are no longer collapsible". "Collapse all" also skips ghost parents, so it never fully collapses the view. This reverses the "Ghost parent (always expanded)" decision from xpo-2dd9a8.

The "always expanded" behaviour had a hidden job: search and filters run before row building (`Backlog.tsx` `filteredIssues`), so a matching child whose parent doesn't match is shown under a **ghost parent**. If ghosts start honouring the persisted collapsed set, a previously collapsed epic would hide the very search hit the user is looking for. Hence the filter override.

## Current behaviour (root cause)
- `useBacklogRows.ts` `addTree`: children render when `isGhost || expandedNodes.has(issue.id)`, so ghosts ignore the expanded state.
- `nodeIndicator()` returns `"expanded"` for ghost parents → static chevron `<span>`, no click handler.
- `Backlog.tsx` drop-target logic treats `overRow.isGhostParent` as expanded.
- Keyboard →/← already key off `hasVisibleChildren` + `expandedNodes`, so they'll work once rows honour the state.

Regular (non-ghost) parents' toggling was checked and works: click, keyboard, and expand/collapse all.

## How
- `useBacklogRows`: render children only when `expandedNodes.has(issue.id)`, the same for ghost and real parents.
- `nodeIndicator()`: drop the `"expanded"` state. Any nested parent with visible children returns `"toggle"`, so `BacklogIssueRow` loses the static-chevron branch.
- `Backlog.tsx` drop logic: remove the `|| overRow.isGhostParent` special case.
- **Expanded state is shared per issue ID**: collapsing a ghost parent also collapses the real parent row (in its own status group), and vice versa. That's one persisted `exponential-backlog-nodes-collapsed` set, as today. Default stays expanded.
- **Filtering mode** (`search` non-empty or `hasActiveFilters(filters)`):
  - `expandedNodes` starts as *all* parents expanded; the persisted collapsed set is ignored.
  - Toggles (chevron, →/←, Expand/Collapse all) write to a transient in-memory collapsed set instead of localStorage, so the user can still tidy results.
  - The transient set resets whenever the query or filters change, and is discarded when filtering ends; the persisted state is then restored untouched.

## Decisions
- **Shared state per issue ID, not per row/group.** The user's intent is "hide epic X's children", not "hide this row". Per-ID state also survives status changes, which per-group state would not. The xpo-2dd9a8 "leak" was a ghost toggle changing a real parent elsewhere while the ghost itself stayed expanded; once ghosts honour the state, the behaviour is consistent.
- **Filtering never hides matches by default.** Matches standard filtered-tree behaviour (file explorers, outline views). Also fixes the smaller existing case of a real parent that matches, was collapsed, and hides a matching child.

## Out of scope
- A muted "N hidden" hint on collapsed ghost rows (group header count can exceed visible rows). Nice to have; file separately if wanted.

## Acceptance Criteria
- [ ] Clicking a ghost parent's chevron collapses/expands its children; the chevron rotates
- [ ] → / ← expand/collapse a focused ghost parent
- [ ] "Collapse all" / "Expand all" also affect ghost parents
- [ ] Ghost parents default to expanded
- [ ] Real parents' behaviour is unchanged (outside filtering mode)
- [ ] Drag-and-drop "below" a collapsed ghost parent doesn't target its hidden children
- [ ] With epic X collapsed, searching for a child of X shows that child (ghost parent X rendered expanded)
- [ ] Same for an active filter (e.g. label filter matching only X's child)
- [ ] Toggling while filtering does not change `exponential-backlog-nodes-collapsed`
- [ ] Clearing the search/filters restores the previously persisted collapsed state
- [ ] Changing the query resets transient toggles (everything expanded again)
- [ ] Unit tests: `nodeIndicator` returns `toggle` for ghost parents; row building hides ghost-parent children when collapsed; filtering mode ignores the persisted collapsed set
- [ ] `make test` passes
