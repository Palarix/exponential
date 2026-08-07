# Walkthrough: Sub-issues table label overflow fix

## What changed

One file modified: `web/src/components/IssueDetail/SubIssuesTable.tsx`.

### The problem

Each sub-issue row is a flex container with labels rendered as direct children. When an issue had 5+ labels, the row wrapped to multiple lines because nothing constrained the label count or the row's overflow.

### The fix

Replaced `{child.labels?.map(...)}` (renders all labels) with the same capped pattern used in the cascade dialog and backlog drag overlay:

- `{child.labels?.slice(0, 2).map(...)}` — show at most 2 labels
- `{(child.labels?.length || 0) > 2 && <span>+{count}</span>}` — show "+N" for the rest

This matches the existing convention in `PropertySidebar.tsx` (line 759) and `Backlog.tsx` (line 1866) where labels are capped at 2 with a "+N more" count.

The initial attempt used `overflow-hidden` with `max-w-[50%]` to clip labels, but that produced a worse UX — labels were cut mid-badge. The slice + count approach is cleaner and consistent with the rest of the app.