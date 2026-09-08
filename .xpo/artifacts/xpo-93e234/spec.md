# Spec: Skip ghost children when parent is already visible

## What

In nested hierarchy mode, children appear twice on the same tab — once in their own
status group (under a ghost parent) and again as ghost children under the real parent
in the parent's status group. The ghost children under the real parent are redundant
when the child's own status group is visible.

## Why

On the Active tab, an epic like "UI Polish" (DOING) shows in the In Progress group
with all its children — including PLANNED ones rendered as ghost children. Those same
PLANNED children already appear in the Planned group under a ghost copy of the parent.
This is confusing and clutters the view.

## How

In `useBacklogRows.ts`, in the ghost child filter for **real parents** (not ghost
parents), add a check: if the child's status is in `visibleStatuses` (i.e., the
child's status group is rendered on this tab), skip it. That child already appears in
its own status group — under a ghost parent if needed.

The filter for real parents changes from:
```
groupIssueIds.has(c.id) || showGhosts || !isTerminal(c.status)
```
to:
```
if (groupIssueIds.has(c.id)) return true;       // real child in this group
if (visibleStatuses.includes(c.status)) return false;  // appears in own group
return showGhosts || !isTerminal(c.status);      // existing ghost logic
```

Ghost parent logic is unchanged — ghost parents are still created whenever orphaned
children need hierarchy context.

**Key property**: group header counts and story points are unchanged — they reflect
the number of issues with that status, regardless of where they render visually.

## Acceptance Criteria

- [ ] Active tab: PLANNED children of a DOING parent appear in the Planned group
      (under ghost parent), NOT as ghost children under the real parent in In Progress
- [ ] Backlog tab: ghost children still appear for children whose status group is not
      visible (e.g., DOING children on the Backlog tab)
- [ ] All Issues tab: no ghost children under real parents (every status group is visible)
- [ ] `make test` passes
