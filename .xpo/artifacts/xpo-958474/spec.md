# Fix: Markdown checklists barely visible and not left-aligned

## What
Checkbox squares in rendered markdown are nearly invisible and indented from the left margin.

## Root cause

**Low contrast:** Border uses `--color-border-default` (`#282828`) on `--color-surface` (`#0f0f0f`) — barely distinguishable. The design system has `--color-border-control` (`#404040`) specifically for checkboxes/toggles but it's not used here.

**Indentation:** `ul.contains-task-list` has `padding-left: 0.25em`, pushing checkboxes right of the content margin.

## Fix
In `index.css`:
1. Change checkbox border from `--color-border-default` to `--color-border-control`
2. Change `ul.contains-task-list` padding-left from `0.25em` to `0`

## Acceptance criteria
- [ ] Checkbox borders clearly visible against dark background
- [ ] Checklist items aligned flush-left with surrounding content
- [ ] `make test` passes
