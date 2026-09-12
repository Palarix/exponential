# Heading — Implementation Walkthrough

## What was built

A `Heading` UI primitive that replaces 19 instances of the `text-sm font-medium text-[var(--color-text-primary)]` pattern across the app. The component supports element type selection, subtitle composition, and text overflow variants.

## Structure

Three new files in `web/src/components/ui/`:

- **`heading-utils.ts`** — two pure functions:
  - `headingClass(opts)` — builds the className string, optionally adding `leading-snug`, `truncate`/`min-w-0`, or `line-clamp-{n}`
  - `headingWrapperClass()` — returns the flex wrapper classes for subtitle layout
- **`Heading.tsx`** — the React component, composing the utilities with `as`/`subtitle`/`leading`/`truncate`/`clamp` props
- **`Heading.test.ts`** — 7 unit tests covering base classes, leading, truncate, clamp, and combinations

## API

```tsx
<Heading title="Board" />                          // default: <span> with base classes
<Heading title="Labels" as="h1" />                  // semantic heading element
<Heading title="Dependencies" subtitle={…} />       // flex wrapper with subtitle
<Heading as="p" leading="snug" clamp={2} title={…} /> // card title with line clamping
<Heading truncate title={…} />                       // ellipsis overflow
```

## Migration

All 19 sites were 1:1 replacements — identical rendered HTML, no layout or behavioral changes.

### TopBar view titles (7)
Board, Cycles, Labels (`as="h1"`), Dashboard, Inbox, Timeline, Dependencies (`subtitle={stats}`)

### Structural headings (4)
BoardColumn, BacklogGroupHeader, Dashboard Section, PendingEventsPanel (`as="h2"`)

### Card/commit titles (2)
BoardCard (`as="p" leading="snug" clamp={2}`), Timeline commit detail (`as="p" leading="snug"`)

### Truncated labels (2)
ArtifactList (`truncate`), DepGraph focus header (`truncate`)

### Author names (2)
ActivityTimeline, MergeView comments — these are semantically `Text` rather than `Heading`; noted in xpo-e0334f for swap when the `Text` component lands.

### Empty state headings (2)
Cycles (`as="h3"`), Inbox (`as="p"`)

## Excluded (2)

- `PendingChanges/PendingChanges.tsx:32` — breadcrumb button label in a navigation chain
- `PendingChanges/PendingChanges.tsx:81` — list item title with hover color transition (interactive behavior beyond static styling)

## Key decisions

- **`Heading` stays separate from future `Text` component**: headings are semantically distinct — a future design change to all headings (e.g. different weight or size) should apply uniformly without affecting body text. Filed xpo-e0334f for the `Text` extraction.
- **`as` includes `'p'`**: card titles and empty state text render as `<p>` elements, which is semantically correct for those contexts.
- **`truncate` adds `min-w-0`**: required in flex layouts for CSS truncation to work; bundling it avoids a common gotcha.

## Acceptance criteria

- [x] `Heading` component created in `web/src/components/ui/`
- [x] `as` prop renders the correct element (span/h1/h2/h3/p, default span)
- [x] `subtitle` renders alongside title when provided
- [x] `leading`, `truncate`, `clamp` props apply correct classes
- [x] All 19 instances migrated
- [x] No visual regression — identical rendered HTML
- [x] `make test` passes (321 frontend tests, 13 Go packages)