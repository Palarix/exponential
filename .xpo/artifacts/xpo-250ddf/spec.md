# Heading — Extract shared heading component

## What

Extract a `Heading` component from the repeated `text-sm font-medium text-[var(--color-text-primary)]` pattern used across the app for view titles, column headers, section titles, card titles, author names, and other structural text.

## Why

19 sites repeat the same className string with minor variations (leading, truncation, line-clamping). A single component eliminates the duplication and standardises the heading pattern.

## API

```tsx
interface HeadingProps {
  title: string;
  as?: 'span' | 'h1' | 'h2' | 'h3' | 'p';  // default 'span'
  subtitle?: ReactNode;
  leading?: 'snug';
  truncate?: boolean;
  clamp?: number;
}
```

- `as` controls the rendered element, defaulting to `span`.
- When `subtitle` is present, wraps in a flex container with `gap-3`.
- `leading` adds `leading-snug` for compact multi-line text.
- `truncate` adds `min-w-0 truncate` for overflow ellipsis.
- `clamp` adds `line-clamp-{n}` for multi-line truncation.

## Migration sites (19)

| # | File | Props used |
|---|------|-----------|
| 1 | `Board/Board.tsx` | `title="Board"` |
| 2 | `Cycles/Cycles.tsx:194` | `title="Cycles"` |
| 3 | `Labels/Labels.tsx` | `title="Labels" as="h1"` |
| 4 | `Dashboard/Dashboard.tsx` | `title="Overview"` |
| 5 | `Inbox/Inbox.tsx:177` | `title="Notifications"` |
| 6 | `Timeline/Timeline.tsx:360` | `title="Timeline"` |
| 7 | `Dependencies/Dependencies.tsx` | `title="Dependencies" subtitle={…}` |
| 8 | `Board/BoardColumn.tsx` | `title={column.label}` |
| 9 | `Backlog/BacklogGroupHeader.tsx` | `title={label}` |
| 10 | `Dashboard/Section.tsx` | `title={title}` |
| 11 | `PendingEventsPanel.tsx` | `title="Pending Changes" as="h2"` |
| 12 | `Board/BoardCard.tsx` | `as="p" leading="snug" clamp={2}` |
| 13 | `Timeline/Timeline.tsx:695` | `as="p" leading="snug"` |
| 14 | `IssueDetail/ArtifactList.tsx` | `truncate` |
| 15 | `Dependencies/DepGraph.tsx` | `truncate` |
| 16 | `IssueDetail/ActivityTimeline.tsx` | `title={entry.author}` |
| 17 | `IssueDetail/MergeView.tsx` | `title={name}` |
| 18 | `Cycles/Cycles.tsx:564` | `as="h3"` (empty state) |
| 19 | `Inbox/Inbox.tsx:204` | `as="p"` (empty state) |

## Excluded (2)

- `PendingChanges/PendingChanges.tsx:32` — breadcrumb button label (navigation context)
- `PendingChanges/PendingChanges.tsx:81` — list item with hover color transition (interactive behavior)

## Acceptance criteria

- [x] `Heading` component created in `web/src/components/ui/`
- [x] `as` prop renders the correct element (default `span`)
- [x] `subtitle` renders alongside title when provided
- [x] `leading`, `truncate`, `clamp` props apply correct classes
- [x] All 19 instances migrated
- [x] No visual regression
- [x] `make test` passes
