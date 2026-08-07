# Cap rendered issue count in Board and Backlog views

## Overview

At ~5k issues, rendering every issue as a DOM node causes ~1s lag on view switches. The fix is a render cap — filter/sort the full dataset in memory, but only mount the first N DOM nodes per group. A "+N more" button reveals additional batches.

## Board View (`BoardColumn.tsx`)

The DONE column already has a cap at 5 items with a "+N more" button. Generalize this:

- Introduce `COLUMN_VISIBLE_COUNT = 50` alongside the existing `DONE_VISIBLE_COUNT = 5`
- Change `shouldTruncate` to apply to all columns, using the appropriate cap per column
- Keep the existing `showAll` state, `visibleIds` slicing, and "+N more" button pattern

## Backlog View (`Backlog.tsx`)

No render cap exists. Add one per status group section:

- Cap: 100 issue rows per group
- State: a `Set<string>` of group statuses where the user has clicked "show all"
- In the sections rendering loop, slice `issueRows` to the first 100 unless the group is in the "show all" set
- Render a "+N more" button after the sliced rows
- Preserve original row indices (`index: i`) for keyboard navigation, DnD, and popover anchoring — the cap only affects which rows are mounted, not their index values

## Acceptance Criteria

- [ ] Board columns cap at 50 visible cards (DONE stays at 5)
- [ ] Backlog groups cap at 100 visible rows
- [ ] "+N more" button reveals all remaining items in the group
- [ ] Filtering, sorting, and searching still operate on the full dataset
- [ ] Keyboard navigation and DnD work correctly with the cap
- [ ] No regressions on small issue counts