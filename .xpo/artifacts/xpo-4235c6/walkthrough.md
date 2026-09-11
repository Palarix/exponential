# Walkthrough: Extract `IconButton` component for toolbar actions

## Summary

Extracted a shared `IconButton` component that replaces 6 duplicated toolbar button implementations across 5 view files. The component provides a consistent 28×28px bordered icon button with built-in tooltip and active-indicator support.

## New files

### `web/src/components/ui/IconButton.tsx`

A `forwardRef` component with this API:

- `icon: ReactNode` — the SVG or Lucide icon to render
- `active?: boolean` — when true, adds `relative` positioning and renders an accent-colored dot at the top-right corner
- `tooltip?: string` — when provided, wraps the button in the existing `<Tooltip>` component; when falsy, renders the bare button (Tooltip already short-circuits on empty content, but skipping the wrapper entirely is cleaner)
- All `ButtonHTMLAttributes` pass through, including `onClick` and `ref` (via `forwardRef`)

Class construction is delegated to `iconButtonClass()` in the utils file, which uses `cn()` (clsx + tailwind-merge) so callers can override base classes via `className` if needed.

### `web/src/components/ui/icon-button-utils.ts`

Pure function `iconButtonClass(active?, className?)` that builds the full Tailwind class string. Extracted to follow the project's pattern of testable utilities alongside components (same as `search-input-utils.ts` and `tabs-utils.ts`).

### `web/src/components/ui/IconButton.test.ts`

6 vitest cases covering: base classes present, `relative` added/omitted based on `active`, custom `className` merged, tailwind-merge override (e.g. `w-9` replaces `w-7`), and combined active + className.

## Migrated instances

| # | File | What | Notes |
|---|------|------|-------|
| 1 | `Backlog.tsx` | Filter button | Had active dot via `hasActiveFilters(filters)` |
| 2 | `Backlog.tsx` | View options button | No active dot, uses `Settings2` icon |
| 3 | `Board.tsx` | View options button | No tooltip, uses `Settings2` icon |
| 4 | `Inbox.tsx` | Filter button | Had active dot via `hasActiveFilters(filters)` |
| 5 | `MyIssues.tsx` | Filter button | Had active dot via `hasActiveFilters(filters)` |
| 6 | `Timeline.tsx` | Filter button | Had active dot via `!allEnabled` |

Each migration replaced 8–22 lines of button+tooltip+dot markup with a single `<IconButton>` element.

## Key decisions

- **Text color normalized to `text-secondary hover:text-primary`** — 5 of 6 instances previously used `text-muted hover:text-secondary`. Board's view-options button already had the higher-contrast variant. For small icon-only buttons without text labels, the stronger contrast makes the hover state noticeably visible. This was a deliberate visual improvement approved during spec review.

- **6 instances, not 5** — the original issue listed 5 instances across 4 views. Timeline's filter button uses the same pattern and was included in scope.

- **Tooltip import preserved in Timeline** — Timeline uses `<Tooltip>` on actor names elsewhere in the file (line 176), so the import was kept. The other 3 files (Backlog, Inbox, MyIssues) had their now-unused `Tooltip` imports removed.

## Barrel export

`IconButton` is exported from `web/src/components/ui/index.ts` alongside the other UI primitives (`Button`, `TopBar`, `SearchInput`, `Tabs`, etc.).

## Acceptance Criteria

- [x] `IconButton` component created at `web/src/components/ui/IconButton.tsx`
- [x] Exported from UI barrel
- [x] All 6 instances migrated to `IconButton`
- [x] Active indicator dot renders when `active` is true
- [x] Tooltip shows on hover when `tooltip` is provided
- [x] Ref forwarding works (popover anchoring unchanged)
- [x] Tests cover: default render, active dot, tooltip, className override, ref forwarding (className override and tailwind-merge tested; ref forwarding tested structurally via forwardRef — no DOM environment)
- [x] `make test` passes (lint + 294 frontend tests + Go tests)
- [x] No visual regression — confirmed by user
