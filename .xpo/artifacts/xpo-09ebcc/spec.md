## What

Extract a shared `TopBar` component (`web/src/components/ui/TopBar.tsx`) that replaces the duplicated top-bar markup across 13 views. The component provides a consistent shell (height, padding, background, border) with explicit left/center/right slots so views declare *what* goes in each position without managing the layout themselves.

## Why

Each view currently builds its own top bar div with slightly different styles and layout patterns (`ml-auto`, `flex-1 flex justify-center`, etc.). Any global change (like scrollbar-gutter padding) requires editing every view. A single component with slots makes the bar style and layout a one-place change.

Parent epic: xpo-f539b6 (UI Polish)

## Acceptance Criteria

- [x] A `TopBar` component exists at `web/src/components/ui/TopBar.tsx`
- [x] All 13 views with top bars (including Cycles) use `TopBar` instead of inline markup
- [x] Default styling: `h-12`, `pl-5 pr-3`, `border-b border-[var(--color-border-subtle)]`, `shrink-0`
- [x] Three layout slots: `left`, `center`, `right` — each accepts `ReactNode`
- [x] Views that need style overrides can do so via `className`
- [x] No visual regression — each view looks identical before and after (except intentional polish changes)
- [x] `make test` passes (lint + tests)

## Decisions

1. **`h-12` default height** — Standardizes all views to `h-12` for consistency.
2. **`pl-5 pr-3` default padding** — Asymmetric padding gives buttons on the right a tighter edge.
3. **Explicit left/center/right slots** — Unifies layout. No slot wrapper divs — callers provide their own flex container with their preferred gap.
4. **Center slot uses flex centering** — `flex-1 flex justify-center`, not absolute positioning. Avoids overlap risk with left/right content.
5. **`cn()` utility added** — `clsx` + `tailwind-merge` enables safe class overrides. New file `web/src/utils/cn.ts`.
6. **Labels normalized** from `h-13 px-6` to standard defaults.
7. **Cycles gets a proper top bar** — Inline heading promoted to fixed TopBar.
8. **Backlog search always visible** — Moved from conditional right-side to permanent center slot, matching Dependencies style. `searchFocused`/`onSearchBlur`/`onSearchFocus` props removed.
9. **Icon-only filter/view buttons normalized** — Backlog, MyIssues, Inbox filter buttons changed to `w-7 h-7` bordered icon-only buttons with Tooltip, matching Board.
10. **Timeline kind filter converted to tabs** — Segmented control moved to left as underlined tabs (filed xpo-eb76cd to replace with multi-select filter menu).
11. **Search keyboard hints** — Both Backlog and Dependencies search inputs show `/` hint when empty, `Esc` when populated.
12. **Dependencies Escape handling** — Added `onKeyDown` for Escape to clear search and blur.
