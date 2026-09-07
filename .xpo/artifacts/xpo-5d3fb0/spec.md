# Inline create row layout fix

## What

The inline create row in Backlog status groups doesn't align with normal issue rows.

## Why

Three mismatches between the inline create row and normal issue rows:

1. **Priority icon** — issue row wraps in `<div><button className="w-6 h-6 -m-1">`, inline row places `PriorityIcon` bare (16px).
2. **Issue ID column** — issue row uses `CopyableId` (variable-width mono text), inline row uses `<span className="w-28 shrink-0" />` (fixed 112px).
3. **Status icon** — same wrapping mismatch as priority.

## How

In `Backlog.tsx`, update the inline create row to:

1. Wrap `PriorityIcon` in a non-interactive div matching the issue row's `w-6 h-6 -m-1` sizing.
2. Replace the `w-28` spacer with a dimmed placeholder ID (`xpo-······`) using the same classes as `CopyableId` (`font-mono text-xs text-left shrink-0 tabular-nums text-[var(--color-text-muted)]`).
3. Wrap `StatusIcon` in a non-interactive div matching the issue row's sizing.

## Acceptance Criteria

- Inline create row columns visually align with normal issue rows
- Placeholder ID is visible but clearly non-interactive (muted color)
- No functional changes to create behavior
