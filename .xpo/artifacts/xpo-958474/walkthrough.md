# Walkthrough: Markdown checklist visibility and alignment

## What changed

**Single file:** `web/src/index.css`

Fixed checkbox styling in both rendered markdown (`.prose-exponential` task lists) and the tiptap editor (`.tiptap ul[data-type="taskList"]`).

## Changes

### Rendered markdown (remark-gfm task lists)

| Property | Before | After |
|----------|--------|-------|
| `ul.contains-task-list` padding-left | `0.25em` | `0` |
| `li.task-list-item` padding-left | inherited `0.375em` from base `li` | `0` |
| checkbox border | `--color-border-default` (`#282828`) | `--color-border-control` (`#404040`) |
| checkbox margin-right | `0.25em` | `0.5em` |

### Tiptap editor task lists

| Property | Before | After |
|----------|--------|-------|
| `ul[data-type="taskList"]` padding-left | `0.25em` | `0` |
| `ul[data-type="taskList"] li` padding-left | inherited `0.375em` | `0` |
| checkbox border | `--color-border-default` (`#282828`) | `--color-border-control` (`#404040`) |
| checkbox margin-right | none | `0.5em` |

## Why these values

- `--color-border-control` (`#404040`) is the design system's dedicated token for checkboxes and toggles — nearly double the contrast of `--color-border-default` (`#282828`) against the `#0f0f0f` canvas
- Zero padding on the task list `ul` and `li` puts checkboxes flush-left with ordered/unordered list content above and below

## Acceptance criteria

- [x] Checkbox borders clearly visible against dark background — switched to `--color-border-control` (`#404040`)
- [x] Checklist items aligned flush-left with surrounding content — zeroed padding on both `ul` and `li`
- [x] Tiptap editor checklists match rendered markdown appearance — same fixes applied to `.tiptap` task list rules
- [x] `make test` passes
