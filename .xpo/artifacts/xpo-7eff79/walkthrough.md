# Walkthrough: Notification inbox label overflow fix

## What changed

One file modified: `web/src/components/Inbox/Inbox.tsx`.

### The fix

The label rendering block (line ~393) rendered all labels via `issue.labels.map(...)`. Replaced with `issue.labels.slice(0, 2).map(...)` and a conditional `+{count}` span when `labels.length > 2`.

This is the same pattern used in `SubIssuesTable.tsx`, `PropertySidebar.tsx` (cascade dialog), and `Backlog.tsx` (drag overlay) — cap at 2 visible labels, show "+N" for the rest.