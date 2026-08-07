# Walkthrough: Primary vs Metadata Label Ordering

## What changed

Labels on issues are now split into two groups for display: **metadata labels** (custom/ad-hoc labels like "Frontend", "UX", "WebApp") and **primary labels** (labels matching the `default_labels` config — typically "bug", "feature", "epic", "improvement"). Metadata labels render first (sorted alphabetically), primary labels render after (sorted by their position in the config).

## How it works

### `splitLabels()` in `web/src/utils/labels.ts`

This is the core utility. It takes an issue's labels array and the `defaultLabels` context (from `DefaultLabelsContext`, provided at the app root), and returns `{ primary, metadata }`:

- **primary**: labels whose name (case-insensitive) matches any entry in `defaultLabels`, sorted by their position in the config list.
- **metadata**: everything else, sorted alphabetically (case-insensitive).

The function is pure — no side effects, easy to test.

### View integration

Each view calls `splitLabels()` and renders metadata first, then primary, all using the same `LabelBadge` component:

- **Backlog** (`Backlog.tsx`): The labels column on the right side of each row. The component reads `DefaultLabelsContext` via `useContext` and calls `splitLabels()` inside the label rendering IIFE. The label picker popover remains unchanged — it still operates on the full `issue.labels` array.

- **Board** (`BoardCard.tsx`): The bottom metadata row of each card. Same pattern — `useContext(DefaultLabelsContext)` at the top of `BoardCardContent`, `splitLabels()` to get the two arrays, render metadata then primary.

- **Detail sidebar** (`PropertySidebar.tsx`): The "Labels" card. Same pattern. The `+` button and `LabelPicker` remain attached to the full label set.

### Reserve component: `LabelIndicator`

During development we explored a small colored strip indicator (thin rounded rectangles in the primary label's color) rendered before the issue title. The component is built and exported (`Badge.tsx`, `index.ts`) but not rendered anywhere. It accepts `labels: string[]` and renders a compact row of colored bars, one per label.

### `contrastTextColor()` in `labels.ts`

A WCAG-based utility that returns `#000000` or `#ffffff` depending on a hex color's relative luminance. Built during early iterations where primary labels had solid colored backgrounds. Retained as a general utility.

## What didn't change

- The `LabelBadge` component itself is untouched — same dot, border, text style.
- The `LabelPicker` and label editing flows are unchanged.
- The backend/API returns labels in the same order as before — sorting is purely frontend.
- The `default_labels` config (used to determine which labels are "primary") already existed and is reused as-is.
