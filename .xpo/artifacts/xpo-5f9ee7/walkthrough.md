## What changed

Backlog rows now dynamically fit labels into available space instead of showing all labels (which squished titles) or using a hard cap at 2 (which wasted space). Board cards keep a static cap at 2.

## The problem

Issues with many labels consumed all horizontal space in backlog rows, truncating titles to a few characters and pushing metadata off-screen. A hard cap at 2 (the initial fix attempt) solved the squishing but wasted space on wide rows.

## How it works

### OverflowLabels component (`web/src/components/ui/OverflowLabels.tsx`)

A reusable component that measures available space and renders as many labels as fit:

1. **Hidden measurer**: All labels are rendered in a hidden div (`position: fixed`, off-screen, `visibility: hidden`). This div is always present and provides stable label width measurements — `visibility: hidden` elements are laid out normally and have measurable `offsetWidth`.

2. **Budget calculation**: A `ResizeObserver` watches the closest `[data-backlog-row]` ancestor. On each resize, the label budget is computed as `35% of the row width`. This ratio was chosen because it leaves ~65% for the title, ID, status, priority, estimate, date, and other fixed elements — enough for a readable title in all common row widths.

3. **Fitting loop**: Iterates the hidden measurer's children, summing their widths with gaps. When the next label would exceed the budget, it stops — but only after reserving space for the `+N` overflow badge (28px) if there are remaining labels.

4. **Stable measurement**: The row width is determined by the viewport/scroll container, not by label content. The `flex-1` spacer in the row absorbs width changes when labels shrink, but the row itself stays the same width. This avoids circular dependency between the measurement and the rendering.

### Integration in Backlog.tsx

- Added `data-backlog-row` attribute to the row div (the `IssueRowDnd` content) so `OverflowLabels` can find its measurement target via `closest()`.
- Replaced the inline label rendering IIFE with `<OverflowLabels labels={issue.labels} />`. The component handles `splitLabels` internally (metadata labels first, then primary).
- The empty-state `+ label` hover text remains inline since it doesn't involve `OverflowLabels`.

### Board cards (BoardCard.tsx)

Kept the static cap at 2 — board cards are narrow fixed-width columns where dynamic measurement adds no value. The hard cap was the right call there.

## Why 35%?

At common row widths:
- 1400px row → 490px for labels → fits ~5-6 labels
- 1000px row → 350px for labels → fits ~3-4 labels
- 700px row → 245px for labels → fits ~2-3 labels

This keeps the title readable (65% minus ~350px of fixed elements) while showing a useful number of labels. The minimum budget floor is 100px (enough for at least one label).
