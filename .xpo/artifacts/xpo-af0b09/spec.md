# Merge view live refresh

## What

MergeView fetches data once on mount and never refreshes.

## Changes

1. **Auto-refresh on new commits**: Added `bs.head_sha` to the data-loading `useEffect` dependency. SSE events already update `branch_stats` — now MergeView re-fetches when the head moves.

2. **Manual refresh button**: `RefreshCw` icon at the right end of the tab bar. Calls `refreshData()` to re-fetch commits, files, diff, and mergeability on demand.

3. **`.xpo/` filtering**: Files with paths starting with `.xpo/` are filtered from the files list and the parsed diff map.

4. **Scrollbar fix**: Global `scrollbar-gutter: stable` replaced with `overflow-y: scroll` targeting Tailwind overflow classes — reserves only the thin 6px scrollbar width instead of 15px. MergeView top bar adjusted to `pl-5 pr-3` to balance the gutter.

## AC

- Merge view refreshes when new commits are pushed
- Manual refresh button available on all tabs
- `.xpo/issues.db` not shown in files/diff
- No excessive scrollbar gutter space
