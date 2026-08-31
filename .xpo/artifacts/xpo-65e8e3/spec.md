# Add "Done" view tab to the backlog

## What
Add a fourth tab "Done" to the backlog tab bar, alongside "All Issues", "Active", and "Backlog".

## Why
DONE issues are only visible in "All Issues" under a collapsed group. A dedicated tab makes it easy to browse completed work and audit history.

## How

1. Extend the `Tab` type: `"all" | "active" | "backlog" | "done"`
2. Add entry to `TAB_CONFIGS`: `done: { label: "Done", statuses: ["DONE"] }`
3. Default sort for the Done tab: `updated` (most recently completed first), matching the existing DONE group behaviour
4. Hierarchy mode persisted independently via localStorage key `exponential-backlog-hierarchy-done`
5. Same flat/nested toggle support as other tabs

## Acceptance Criteria
- [ ] "Done" tab appears after "Backlog" in the tab bar
- [ ] Shows only DONE issues
- [ ] Default sort is by `updated` (most recently completed first)
- [ ] Flat/nested toggle works and is persisted independently
- [ ] Ghost parents appear in nested mode for children whose parent is not DONE
- [ ] Tab selection persists across navigation (same as other tabs)