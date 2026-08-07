# Walkthrough: Ghost parent row duplicates popover

## Problem

In Active/Backlog views, an issue can appear twice in the row list: once as a real row in its own status group and once as a ghost parent in a child's status group. The popover state was `{ issueId: string, type: string }`, so both rows matched the same `openPopover?.issueId === issue.id` check. Clicking a status/priority/labels/estimate popover on either row opened it on both simultaneously.

## Fix

Changed the popover key from `issueId` to `rowIndex`. Each row in the `rows` array has a unique index, even when the same issue appears multiple times.

### `web/src/components/Backlog/Backlog.tsx`

**State type change** (line 116):
```ts
// Before
const [openPopover, setOpenPopover] = useState<{
  issueId: string;
  type: "status" | "estimate" | "labels" | "priority";
} | null>(null);

// After
const [openPopover, setOpenPopover] = useState<{
  rowIndex: number;
  type: "status" | "estimate" | "labels" | "priority";
} | null>(null);
```

**Popover toggle clicks** — all four popover types (priority, status, labels, estimate) updated from:
```ts
setOpenPopover(
  openPopover?.issueId === issue.id && openPopover?.type === "priority"
    ? null
    : { issueId: issue.id, type: "priority" }
)
```
to:
```ts
setOpenPopover(
  openPopover?.rowIndex === i && openPopover?.type === "priority"
    ? null
    : { rowIndex: i, type: "priority" }
)
```

Where `i` is the row index from the `issueRows.map(({ row, index: i })` loop.

**Popover visibility checks** — all four types updated from `openPopover?.issueId === issue.id` to `openPopover?.rowIndex === i`.

**Keyboard handler** — the `s`, `l`, `e`, `p` shortcuts use `focusedIndexRef.current` instead of `row.issue.id`:
```ts
const ri = focusedIndexRef.current;
setOpenPopover({ rowIndex: ri, type: "status" });
```

## What was NOT changed

The `contextMenu` state still uses `issueId` — this is correct because context menus render as a portal at fixed screen coordinates (not inside the row), so the duplicate-row problem doesn't apply.
