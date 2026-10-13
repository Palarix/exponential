# Walkthrough: attachment indicator shows its count again

## What changed
`AttachmentIndicator` (`web/src/components/ui/AttachmentIndicator.tsx`) renders `{count}` after the paperclip icon. Before, it showed only the icon and kept the count in its `title`. Backlog rows (`BacklogIssueRow.tsx`) and board cards (`BoardCard.tsx`) both use this component, so both show the count now. Neither caller changed.

## Why
xpo-219de9 decided that the attachment pill shows only its icon, while the branch, chat and sub-issue pills show numbers. In use, the bare paperclip looked out of place next to the other pills, and you had to hover to read the count. The user reversed that decision. The spec for this issue is now the current rule: every row indicator pill shows its count.

## How it fits
The markup matches `BranchBadge`: the icon and the number sit side by side in the shared `indicatorPill` span. Its `gap-1` and `tabular-nums` handle spacing and digit width, so no styling changed. The tooltip still reads "1 attachment" or "N attachments".

## Tests
`RowIndicators.test.tsx` used to assert that no number follows the icon (`not.toMatch(/<\/svg>\d/)`). It now asserts `</svg>2</span>`. I changed the test first and ran it to see it fail, then changed the component.

## Acceptance Criteria
- [x] The attachment pill shows the paperclip and the count, on backlog rows and board cards. Both render the shared `AttachmentIndicator`, which now includes the count. The user checked it on the board.
- [x] The tooltip still reads "1 attachment" / "N attachments". The existing `title` assertions still pass.
- [x] `RowIndicators.test.tsx` asserts the visible count (`</svg>2</span>`).
- [x] `make test` passes (exit 0: lint, 832 frontend tests, Go tests).
