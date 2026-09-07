# Scrollbar layout shift fix

## What

Vertical scrollbar appearance causes layout to shift left across all views.

## Why

When content overflows and a scrollbar appears, the content area shrinks to accommodate it, causing a visible horizontal jump.

## How

Add `scrollbar-gutter: stable` in `index.css` right after the existing scrollbar styling block (line 143). This reserves space for the scrollbar gutter even when no scrollbar is visible, preventing the layout shift.

Target: elements that use `overflow: auto` or `overflow: scroll` (the Tailwind `overflow-y-auto` class). Use a global selector to catch all scroll containers at once.

Also remove the one-off inline `scrollbarGutter: "stable"` from `IssueDetail.tsx:313` since the global rule will cover it.

## Acceptance Criteria

- No layout shift when scrollbar appears/disappears in any view
- Scrollbar gutter space is reserved even when content doesn't overflow
- Existing thin scrollbar styling preserved
