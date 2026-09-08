# Merge view live refresh

## What changed

The merge view now stays up-to-date when the branch changes, instead of only loading data on mount.

## Auto-refresh via head_sha

Extracted the data-loading `Promise.all` into a `refreshData` callback. The `useEffect` that calls it depends on `[refreshData, bs.head_sha]`. The existing SSE pipeline (`watch_git.go` → SSE `UPDATE` → `App.tsx` re-fetches issues → `branch_stats.head_sha` updates) now flows through to MergeView automatically.

## Manual refresh button

A `RefreshCw` icon at the right end of the tab bar calls `refreshData()`. Placed away from the X close button to avoid accidental clicks. Covers the gap where uncommitted working directory changes don't trigger SSE events.

## .xpo/ filtering

Files with paths starting with `.xpo/` are filtered in two places:
- The files list: filtered in the `refreshData` callback after fetch
- The diff: filtered in `parseDiffByFile()` which deletes `.xpo/` keys from the result map

## Scrollbar approach

Replaced `scrollbar-gutter: stable` (reserved ~15px gutter) with `overflow-y: scroll` targeting Tailwind overflow classes. With the existing 6px `::-webkit-scrollbar` styling and transparent track, the scrollbar space is always reserved but only 6px wide. MergeView top bar adjusted to `pl-5 pr-3` to balance the gutter visually.
