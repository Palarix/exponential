# Walkthrough: Status icons in the Relations card are squished

## What was wrong
In the issue sidebar, each Relations row is a `flex items-center gap-2` container. It holds the kind label (`shrink-0`), a `StatusIcon`, and a `truncate` title span. `StatusIcon` renders a bare `<svg width={size}>`, and a flex item defaults to `flex-shrink: 1`. So when a title overflowed, flexbox took the overflow from the title *and* the SVG. The `width` attribute only sets the preferred size and does not stop the item from shrinking. Longer titles and longer kind labels ("dependency of") squeezed the icon more.

## The fix
The fix lives in the shared component, not at individual call sites. An icon should never shrink, so every caller should get this behaviour by default.

- `web/src/components/ui/status-icon-utils.ts` adds `statusIconClass(className?)`, which returns `cn('shrink-0', className)`. `cn` (tailwind-merge) keeps caller classes such as `inline-block align-[-1px] mx-0.5` in the move-children prompt.
- `web/src/components/ui/StatusIcon.tsx` computes `cls` once and puts it on the root `<svg>` of every status branch.

No call sites changed. The parent picker, Relations rows, add-relation picker, and the other views that put `StatusIcon` in a flex row now render it at full size. The title span is still shrinkable, so `truncate` still produces the ellipsis.

## Why a utils file
UI tests in this repo are pure `.ts` vitest tests of class-builder functions (see `icon-button-utils.ts` / `IconButton.test.ts`), not DOM renders. Moving the class logic into `status-icon-utils.ts` follows that pattern and makes the guarantee testable.

## Tests
`StatusIcon.test.ts` checks that `statusIconClass()` always includes `shrink-0` and merges a custom className. It was written first and failed before the helper existed. `make test` passes.

## Acceptance criteria
- [x] Relations card status icons render at a consistent 12×12 no matter how long the title or kind label is. Verified by the user in the UI ("lgtm").
- [x] Titles still truncate with an ellipsis. The title span is unchanged and stays shrinkable.
- [x] Other StatusIcon flex-row call sites are fixed by the central change (PropertySidebar L427/L660/L730 and the other views).
- [x] Unit test covers the non-shrinking icon (`StatusIcon.test.ts`, 2 tests passing).
- [x] `make test` passes (exit 0).
