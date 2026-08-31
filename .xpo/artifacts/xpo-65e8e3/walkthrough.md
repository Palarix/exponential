# Walkthrough: Add "Done" view tab to the backlog

## What changed

Added a fourth tab "Done" to the backlog tab bar. Tab order is now: All Issues → Backlog → Active → Done.

## Files modified

- **`web/src/components/Backlog/Backlog.tsx`** — extended the `Tab` type union with `"done"`, added `done` entry to `TAB_CONFIGS`, reordered tabs, and adjusted the default group expansion logic so the DONE status group starts expanded on the Done tab (it was previously always collapsed by default since it's a completed-work section in "All Issues").

- **`web/src/components/Backlog/useBacklogRows.ts`** — added `done` entry to the parallel `TAB_CONFIGS` and reordered to match.

## How it works

The backlog's tab system is data-driven via `TAB_CONFIGS` — each tab maps to a set of visible statuses. Adding a tab is just adding an entry. Everything else flows from existing logic:

- **Sort**: the DONE status group already forces `updated` sort (line 125 in `useBacklogRows.ts`), so the Done tab automatically sorts by most recently completed.
- **Hierarchy**: the `hierarchyMode` state is persisted per tab via `exponential-backlog-hierarchy-${activeTab}` in localStorage, so the Done tab gets its own independent toggle.
- **Ghost parents**: nested mode already renders ghost parents for children whose parent is in a different status group — this works unchanged on the Done tab.
- **Expand/collapse default**: the only adjustment needed was in the `useEffect` that initializes `expandedGroups` when switching tabs — it previously excluded DONE from the default expanded set. Now it checks `activeTab === "done"` to expand it when that's the active view.