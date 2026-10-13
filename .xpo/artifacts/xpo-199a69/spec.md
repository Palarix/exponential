# Attachment indicator lost its count

## What
The attachment pill on backlog rows and board cards shows the paperclip and the attachment count, like the branch, chat and sub-issue pills. The tooltip stays.

## Why
xpo-219de9 made the attachment pill icon-only, with the count in the tooltip. Next to pills that all show a number, the bare paperclip reads as inconsistent, and the count takes a hover to read. This reverses that one decision of xpo-219de9 ("The attachment pill shows only its icon").

## How
- `AttachmentIndicator` renders `{count}` after the icon, the same markup as `BranchBadge` (`</svg>N`). Both callers pick it up.
- `RowIndicators.test.tsx` asserts the count is visible instead of absent.

## Acceptance Criteria
- [ ] The attachment pill shows the paperclip and the count, on backlog rows and board cards.
- [ ] The tooltip still reads "1 attachment" / "N attachments".
- [ ] `RowIndicators.test.tsx` asserts the visible count.
- [ ] `make test` passes.
