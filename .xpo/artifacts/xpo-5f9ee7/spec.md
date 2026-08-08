## What

Dynamically cap rendered labels in backlog rows based on available space, with `+N` overflow when labels don't fit. Board cards use a static cap at 2.

## Why

Issues with many labels squish the title to a few characters and push metadata off-screen. A hard cap at 2 wastes space when the row is wide. The fix should be responsive — show as many labels as fit, cap only when necessary.

## How

### New `OverflowLabels` component (`web/src/components/ui/OverflowLabels.tsx`)

A reusable component that dynamically fits labels into available space:

1. Renders ALL labels in a hidden measurer (`position: fixed`, off-screen, invisible) to get their individual widths
2. Uses a `ResizeObserver` on the closest `[data-backlog-row]` ancestor to detect row width changes
3. Computes available width by subtracting all sibling element widths from the row width
4. Fits as many labels as possible into the available width, reserving space for the `+N` indicator when needed
5. Renders only the visible labels + overflow count

The measurement is stable because:
- Row width is determined by the viewport/container, not by label content
- Sibling widths (id, status, priority, estimate, date, etc.) are independent of labels
- Label widths come from the hidden measurer (constant for a given label set)

### Backlog rows (`Backlog.tsx`)

- Added `data-backlog-row` attribute to the row div for the ResizeObserver target
- Replaced inline label rendering with `<OverflowLabels labels={issue.labels} />`
- The `OverflowLabels` component handles `splitLabels`, label ordering (metadata first), and overflow

### Board cards (`BoardCard.tsx`)

Kept the static cap at 2 — board cards are narrow fixed-width columns where a dynamic approach adds no value.

## Acceptance Criteria

- Backlog rows show as many labels as fit, with `+N` for the rest
- Narrowing the window reduces visible labels; widening shows more
- Titles always have reasonable room (not squished to a few characters)
- Board cards still cap at 2 with `+N`
- Clicking the labels area still opens the label popover
