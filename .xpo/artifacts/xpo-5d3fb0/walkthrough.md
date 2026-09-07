# Inline create row layout fix

## What changed

The inline create row that appears when clicking `+` on a status group header in the Backlog view had misaligned columns compared to normal issue rows.

## Root cause

Three structural mismatches in `web/src/components/Backlog/Backlog.tsx`:

1. **Priority icon** was placed bare (16px), while issue rows wrap theirs in a `w-6 h-6 -m-1` button container — different effective width.
2. **Issue ID column** used a fixed `w-28` (112px) spacer, while real rows use `CopyableId` with variable-width mono text.
3. **Status icon** had the same bare-vs-wrapped mismatch as priority.

## Fix

Wrapped both icons in non-interactive `w-6 h-6 -m-1` divs matching the issue row structure, and replaced the blank spacer with a dimmed `xpo-······` placeholder using the same `font-mono text-xs tabular-nums` styling as `CopyableId`. The placeholder uses `opacity-40` to make it clearly non-interactive.

## Key decision

Used `opacity-40` rather than hiding the ID column entirely — the visible placeholder gives users a visual cue about where the ID will appear once the issue is created.
