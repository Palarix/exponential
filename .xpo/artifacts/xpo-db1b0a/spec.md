# MergeView crashes when branch is deleted while view is open

## What

`MergeView.tsx` crashes with `Cannot read properties of undefined (reading 'head_sha')` when the viewed issue's branch is deleted (e.g. after `xpo merge` completes in the background).

## Why

The component uses a non-null assertion (`issue.branch_stats!`) and places `bs.head_sha` in a `useEffect` dependency array. When an SSE update arrives after the merge and `branch_stats` becomes `undefined`, the assertion fails on re-render.

## How

1. Replace the non-null assertion with optional chaining: `issue.branch_stats?.head_sha`.
2. In the `useEffect` dependency array, use `issue.branch_stats?.head_sha` so it safely evaluates to `undefined`.
3. When `branch_stats` is `undefined`, render a "branch merged" state instead of the normal merge view — the merge is already complete, so the diff/stats are no longer relevant.
4. Optionally auto-close the merge view after a short delay or on user action.

## Acceptance Criteria

- [ ] MergeView does not crash when `branch_stats` becomes `undefined` mid-render
- [ ] When `branch_stats` is missing, the component shows a "branch merged" indicator instead of crashing
- [ ] No regressions when `branch_stats` is present (normal merge view still works)
- [ ] `make test` passes (lint + tests)
