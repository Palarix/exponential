# Walkthrough: MergeView stops re-rendering the whole diff

## The problem
While MergeView's Files → Uncommitted view is open, the uncommitted diff polls every 3s. Each poll flips the query's `isFetching`, MergeView reads that flip, and MergeView re-renders. A manual Refresh does the same across five queries, and so does any other MergeView state change (the spin, freshness, tab state).

Nothing below MergeView was memoized, so every one of those renders re-rendered every row of every file. On a 1,000-line diff that was a ~190–200ms long task per poll, blocking scrolling and input every 3 seconds, even when nothing had changed. It was found during xpo-ec0480. It also forced the frame-deferral in `useSpinOnce`, because the spin's first frame couldn't paint behind that render.

## Why memo alone was almost enough
The data side was already stable for the unchanged case:
- TanStack's structural sharing hands back the same diff string when a poll returns identical text.
- So `useMemo(() => parseDiffByFile(diff), [diff])` keeps returning the same `Map`.
- `dirtyFilesSet` (memoized on structurally-shared mergeability), `setSelectedFile` and `setFileFilter` are stable too.

So adding memo boundaries to the diff subtree makes an unchanged poll re-render only the cheap MergeView shell.

## Why memo alone wasn't enough for a changed diff
`parseDiffByFile` builds a fresh lines array for every file on every parse. When one file changes, every file gets a new array, and a memoized per-file component would still re-render all of them.

`reuseUnchangedFiles(prev, next)` in `diff-utils.ts` is pure and was tested first. For each file in `next` whose lines equal `prev`'s, it carries over `prev`'s array. If nothing changed at all, it returns `prev` itself: same file set, same order, same lines. It follows `next`'s order, and a reorder counts as a change.

`useStableFileDiffs(map)` in MergeView applies it across renders, wrapping the branch, uncommitted and commit diff maps:

```ts
const [state, setState] = useState(() => ({ input: next, output: next }));
if (state.input !== next) {
  const output = reuseUnchangedFiles(state.output, next);
  setState({ input: next, output });
  return output;
}
return state.output;
```

This is React's documented "store information from previous renders" pattern: setState during render, with React re-running the component immediately before committing. The more obvious "previous value in a ref" isn't allowed: react-hooks v7's `refs` rule forbids reading `ref.current` during render.

## The memo boundaries
- **`memo`** wraps `CommitsTab`, `FileTreeView`, `FilesTab`, `DiffViewer`, `UnifiedDiff` and `SplitDiff`.
- **New `DiffFileCard`:** the per-file card (header, +/- counts, body) used to be inline JSX in `DiffViewer`'s loop. It's extracted into a memoized component with props `file`, `lines`, `collapsed`, `mode` and `onToggle`.
  - `onToggle` is `DiffViewer`'s `useCallback`-stable `toggleFile`.
  - So a card re-renders only when its own lines, collapse state or the unified/split mode change.
- **Per-render work moved into `useMemo`:** the +/- counts in the card, and `addLineNumbers` plus the meta filter in `UnifiedDiff`. `SplitDiff` already memoized `buildSplitLines`.
- **The recursive tree gotcha.** `FileTreeView` renders `<FileTreeView>` for subfolders. Written as `memo(function FileTreeView(...))`, the inner name binds to the *unwrapped* function inside its own body, so recursion would silently bypass memo. The inner function is named `FileTreeLevel`, so the JSX resolves to the memoized const.

## Results
Measured on the same diff for both runs: this branch's ~1,015-row uncommitted diff, made of filler appended to CHANGELOG.md and README.md and reverted afterwards. "Before" ran `main`'s frontend on :5292 and "after" ran this worktree's on :5291, both against the same board API, with long tasks from `PerformanceObserver`.

| Scenario | Before | After |
|---|---|---|
| Idle uncommitted polling, 9s | 200, 191, 188ms | none |
| Manual Refresh, unchanged data | 197ms | none |
| One file changed (README, 309 rows) | 191, 193ms | 85ms, that card only |

## Notes for future readers
- **Keep props into the diff subtree referentially stable.** An inline lambda or object literal passed to `FilesTab`, `DiffViewer` or `DiffFileCard` silently brings back full re-renders. Add it to a `useCallback` or `useMemo` instead.
- **Any new parsed-diff map should go through `useStableFileDiffs`.**
- **Remaining cost scales with the changed file:** ~85ms for 300 rows. If single huge files become a problem, the next step is virtualizing the rows. That was out of scope here, because of the scroll-to-file and sticky-header interactions.
- **`useSpinOnce`'s deferral of the work by one frame is still harmless**, and still helps when a refresh really changes a big diff. Leave it.

## Acceptance criteria
- [x] An uncommitted poll returning an unchanged diff causes no long task (>50ms). Evidence: 9s of polling recorded no long tasks after the change, against 3 tasks of 188–200ms before, on the same diff.
- [x] A manual Refresh with unchanged data causes no long task. Evidence: none after, against 197ms before.
- [x] A change to one file re-renders only that file's card. Evidence: changing README (309 of 1,015 rows) cost 85ms after, against 191 and 193ms before. The cost now scales with the changed file.
- [x] TDD: `reuseUnchangedFiles` was tested first. Evidence: 5 tests covering all unchanged → `prev`, one changed → others reused, add/remove → new map with reuse, reorder → follows `next`, and no `prev` → `next`. They ran red before the implementation.
- [x] No visual or behavioural regression. Evidence, from a browser pass:
  - split/unified switch the table layout
  - collapse all gives 0 rows, expand all gives 1,015
  - a single card collapses while the other keeps 706 rows
  - folder collapse works (5 → 1 → 5 tree entries)
  - the filter "READ" leaves only README.md
  - selecting a file highlights it and scrolls to it
  - the Commits tab diff renders its 3 files
  - the user tophatted ("lgtm")
- [x] `make test` passes. Evidence: exit 0, 511 frontend tests plus Go and lint.
