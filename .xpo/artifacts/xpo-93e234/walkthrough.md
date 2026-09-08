# Walkthrough: Skip ghost children when parent is already visible

## Context

The backlog's nested hierarchy mode uses "ghost" rows to maintain parent-child context
across status groups. When a child's status differs from its parent's, two mechanisms
kick in:

1. **Ghost parent** — a faded parent row injected into the child's status group so the
   child appears nested under its parent for context.
2. **Ghost children** — faded child rows shown under the real parent in the parent's
   status group, giving the parent a complete picture of its children.

The bug: both mechanisms fired simultaneously when both status groups were visible on
the same tab, causing every child to appear twice.

## What changed

**`web/src/components/Backlog/useBacklogRows.ts`** — one filter addition in the ghost
child logic for real parents (line ~243).

When building the visual children list for a real (non-ghost) parent, the existing
filter decided which children to show as ghosts:

```ts
groupIssueIds.has(c.id) || showGhosts || !isTerminal(c.status)
```

The fix adds an early exit: if the child's status is in `visibleStatuses` (the status
groups rendered on this tab), skip it. That child already appears in its own status
group — under a ghost parent if needed — so showing it again as a ghost child is
redundant.

```ts
if (groupIssueIds.has(c.id)) return true;        // real child in this group
if (visibleStatuses.includes(c.status)) return false;  // has its own visible group
return showGhosts || !isTerminal(c.status);       // existing ghost logic
```

## How the pieces fit

- **Active tab** (PLANNED, DOING, BLOCKED): a DOING parent no longer shows PLANNED
  children as ghosts, because the Planned group is visible. Those children appear in
  Planned under a ghost parent.
- **Backlog tab** (BACKLOG only): a BACKLOG parent still shows DOING children as ghosts,
  because the In Progress group is not visible on this tab.
- **All Issues tab**: no ghost children under any real parent, since every status group
  is visible. Children always appear in their own status group.
- Ghost parent logic is completely unchanged.

## Key decisions

- **Group counts unchanged**: the header count still reflects all issues with that
  status, even if some render visually under a parent in another group. This is
  consistent with how counts worked before (they never matched visible rows exactly
  in nested mode).
- **Minimal change**: only the ghost child filter was touched. Ghost parent creation,
  rendering, drag-and-drop, and the `showGhosts` toggle are unaffected.
