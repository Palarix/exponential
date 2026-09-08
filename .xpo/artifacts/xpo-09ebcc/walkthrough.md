## Overview

Extracted a shared `TopBar` component that replaces 13 duplicated top-bar implementations with a single composable primitive. Along the way, several consistency issues surfaced during review and were fixed in-place.

## New files

### `web/src/utils/cn.ts`

A `cn()` utility combining `clsx` (conditional class joining) with `tailwind-merge` (conflict-aware Tailwind class merging). This was necessary because TopBar uses a `className` escape hatch for per-view overrides, and simple string concatenation can't resolve conflicting Tailwind utilities (e.g. `px-4` vs `pl-5 pr-3`). Dependencies: `clsx@2.1.1`, `tailwind-merge@3.6.0`.

### `web/src/components/ui/TopBar.tsx`

The component accepts three layout slots as props:

```tsx
interface TopBarProps {
  left?: ReactNode;    // titles, tabs, breadcrumbs
  center?: ReactNode;  // search (flex-centered between left/right)
  right?: ReactNode;   // actions, badges, filters
  className?: string;  // override defaults via tailwind-merge
}
```

**Defaults:** `h-12 pl-5 pr-3 border-b border-[var(--color-border-subtle)] shrink-0`

**Layout:** The three slots sit in a flex row. A `flex-1` spacer between left and right pushes them apart. When `center` is provided, it replaces the spacer with `flex-1 flex justify-center min-w-0 px-4`, centering the content in the available space between left and right. No wrapper divs on left/right — callers provide their own flex container with their preferred gap.

## Migrated views

All 13 views now use `<TopBar>` instead of inline top-bar markup:

| View | Notable changes |
|------|----------------|
| **Backlog** | Search moved from conditional right-side to always-visible center slot (Dependencies style). `searchFocused`/`onSearchBlur`/`onSearchFocus` props removed from component and App.tsx. Filter + view-options buttons normalized to `w-7 h-7` bordered icon-only style. Keyboard hint (`/` / `Esc`) added to search. |
| **Board** | Straightforward migration, no visual changes. |
| **Dashboard** | Straightforward migration, no visual changes. |
| **Timeline** | Kind filter (All/Issues/Commits) converted from segmented control on the right to underlined tabs on the left. Person filter stays on right. (Filed xpo-eb76cd to replace tabs with a multi-select filter menu.) |
| **Dependencies** | Search input gets `onKeyDown` for Escape (clear + blur). Keyboard hint added. `searchRef` refactored from callback ref to `useRef` + `useEffect`. |
| **DepGraph** | Straightforward migration. |
| **IssueDetail** | Removed `ml-4` from right section (spacer handles it now). |
| **MergeView** | Overrides via `className="bg-[var(--color-surface-1)] border-[var(--color-border-default)]"`. |
| **MyIssues** | Filter button changed to icon-only with Tooltip. Gap normalized to `gap-2`. |
| **Inbox** | `className="pl-4"` overrides default left padding. Filter button changed to icon-only with Tooltip. |
| **Labels** | Normalized from outlier `h-13 px-6` to standard defaults. |
| **PendingChanges** | Straightforward migration. |
| **PendingEventsPanel** | `className="px-4"` overrides default padding. |
| **Cycles** | Restructured from `overflow-y-auto` root with inline heading to `flex flex-col` with TopBar above a `flex-1 overflow-y-auto` scroll area. |

## Design rationale

**Why slots, not children?** Left/center/right is the dominant layout pattern across all views. Encoding it in the component means views declare *what* goes where without managing `ml-auto`, `flex-1`, or `justify-center` themselves. Each slot is a plain `ReactNode`, so the API is flexible — it standardizes positioning, not content.

**Why flex centering, not absolute?** Absolute centering (`position: absolute; left: 50%; transform: translateX(-50%)`) would give true visual center regardless of left/right widths, but risks overlapping left/right content when space is tight. Flex centering (`flex-1 flex justify-center`) is space-aware and matches the existing Dependencies search pattern.

**Why `cn()` / tailwind-merge?** Tailwind CSS determines cascade order by its own stylesheet generation order, not by class attribute order. Without tailwind-merge, `pl-5 px-4` won't resolve correctly — whichever class appears later in the generated stylesheet wins, regardless of order in the HTML. The `cn()` utility is 3KB and is the standard companion for Tailwind component libraries.

## Spin-off issues

- **xpo-eb76cd** — Replace Timeline tabs with event-type filter menu (multi-select checkboxes instead of exclusive tabs)
- **xpo-e1acfd** — Fix uneven hover/selection backgrounds in filter menu items
