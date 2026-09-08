# Fix uneven hover/selection backgrounds in filter menu items

## What changed

`web/src/index.css` — two lines: the global scrollbar thumb color changed from `--color-border-default` to `--color-border-overlay` in both the webkit and standards-track scrollbar declarations.

## Why

In dark mode, `--color-border-default` (#282828) is nearly indistinguishable from `--color-surface-3` (#222222), the background used by menus and popovers. The 6px-wide scrollbar thumb was invisible, so when a sub-menu had enough items to scroll (e.g. Labels with 18+ items), the scrollbar gutter looked like dead space — making it appear that item hover backgrounds stopped short of the menu's right edge.

## Why this color

`--color-border-overlay` (#323232) is the existing token for popover/modal edges — 10 hex steps brighter than `border-default`. It's visible against dark `surface-3` backgrounds but still subtle on the lighter surfaces used by the main content area, sidebar, and diff viewer.

## What's deferred

The `scrollbar-gutter: stable` fix (reserving scrollbar space consistently so items never shift width) belongs in the Popover and Menu component extractions (xpo-5d1c38, xpo-de8d06). This issue is linked to both.

## Acceptance Criteria

- [x] Scrollbar thumbs are visible inside menus and popovers in dark mode — `--color-border-overlay` provides sufficient contrast
- [x] Scrollbar thumbs are not overly bright on non-menu surfaces — tested on backlog, diff viewer, sidebar
- [x] `make lint` passes
