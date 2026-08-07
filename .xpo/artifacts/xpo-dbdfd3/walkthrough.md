# Walkthrough: Fix story point aggregation in status group headers

## What changed

One file: `web/src/components/Backlog/useBacklogRows.ts`.

## The bug

The status group header (e.g., "In Progress ▲ 12") summed story points incorrectly for parent issues. The backend overwrites a parent's `estimate` with the sum of ALL children's estimates regardless of status. The old frontend code used this inflated value directly, only excluding "epic"-labeled parents to avoid double-counting — missing other parent types and including DONE children's points.

## The fix

Replaced lines 76-84. The old code:

```ts
const epicIds = new Set(
  groupIssues.filter(i => i.labels?.includes("epic") && ...).map(i => i.id)
);
const storyPoints = groupIssues.reduce(
  (sum, i) => sum + (epicIds.has(i.id) ? 0 : (i.estimate || 1)), 0
);
```

The new code:

```ts
const groupIssueIds = new Set(groupIssues.map(i => i.id));
const storyPoints = groupIssues.reduce((sum, i) => {
  const children = childrenByParent.get(i.id);
  if (children && children.length > 0) {
    const childrenInGroup = children.filter(c => groupIssueIds.has(c.id));
    if (childrenInGroup.length > 0) return sum; // skip — children counted individually
    const incomplete = children.filter(c => c.status !== "DONE");
    return sum + incomplete.reduce((s, c) => s + (c.estimate || 1), 0);
  }
  return sum + (i.estimate || 1);
}, 0);
```

Three cases:
1. **Leaf issue** (no children): use its own estimate. Unchanged.
2. **Parent with children in this group**: return `sum` unchanged — the children are each counted individually in the same reduce loop, so adding the parent would double-count.
3. **Parent with no children in this group** (all children in other statuses): sum only incomplete children's estimates, excluding DONE.

The `groupIssueIds` set was previously computed inside the `if (expandedGroups.has(...))` block. It's now hoisted above the header calculation so both can use it, and the duplicate declaration inside the block was removed.
