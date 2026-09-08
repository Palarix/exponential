# Walkthrough: Board view keyboard help

## What changed

`web/src/components/KeyboardHelp/KeyboardHelp.tsx` — the `?` keyboard shortcuts overlay.

## Changes

### Board section added

Added a new `BOARD` shortcut group documenting all existing Board keyboard shortcuts (navigation, S/L/E quick-set, Enter, copy ID, and per-column status numbers). The Board view already had full keyboard support — only the help documentation was missing.

### Key separator semantics

The overlay previously used `/` as a separator between all multi-key entries, which was ambiguous — `G / O` looked like "G or O" when it meant "G then O", and `⌘ / K` looked like alternatives when it meant simultaneous press.

Added a `join` field to `ShortcutEntry` with three modes:
- **`"or"`** (default) — renders `/` for alternatives (e.g., `J / ↓`)
- **`"seq"`** — renders `→` for sequences (e.g., `G → O`)
- **`"combo"`** — renders `+` for simultaneous keys (e.g., `⌘ + K`)

A small `KeySeparator` component picks the glyph from a `JOIN_GLYPHS` lookup.

### Status number mappings spelled out

Replaced the opaque `1–7` and `1–5` labels with individual entries showing the actual status each number maps to (e.g., `1 → Backlog`, `2 → Planned`). Board lists all 7 statuses, Issue Detail lists 5. The Pickers section keeps its `1–5` since those are positional selections in a visible list.

### Wider 4-column layout

Widened the overlay from `max-w-2xl` (2-column) to `max-w-5xl` (4-column) to reduce vertical scrolling and make better use of screen width. The 6 groups (Global, Backlog, Board, Issue Detail, Notifications, Pickers & Menus) fit in roughly two rows.
