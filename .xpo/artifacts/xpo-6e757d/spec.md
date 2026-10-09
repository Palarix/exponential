# Convert issue mutations to TanStack `useMutation` and drop `onRefresh` props

## What
Move every write the web app makes into `useMutation` hooks in a new `api/mutations.ts`. Each hook owns its cache work: optimistic patch, rollback, invalidation and the error toast. Components call the hooks directly, so the `onRefresh`, `patchIssue` and `onConfigLabelsChange` props, and the `usePatchIssue`/`useSetConfigLabels` hooks, go away.

## Why
After xpo-de3808 and xpo-ec0480, reads go through TanStack Query, but writes are still hand-wired:
- About 30 call sites run `addDraft`/`createIssue`/… and then call `onRefresh()`. That `onRefresh` is threaded through about 10 components.
- Optimistic updates (`patchIssue`) are only half done. They never roll back, so a failed write leaves the UI showing a value the server rejected until the next SSE event.
- Error handling is uneven: some sites toast, some log to the console, and many just leave an unhandled promise.

## How

### Module layout (same split as `query-keys.ts` and `queries.ts`)
- **`api/query-utils.ts`** (pure, tested) gets:
  - `applyIssuePatches(issues, patches)` returns `{ next, undo }`. `undo` is the list of `{ issueId, patch }` holding each field's *previous* value.
  - Rollback calls `applyIssuePatches(current, undo)`. It reverts only the fields this mutation touched, so a later concurrent optimistic edit isn't overwritten by an older snapshot.
  - It also gets `renameLabel` and `removeLabel` for the label maps.
- **`api/mutation-options.ts`** (tested against a real `QueryClient` with a fake `MutationApi`) holds option factories such as `updateIssuesOptions(qc, api?)` → `{ mutationKey, mutationFn, onMutate, onError, onSettled }`. They're testable without React, the way `invalidateOnServerEvent` is.
- **`api/mutations.ts`** holds thin hooks: `useMutation(useWithErrorToast(factory(qc)))`.

### Hooks
| Hook | Calls | Optimistic | Settled → invalidate |
|---|---|---|---|
| `useUpdateIssues()` (and `useUpdateIssue()` for a single issue) | `addDraft(id,"UPDATE",patch)` for each item, in order | yes, `applyIssuePatches` | `issues` |
| `useCreateIssue()` | `createIssue` + a follow-up `UPDATE` (status/estimate) when needed; resolves to the id | no | `issues` |
| `useDeleteIssue()` | `addDraft(id,"DELETE",{cascade})` | no | `issues` |
| `useAddComment()` | `addDraft(id,"COMMENT",…)` | no | `issues` |
| `useStartWork()` | `startWork` | no | `issues` |
| `useMergeIssue()` | `mergeIssue` | no | `issues` |
| `useAddLabel()` / `useUpdateLabel()` / `useDeleteLabel()` | config label endpoints | `setQueryData(config)` on success | `config`; also `issues` for rename/delete |

- **Concurrency:** `onMutate` calls `cancelQueries(issues)` so an in-flight refetch can't overwrite the optimistic state. Every issue mutation shares the `["issues"]` mutation key, and `onSettled` invalidates only when `qc.isMutating({ mutationKey: ["issues"] })` is 1, meaning this is the last one in flight. A burst of edits (a cascade, or rapid clicks) then causes one refetch instead of flickering through the intermediate states.
- **Errors:** the hooks toast `Couldn't save: <message>` (variant error). Call sites drop their own error handling (IssueDetail's catch-and-toast, the `console.error`s), so nothing toasts twice. Success toasts and UI resets stay at the call sites, as per-call `mutate(vars, { onSuccess, onSettled })` callbacks.
- **Board drag:** `onDragEnd` calls `updateIssue(id, { sort_order, status? })` and clears `isDraggingRef` right away. The optimistic patch changes `issues`, which rebuilds `containers`. On failure, rollback restores `issues` and the card snaps back to its original column.
- **Backlog:** quick edits (status, estimate, priority, labels), the status cascade and the three drag-drop handlers all go through `useUpdateIssue(s)`. They become optimistic (today they wait for the server).
- **IssueDetail:** `saveDraft` is split into `saveUpdate(patch, alsoUpdate?)` and `saveComment`. `saveUpdate` goes through `useUpdateIssues`, so a PropertySidebar status cascade is a single mutation. The local `optimisticTitle`/`optimisticDescription` state is removed: the cache patch already covers it, and rollback replaces its stale-until-SSE behavior.
- **Merge:** `useMergeIssue` doesn't toast, because MergeView shows the error inline in its confirm dialog. `merging` and `commentSaving` come from each mutation's `isPending`.

### Removals
- Props: `onRefresh` (IssueDetail, PropertySidebar, SubIssuesTable, ContextMenu), `patchIssue` (ContextMenu), and `onCreated`'s `await refresh()` in App.
- Props: `onConfigLabelsChange` everywhere (see Decisions).
- Hooks: `usePatchIssue`, `useSetConfigLabels`, and the `patchIssueList` helper (replaced by `applyIssuePatches`). `useRefreshIssues` stays only for `ErrorBoundary onReset` in ViewRouter.
- `MergeView`'s `RefreshButton onRefresh` is unrelated (it's the diff refetch) and stays.

## Acceptance criteria
- [ ] No component accepts `onRefresh`, `patchIssue` or `onConfigLabelsChange` (`grep` over `components/` finds only MergeView's internal `RefreshButton`).
- [ ] No component imports `addDraft`, `createIssue`, `startWork`, `mergeIssue` or the `*ConfigLabel` functions from `api/client`. They're only called from `api/mutation-options.ts`.
- [ ] Board drag: when the write fails, the card returns to its original column and position and an error toast appears.
- [ ] Backlog quick edits (status/estimate/priority/labels) show up immediately and roll back with an error toast on failure.
- [ ] Context menu, Board quick edits and MyIssues quick edits roll back the same way.
- [ ] A Backlog status cascade (parent + children) triggers one issues refetch, not one per child.
- [ ] TDD: `applyIssuePatches` and the option factories (optimistic apply, field-level rollback, invalidate-on-last-settle) have tests written before the implementation.
- [ ] `make test` passes.

## Decisions (confirmed with user)
1. **`onConfigLabelsChange` props are removed too.** The label mutation hooks set the config cache themselves. LabelPicker takes a `canCreateLabels` boolean (default `false`) instead of checking whether the callback exists. `useSetConfigLabels` is deleted. AC: no component accepts `onConfigLabelsChange`.
2. **Delete is not optimistic.** `useDeleteIssue` waits for the server and then invalidates.

## Implementation notes (found while building)
- Removing the try/finally blocks let the React Compiler lint analyze Board and IssueDetail for the first time. That flagged code that predates this issue, which is fixed here without suppressions:
  - **Board:** the `issues → containers` effect became the "adjust state when a prop changes" pattern. `syncedIssues` tracks `issues` during a drag too, and columns are only rebuilt when not dragging, so a drop never flashes stale columns. `containersRef` is synced in a layout effect, and `openPopoverRef` in an effect.
  - **IssueDetail:** the reset when `issue.id` changes became the same render-time pattern.
- **Backlog indicator drop:** `status` from `getRowStatusGroup` can be `null`. The code used to send `status: null` in that case; it now leaves `status` out of the patch.
