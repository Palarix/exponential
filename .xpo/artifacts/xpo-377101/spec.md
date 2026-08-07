# Spec: Ghost parent row duplicates popover

## Problem

In Active/Backlog views, an issue can appear twice: as a real row in its own status group and as a ghost parent in a child's status group. The popover state is keyed by `issueId`, so clicking a popover (status, priority, labels, estimate) on either row opens it on both — both rows match `openPopover?.issueId === issue.id`.

## Solution

Change the popover key from `issueId` to `rowIndex`. Each row has a unique index in the `rows` array, even when the same issue appears multiple times. This ensures only the clicked row's popover opens.

### Changes in `Backlog.tsx`

1. Change state type: `{ issueId: string, type: ... }` → `{ rowIndex: number, type: ... }`
2. Update all popover toggle clicks: use row index `i` instead of `issue.id`
3. Update all popover visibility checks: `openPopover?.rowIndex === i` instead of `openPopover?.issueId === issue.id`
4. Update keyboard handler: use `focusedIndexRef.current` instead of `row.issue.id`

## Acceptance criteria

- [ ] Clicking a popover on a ghost parent does not open the same popover on the real row (and vice versa)
- [ ] Keyboard shortcuts (s, l, e, p) still open popovers on the focused row
- [ ] Popovers still close correctly via Escape, click-outside, and selection
