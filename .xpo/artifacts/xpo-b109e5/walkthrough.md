## What was built

Terminal-status ghost children (DONE, CANCELED, DUPLICATE) are now hidden by default in the nested hierarchy view, with a "Show done ghosts" toggle to opt back in.

## How the pieces fit together

**`web/src/components/Backlog/useBacklogRows.ts`** — the ghost child filter for real (non-ghost) parents changed from showing all children to:

```typescript
.filter((c) => groupIssueIds.has(c.id) || showGhosts || !isTerminal(c.status))
```

Three cases:
- Real children (`groupIssueIds`): always pass — these are issues that belong in this status group
- Non-terminal ghosts (`!isTerminal`): always pass — a PLANNED child under a DOING parent is useful context
- Terminal ghosts: only pass when `showGhosts` is true

Ghost parents are unaffected — they always show because they provide the structural entry point for orphaned children. Removing them would either hide real issues or require promoting orphans to top-level, which breaks drag-and-drop sort ordering.

The `showGhosts` parameter was added to the hook signature (defaults to `true` for backwards compatibility) and included in the `useMemo` dependency array.

**`web/src/components/Backlog/Backlog.tsx`** — wiring:
- `showGhosts` state initialized from `localStorage` key `exponential-backlog-show-done-ghosts`, defaulting to `false`
- `toggleGhosts` callback flips and persists the value
- Passed as the new ninth argument to `useBacklogRows`
- "Show done ghosts" button in the view options menu, rendered only when `hierarchyMode === "nested"` (ghosts don't exist in flat mode), positioned after "Show empty groups"

## Key decisions

- **Default off**: the original bug report was that done ghosts are noisy — off-by-default fixes the reported issue while the toggle preserves the option for users who want full tree context
- **Only terminal statuses filtered**: non-terminal ghosts (e.g. PLANNED child under DOING parent in Active tab) are always useful context and never filtered
- **Ghost parents always shown**: filtering them creates an orphan-promotion problem that breaks DnD reordering — keeping them is simpler and they're lightweight visual context
