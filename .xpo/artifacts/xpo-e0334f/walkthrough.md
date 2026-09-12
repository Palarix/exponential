# Text — Implementation Walkthrough

## What was built

A `Text` UI primitive that captures the repeated `text-{size} text-[var(--color-...)]` typography patterns across the app. ~69 instances migrated across 18 files.

## Structure

Three new files in `web/src/components/ui/`:

- **`text-utils.ts`** — type exports (`TextColor`, `TextSize`, `TextWeight`) and `textClass(opts)` utility that maps props to className strings via lookup maps
- **`Text.tsx`** — the React component, composing the utility with `children`, `as`, `color`, `size`, `weight`, `mono`, `tabular`, `truncate` props
- **`Text.test.ts`** — 9 unit tests covering defaults, all color/size/weight maps, extras, and combinations

## API

```tsx
<Text>muted text</Text>                             // text-sm text-[var(--color-text-muted)] font-normal
<Text color="primary">primary</Text>                // text-sm text-[var(--color-text-primary)] font-normal
<Text size="xs" mono>sha</Text>                     // text-xs text-[var(--color-text-muted)] font-normal font-mono
<Text weight="medium" color="primary">name</Text>   // text-sm text-[var(--color-text-primary)] font-medium
<Text color="success">+3</Text>                     // text-sm text-[var(--color-success)] font-normal
<Text tabular>42</Text>                              // text-sm text-[var(--color-text-muted)] font-normal tabular-nums
```

## Key design decision: `font-normal` always emitted

During tophat, the "pts" unit text on the Dashboard velocity table appeared bolder than before. Root cause: the original `<span>` had explicit `font-normal` to override the parent `<td>`'s `font-semibold`. The initial `Text` implementation mapped `weight="regular"` to an empty string, allowing CSS inheritance to bleed through.

Fix: `weight="regular"` now emits `font-normal` explicitly. This makes `Text` self-contained — it never inherits weight from parent elements, which is the correct behavior for a typography primitive.

## Migration breakdown

### Heading → Text swaps (2)
- `ActivityTimeline.tsx` — author name display
- `MergeView.tsx` — comment author name display

### By pattern
- **Muted text** (most common): timestamps, labels, counts, empty state messages
- **Primary text**: names, titles, values displayed as content
- **Secondary text**: descriptions, change summaries
- **Mono text**: SHAs, issue IDs, filenames, code references
- **Tabular text**: counts, stats, percentages
- **Semantic colors**: success (diff additions), error (diff deletions, error messages)

### Files touched (18)
BoardCard, BoardColumn, Cycles, ActivityFeed, Dashboard, Section, DepGraph, Inbox, ActivityTimeline, ArtifactList, IssueDetail, MergeView, PropertySidebar, Labels, MyIssues, PendingChanges, PendingEventsPanel, Timeline

### What was skipped
- Interactive elements with `hover:`, `transition-`, `cursor-` classes
- Elements with `style={{ }}` conditional styling
- Elements with layout-specific classes that made extraction awkward
- `Heading` and `CountBadge` instances (separate components)

## Acceptance criteria

- [x] `Text` component created in `web/src/components/ui/`
- [x] Props: `color`, `size`, `weight`, `mono`, `tabular`, `truncate`, `as`
- [x] Default: `<span className="text-sm text-[var(--color-text-muted)] font-normal">`
- [x] Swapped `Heading` → `Text` for author names (ActivityTimeline, MergeView)
- [x] Migrated muted, secondary, primary, and semantic color instances
- [x] `Heading` remains separate
- [x] `make test` passes (330 frontend tests, 13 Go packages)