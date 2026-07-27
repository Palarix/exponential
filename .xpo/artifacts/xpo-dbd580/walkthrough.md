# Walkthrough: Improve prose-exponential markdown rendering

## What changed

All changes are in a single file: `web/src/index.css`, within the `.prose-exponential` class and its Tiptap editor counterpart.

## Tables (new)

Previously there were zero table styles — tables rendered as unstyled HTML. Now:

- `table` gets `border: 1px solid var(--color-border-default)` and `border-collapse: collapse`
- `th` and `td` get `border: 1px solid var(--color-border-subtle)`, `padding: 0.5em 0.75em`, `text-align: left`, `vertical-align: top`
- `thead th` gets a distinct look: `background: var(--color-surface-1)`, `color: var(--color-text-secondary)`, `font-size: 0.875em`, `font-weight: 600`, and border uses `--color-border-default` (slightly stronger than body cell borders)
- Inline code inside tables is slightly smaller (`font-size: 0.9em`)

The same table styles are duplicated under `.prose-exponential .tiptap` so the Tiptap editor renders tables identically to the read-only view.

## Headings

**Before:** All headings shared `margin: 1.75em 0 0.75em` and inherited the body `line-height: 1.75`. Sizes were h1 `1.25em`, h2 `1.1em`, h3 `1em` — barely distinguishable.

**After:** Each heading level has its own top margin to establish visual hierarchy:

| Level | Font size | Top margin |
|-------|-----------|------------|
| h1 | 1.375em | 2.5em |
| h2 | 1.2em | 2em |
| h3 | 1.05em | 1.5em |
| h4 | 1em | 1.25em |

All headings share `line-height: 1.3` and `margin-bottom: 0.75em`. The `:first-child` reset still removes top margin on the first element.

## Lists

**Before:** `padding-left: 1.5em` on all `ul`/`ol`, creating a noticeable indent from body text.

**After:**
- First-level lists: `padding-left: 1.125em` — just enough for the marker (disc/number) to fit, so text is nearly flush with body paragraphs
- `li` gets `padding-left: 0.375em` — creates a small visual gap between the marker and the text content
- Nested lists (`li > ul`, `li > ol`): keep `padding-left: 1.5em` for clear visual hierarchy
- Added `li > p` margin rules to prevent double-spacing when GFM wraps list item content in `<p>` tags — `margin: 0.375em 0` with first/last-child resets

## Vertical rhythm

Small increases to breathing room between block elements:
- Paragraphs: `0.75em` → `0.875em`
- Code blocks (`pre`): `0.75em` → `1em`
- Blockquotes: `0.75em` → `1em`
- Horizontal rules: `1em` → `1.5em`
- Lists: `0.75em` → `0.875em` (matches paragraphs)

## Inline code

- Font size: `0.9em` → `0.875em` (slightly smaller relative to body)
- Vertical padding: `0.25em` → `0.2em` (tighter, less "boxy")
- `pre code` (code inside code blocks): explicit `font-size: 0.9375em` so it doesn't inherit the inline-code shrink

## Key decisions

**`padding-left: 1.125em` for first-level lists** — The user wanted lists nearly flush with body text but with markers (bullets/numbers) staying within the text column. This value is the sweet spot: just enough for `list-style-position: outside` markers to fit without visible indentation of the text content.

**`padding-left: 0.375em` on `li`** — Creates the gap between marker and text. This is separate from the list-level padding so nested lists don't compound the gap.

**Full grid borders on tables** — The first iteration used only row separators (bottom borders on `td`), but the user preferred all borders for clearer cell delineation. Header row gets `--color-border-default` (stronger) while body cells get `--color-border-subtle`.

**Graduated heading margins** — Rather than a single shared `margin-top`, each level gets its own value. This makes the document structure scannable — an h2 creates more visual separation than an h3, reinforcing the hierarchy.