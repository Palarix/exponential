# Walkthrough: Fix sub-issues preview overflow

## What changed

Both `Backlog.tsx` and `PropertySidebar.tsx` had identical preview child rows in the "Update sub-issues?" modal. Both were updated the same way.

## Fixes

### Label cap

Each child row previously rendered all labels via `issue.labels?.map(...)`. Now the labels are first run through `splitLabels()` to order primary labels before metadata, then sliced to 2. If more exist, a `+N` text span is appended.

```tsx
const { primary: pl, metadata: ml } = splitLabels(rawLabels, defaultLabels);
const allLabels = [...pl, ...ml];
const extraCount = Math.max(0, allLabels.length - 2);
// render allLabels.slice(0, 2) + "+N" if extraCount > 0
```

Primary-first ordering ensures the most important labels (bug, feature, epic, improvement) always appear in the visible two slots.

### Title truncation

The title `<span>` already had `truncate min-w-0` but sat alongside a `<div className="flex-1" />` spacer that consumed all remaining space. Removed the spacer and added `flex-1` directly to the title span so it takes the available width and actually truncates.

### Scrollable list

Added `max-h-64 overflow-y-auto` to the child list container. For epics with many children, the list scrolls within the modal instead of pushing buttons off-screen. 16rem (~6 rows) keeps the action buttons visible.
