# MergeView crash fix — branch deleted while view is open

## What changed

`web/src/components/IssueDetail/MergeView.tsx` — four edits to guard against `branch_stats` becoming `undefined` mid-render.

## Problem

When `xpo merge` completes in the background (or the branch is otherwise deleted), an SSE update sets `issue.branch_stats` to `undefined`. The component used a non-null assertion (`issue.branch_stats!`) and placed `bs.head_sha` in a `useEffect` dependency array, causing a crash on re-render:

```
Cannot read properties of undefined (reading 'head_sha')
```

## Fix

1. **Removed the non-null assertion** — `const bs = issue.branch_stats` (plain assignment, can be `undefined`).
2. **Optional chaining in useEffect dep** — `bs?.head_sha` so the dependency safely evaluates to `undefined`.
3. **Guarded `defaultCommitMessage`** — `issue.branch_stats?.branch ?? issue.id` instead of a second non-null assertion.
4. **Early return for missing branch_stats** — after the loading/error guards (so all hooks remain unconditional), the component renders a "Branch merged" indicator with a close button instead of the full merge view.

## Why this approach

Once the branch is gone, the merge view's diff/stats content is meaningless — the merge already completed. Showing a brief status message and a close button is the right UX: it tells the user what happened and lets them dismiss the stale view. The alternative (auto-closing via `useEffect` calling `onClose`) risks unexpected navigation during a merge flow.

## Acceptance Criteria

- [x] MergeView does not crash when `branch_stats` becomes `undefined` mid-render — guarded by optional chaining and early return
- [x] When `branch_stats` is missing, the component shows a "branch merged" indicator — early return renders GitMerge icon + message + close button
- [x] No regressions when `branch_stats` is present — normal merge view render path unchanged after the guard
- [x] `make test` passes (lint + tests) — confirmed, `tsc` also clean
