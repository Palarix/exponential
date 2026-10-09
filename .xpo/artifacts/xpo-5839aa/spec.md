# Spec: Stop MergeView re-rendering the whole diff on polls and refreshes

## What
Make re-renders of the MergeView shell cheap, so they no longer re-render every diff line:
- the uncommitted diff's `isFetching` flipping twice per 3s poll
- a manual Refresh flipping five queries
- any other state change, such as the spin, freshness or a tab hover

When the diff does change, only the files that changed should re-render.

## Why
Measured on the xpo-ec0480 branch (3 files, about 550 diff lines): a Refresh caused long tasks of 98+83ms, and 191+183ms in other runs. While Files → Uncommitted is open, the same full re-render happens on every poll, even when nothing changed. That's about 100–200ms of blocked main thread every 3s, and it grows with the diff. It also forced the frame-deferral workaround in `useSpinOnce` (xpo-ec0480).

## Root cause
- **No memo boundaries.** `FilesTab`, `DiffViewer`, `UnifiedDiff` and `SplitDiff` are plain function components, so every MergeView render re-renders every row of every file. `UnifiedDiff` also recomputes `addLineNumbers` on each render.
- **The data is already stable.** An unchanged poll returns the same diff string (structural sharing), so the `useMemo(parseDiffByFile)` map keeps its identity. `dirtyFilesSet`, `setSelectedFile` and `setFileFilter` are stable too. So memoizing the subtree is enough for the unchanged case.
- **A changed diff gives every file a new lines array.** `parseDiffByFile` builds fresh arrays, so even with memo, every file would re-render when only one changed.

## How
1. **`memo` the diff subtree:**
   - `FilesTab`, `FileTreeView` and `DiffViewer`
   - a new `DiffFileCard`, extracted from `DiffViewer`'s per-file loop (header, +/- counts, body)
   - `UnifiedDiff` and `SplitDiff`
   - Move `addLineNumbers` into a `useMemo`, and the +/- counts into the card's `useMemo`.
   - The props passed into this subtree must be referentially stable: `useCallback` for handlers, plus the existing memoized maps and sets. Fix any inline lambdas or objects on the path.
2. **Per-file identity reuse:** a new pure helper in `diff-utils.ts`, `reuseUnchangedFiles(prev, next): Map<string, string[]>`.
   - For each file in `next` whose lines equal `prev`'s, it keeps `prev`'s array.
   - If nothing changed at all, including the set and order of files, it returns `prev` itself.
   - MergeView applies it to each parsed diff map (branch, uncommitted and commit diffs) through a small `useStableFileDiffs(map)` hook that holds the previous result in a ref, updated after commit.
   - Result: a memoized `DiffFileCard` only re-renders for files whose lines changed.
3. **No behaviour or visual change.** The markup stays the same, and the same states (collapsed cards and dirs, selection, filter, unified/split mode) still work.

## Out of scope
- **Virtualizing very large diffs.** It's a bigger change, with scroll-to-file and sticky-header interactions. Revisit only if a single changed file is still slow.
- **Removing the `useSpinOnce` frame deferral.** It stays: it's harmless, and still useful when the diff really changes.

## Acceptance criteria
- [ ] An uncommitted poll returning an unchanged diff causes no long task (>50ms). Measure with `PerformanceObserver({ type: "longtask" })` on the same branch, before and after.
- [ ] A manual Refresh with unchanged data causes no long task.
- [ ] A change to one file re-renders only that file's card. Measure with the React Profiler or long-task duration: it should scale with the changed file, not the whole diff.
- [ ] TDD: `reuseUnchangedFiles` is tested first. Cases: all unchanged returns `prev`; one file changed reuses the others' arrays; a file added or removed returns a new map with the unchanged arrays reused; order changes are handled.
- [ ] No visual or behavioural regression: unified/split, expand/collapse all, card and dir collapse, file selection and scroll-to-file, the filter, the Commits tab diff.
- [ ] `make test` passes.

## Open questions
1. **Measuring needs a MergeView with real changes.** The board only shows MergeView for an open issue whose branch has commits, so I'd make a WIP checkpoint commit on this branch partway through, like on xpo-ec0480, then measure before and after on the same diff.
2. **Virtualization stays out of scope** unless measurements say otherwise.
