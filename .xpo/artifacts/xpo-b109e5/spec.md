## What

Completed ghost children (DONE/CANCELED/DUPLICATE) appear in non-terminal filtered views like Active and Backlog, cluttering the tree with irrelevant items.

## Why

The nested hierarchy mode shows all children of a real parent as ghost rows for context, regardless of status. A PLANNED ghost under a DOING parent is useful context — a DONE ghost is noise.

## How

### 1. Filter terminal ghost children (default)

In `useBacklogRows.ts`, the ghost child filter excludes terminal-status ghosts unless the toggle is on:

```
groupIssueIds.has(c.id) || showGhosts || !isTerminal(c.status)
```

- Real children: always shown
- Non-terminal ghosts (PLANNED, DOING, BLOCKED, BACKLOG): always shown
- Terminal ghosts (DONE, CANCELED, DUPLICATE): shown only when `showGhosts` is true

### 2. "Show done ghosts" toggle

- View menu toggle in nested mode, defaults to **off**
- Persisted to `localStorage` key `exponential-backlog-show-done-ghosts`
- Ghost parents unaffected — they always show

## Acceptance Criteria

- [x] Terminal ghost children hidden by default in non-terminal views
- [x] Non-terminal ghost children always shown
- [x] "Show done ghosts" toggle in view menu (nested mode only)
- [x] Toggle defaults to off
- [x] Ghost parents always shown
- [x] Preference persists via localStorage
