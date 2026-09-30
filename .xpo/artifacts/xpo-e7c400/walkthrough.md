# Walkthrough: blank slot for parents with no visible children

## The problem
In nested hierarchy mode, each Backlog row has a 16px slot left of the status icon for the tree toggle. `useBacklogRows` gives each row two flags:

- `hasChildren`: the issue has sub-issues at all.
- `hasVisibleChildren`: at least one sub-issue would render under it in this view.

They differ when every sub-issue is filtered out of the view:
- all sub-issues are terminal and ghosts are hidden;
- all sub-issues sit in another status group on the same tab, where they already render under a ghost parent;
- ghost-child rows, which always get `hasVisibleChildren: false`.

`BacklogIssueRow` treated "has children but none visible" the same as a ghost parent: it rendered a static `<span>` holding the chevron, rotated only when `hasVisibleChildren` was true. So these rows showed an unrotated chevron that looked collapsible but did nothing when clicked. That's what irritated users.

## The fix
The slot decision now lives in a pure function, `nodeIndicator()` in `web/src/components/Backlog/backlog-row-utils.ts`. It returns one of three states:

| state | when | renders |
|---|---|---|
| `toggle` | nested, has visible children, not a ghost parent | clickable chevron (unchanged) |
| `expanded` | nested, ghost parent with visible children | static rotated chevron (unchanged) |
| `none` | everything else | blank `w-4` spacer |

`BacklogIssueRow` computes `indicator` once and switches on it. Previously a nested ternary mixed the ghost and no-visible-children cases inside one branch. The unrotated static chevron is gone: it was only ever reachable in the broken case.

Pulling the decision into a pure function makes it testable without rendering the row (the project's frontend tests are pure vitest, no DOM). It also names the states, so a future change to a slot can't quietly break another.

## Decisions
- **Blank, not an icon.** The first version showed lucide `SquareDashedText` with the tooltip "No sub-issues in this view". On review the user preferred a blank slot. The row still shows `done/total` sub-progress, so it still reads as a parent.
- **Issues with zero sub-issues are unchanged.** They already showed the spacer, and they were never part of the complaint.
- **Alignment.** The chevron wrapper is `w-6 -m-1` (24px − 2×4px margin = 16px), the same as the `w-4` spacer, so switching between them doesn't shift the row.

## Acceptance Criteria
- [x] Nested mode: a parent whose children are all hidden from the view shows a blank slot, not a chevron. `nodeIndicator` returns `none` when `hasVisibleChildren` is false (tests "returns none for a parent whose children are all hidden" / "…ghost parent with no visible children"); confirmed by the user in the app.
- [x] Clickable chevron for parents with visible children is unchanged: the `toggle` branch markup is identical to before (test "returns toggle…").
- [x] Ghost parents with visible children still show the static expanded chevron (test "returns expanded…").
- [x] Row alignment unchanged: the blank slot reuses the existing `w-4` spacer, which is the same width as the chevron slot.
- [x] Unit tests cover all `nodeIndicator` outcomes: 6 cases in `backlog-row-utils.test.ts`, including flat mode and no children.
- [x] `make test` passes: exit 0, 375 frontend tests plus Go suite and lint.
