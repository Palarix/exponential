# Spec: Primary vs Metadata Labels

## Summary

Split an issue's labels into **primary** (matching `default_labels` config) and **metadata** (everything else) groups for display ordering. All labels render with the same visual style; only the sort order changes.

## Definitions

- **Primary labels**: Any label whose name appears in the project's `default_labels` config list (default: `bug`, `feature`, `epic`, `improvement`).
- **Metadata labels**: All remaining labels on the issue.

## Behavior

### Ordering (all views)

Labels are rendered in a single group: metadata labels first (sorted alphabetically, case-insensitive), then primary labels (sorted by their position in `default_labels` config).

### Visual style

All labels use the same `LabelBadge` component — no visual distinction between primary and metadata labels. The ordering alone conveys the grouping.

### Reserve component: `LabelIndicator`

A `LabelIndicator` component exists in `Badge.tsx` (small colored strip bars using primary label colors) for potential future use as an inline title indicator. Not currently rendered in any view.

## Files Changed

1. **`web/src/utils/labels.ts`** — Added `splitLabels()` utility (partitions labels into primary/metadata with proper ordering) and `contrastTextColor()` (reserved for future use).
2. **`web/src/components/ui/Badge.tsx`** — Added `LabelIndicator` component (exported but not currently used in views).
3. **`web/src/components/ui/index.ts`** — Exported `LabelIndicator`.
4. **`web/src/components/Backlog/Backlog.tsx`** — Uses `splitLabels()` to order labels: metadata first, primary last.
5. **`web/src/components/Board/BoardCard.tsx`** — Uses `splitLabels()` to order labels: metadata first, primary last.
6. **`web/src/components/IssueDetail/PropertySidebar.tsx`** — Uses `splitLabels()` to order labels: metadata first, primary last.
