# Spec: Status icons in the Relations card are squished

## What
In the Relations card, a `StatusIcon` sitting next to a long title gets squeezed below its set size. Flexbox shrinks the bare `<svg>` because it has the default `flex-shrink: 1`.

## Why
Icons should always render at the `size` they are given. Relations rows should look the same no matter how long the title is.

## How
- Fix it once in the shared component. `StatusIcon` always puts `shrink-0` on its root `<svg>` and merges the caller's `className` on top (via `cn`).
- Follow the existing `icon-button-utils.ts` pattern: add `status-icon-utils.ts` exporting `statusIconClass(className?)`, which returns `cn('shrink-0', className)`. Every branch of `StatusIcon` uses it.
- No call-site changes are needed. Every call site (PropertySidebar parent picker at L427, Relations at L660, add-relation picker at L730, the inline icon in the move-children prompt at L822, and the other views) gets the fix automatically. `inline-block align-[-1px]` at L822 does not conflict with `shrink-0`.

## Acceptance criteria
- [ ] Relations card status icons render at a consistent 12×12 no matter how long the title or kind label is
- [ ] Titles still truncate with an ellipsis (the title span keeps `truncate` and stays shrinkable)
- [ ] Other StatusIcon flex-row call sites are fixed by the central change
- [ ] Unit test: `statusIconClass()` includes `shrink-0` and merges a custom className
- [ ] `make test` passes
