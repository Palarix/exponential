# Walkthrough: issue mutations via TanStack `useMutation`

## What and why
xpo-de3808 and xpo-ec0480 moved all the web UI's **reads** onto TanStack Query. The **writes** were still hand-wired. A component called `addDraft`/`createIssue`/… itself and then called an `onRefresh()` prop, which had been threaded through about 10 components. Some views also patched the cache optimistically through a `patchIssue` prop, but that patch never rolled back. So when a write failed, the UI kept showing a value the server had rejected until the next SSE event. Error handling varied by call site: some toasted, some logged to the console, and most didn't handle the error at all.

This issue moves every write into a mutation hook that owns its own cache work. That covers the optimistic patch, the rollback, invalidation and the error toast. Components just call the hook.

## Layout: three layers, same split as the reads

```
api/query-utils.ts       applyIssuePatches · renameLabel · removeLabel                     (pure, tested)
api/mutation-options.ts  updateIssuesOptions · createIssueOptions · … · label options      (tested with a real QueryClient)
api/mutations.ts         useUpdateIssues/useUpdateIssue · useCreateIssue · useDeleteIssue ·
                         useAddComment · useStartWork · useMergeIssue · useAdd/Update/DeleteLabel
```

- **`mutation-options.ts`** holds the logic. Each factory takes `(qc, api = clientApi)` and returns plain `MutationOptions`. `MutationApi` is the set of client functions mutations call. Tests pass in `vi.fn()` fakes and run the full lifecycle with `qc.getMutationCache().build(qc, opts).execute(vars)`, the same path `useMutation().mutateAsync` takes, so no React or DOM is needed. This follows the precedent of `invalidateOnServerEvent` in `query-keys.ts`.
- **`mutations.ts`** is thin. `useMutation(useWithErrorToast(factory(qc)))` chains a `Couldn't save: <message>` toast after the factory's own `onError` rollback. `useMergeIssue` is the one hook without a toast, because MergeView shows merge errors inline in its confirm dialog.

## The optimistic update and why rollback is field-level
`updateIssuesOptions` takes a list of `{ issueId, patch }`:

1. `onMutate` calls `cancelQueries(issues)` so an in-flight refetch can't land on top of the optimistic state. It then calls `applyIssuePatches(cache, patches)` and returns `{ undo }` as the mutation context.
2. `mutationFn` sends one `UPDATE` draft per patch, in order. A cascade is therefore *one* mutation.
3. `onError` reapplies `undo` to the *current* cache.

`undo` holds the **previous values of only the patched fields**, with entries in reverse order. The usual TanStack recipe snapshots the whole list and restores it, but that breaks with concurrent edits. Say you change a status, then an estimate, and the status write fails: restoring the snapshot would also wipe out the estimate. With field-level undo, only the status reverts. The test "keeps a concurrent edit to another field when rolling back" covers exactly that case.

## One refetch per burst
Every issue-changing mutation uses the same `mutationKey: ["issues"]`, including create, delete, comment, start and merge. `onSettled` calls `invalidateIssuesWhenIdle`, which refetches only if `qc.isMutating({ mutationKey })` is 1.

One non-obvious detail: TanStack v5 runs `onSettled` *before* it dispatches the success or error state, so the settling mutation still counts as pending. That's why "the last one" is a count of 1, not 0. I checked this against `query-core/build/modern/mutation.js`, and the test "invalidates issues once, after the last concurrent issue mutation settles" locks it in.

`onSettled` returns the invalidation promise, and TanStack awaits it. So per-call `onSuccess` callbacks, such as App's "Issue created" toast, fire after the refetched list is in the cache. That's why App's `onCreated` no longer needs `await refresh()`.

## Labels
The `addLabel`, `updateLabel` and `deleteLabel` options write the new label map into the `config` cache on success, then invalidate `config`. Rename and delete also invalidate `issues`, because the server rewrites the labels on issues. This made `useSetConfigLabels` and the `onConfigLabelsChange` prop chain unnecessary (the user confirmed removing them in this issue).

LabelPicker used to decide whether to offer "Create label" by checking whether that callback existed. It now takes an explicit `canCreateLabels` boolean, as does ContextMenu, and calls `useAddLabel` itself. Each call site that used to pass the callback now passes `canCreateLabels`. The Board's quick-edit label popover never passed it, so it still can't create labels.

## Call-site pattern
Components use `mutate(vars, { onSuccess, onSettled })` and never `mutateAsync`, so nothing leaves an unhandled rejection. Success toasts and UI resets stay at the call site because their wording is context-specific:

```ts
updateIssue(issueId, { estimate }, { onSuccess: () => showToast(`Estimate set to ${estimate || "none"}`) });
```

`isPending` replaces hand-rolled loading flags: `saving` in NewIssueModal, `starting` in PropertySidebar, and `merging`/`commentSaving` in MergeView.

### IssueDetail
`saveDraft(type, payload)` was split into `saveUpdate(patch, alsoUpdate?)` and `saveComment(comment)`. The extra `alsoUpdate` patches let PropertySidebar's "Update all" cascade go out as one mutation together with the parent. The toast wording moved into a small `updateToast(patch)` function.

The local `optimisticTitle`/`optimisticDescription` state is **removed**. The cache patch now does that job, and unlike the old state it rolls back on failure.

### Board drag
`handleDragEnd` is no longer async. It computes the patch, clears `isDraggingRef`, and calls `updateIssue`. The optimistic patch changes `issues`, which rebuilds the columns; a rollback rebuilds them back, which is the card snapping back.

## The React Compiler lint surprise
Removing the `try/finally` blocks let the React Compiler lint analyze `Board` and `IssueDetail` for the first time. It had been bailing out on them before. It then flagged older code, which I fixed with real changes rather than suppressions:

- **Board: syncing columns from issues.** The old `useEffect(() => setContainers(buildContainers(issues)))` is now the "adjust state when a prop changes" pattern, using `syncedIssues` state. `syncedIssues` advances even mid-drag, but columns are only rebuilt when no drag is active. The ordering matters: `mutate()` reaches `onMutate` only after a microtask, and React flushes the drop's `setActiveId(null)` first. If `syncedIssues` stood still during a drag, that flush would rebuild the columns from pre-drop `issues` and flash the card back for a frame. Because it keeps up, the only rebuild after a drop is the one the optimistic patch causes. `containersRef` is now kept in sync by a layout effect instead of being written during render.
- **Board:** `openPopoverRef` is updated in an effect, and `setOpenPopover` was added to two `useCallback` dependency lists the compiler inferred.
- **IssueDetail:** the effect that reset per-issue UI state on `issue.id` uses the same render-time pattern.

## Small fix along the way
On a Backlog indicator drop, `getRowStatusGroup()` can return `null`. The old code then sent `status: null` in the draft; the patch now leaves `status` out.

## Acceptance criteria
- [x] No component accepts `onRefresh`, `patchIssue` or `onConfigLabelsChange`. Grepping `web/src` for `onRefresh|patchIssue|onConfigLabelsChange` finds only MergeView's internal `RefreshButton`, which handles the diff refetch.
- [x] No component imports `addDraft`, `createIssue`, `startWork`, `mergeIssue` or the `*ConfigLabel` functions. A word grep over `components/` and `App.tsx` finds no such imports; only `api/mutation-options.ts` imports them.
- [x] Board drag rolls back on failure. The drop's patch goes through `updateIssuesOptions`, which is covered by "rolls back the patched fields when the write fails", and columns follow `issues`. The user tested it in the browser.
- [x] Backlog quick edits are optimistic and roll back with an error toast. All four use `useUpdateIssue`, and the toast comes from `useWithErrorToast`. The user tested it in the browser.
- [x] Context menu, Board quick edits and MyIssues quick edits roll back the same way. They use the same `useUpdateIssue`; MyIssues goes through ContextMenu.
- [x] A cascade triggers one refetch. Backlog's `applyStatusChange` and PropertySidebar's "Update all" each send one `updateIssues([...])`, covered by "invalidates issues once, after the last concurrent issue mutation settles".
- [x] TDD: `query-utils.test.ts` (8 new tests) and `mutation-options.test.ts` (16 tests) were written first and run red before the implementation.
- [x] `make test` passes: 533 vitest tests, `go vet` and eslint are clean, and the Go suite passes (exit 0). `make frontend` builds.
