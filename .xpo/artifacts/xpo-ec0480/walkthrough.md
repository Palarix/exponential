# Walkthrough: per-view data fetching on TanStack Query

xpo-de3808 moved the core data App owned (issues, config, inbox and user) onto TanStack Query. Every other fetch in the web UI was still hand-rolled: a `useState` for the data, a `useEffect` that called an `api/client` function, and often a second effect that refetched when `issues` changed, as a crude freshness signal.

That had three costs:
- the same data was requested once per mounted consumer
- views went stale until they remounted, because SSE only refreshed issues and inbox
- every component carried its own loading and error boilerplate

This story finishes the migration. Along the way, MergeView's uncommitted-diff polling got honest staleness feedback and a Refresh button that visibly works. Those parts grew out of review, and they're explained in §5–§7.

```
api/query-keys.ts   queryKeys factory · SSE_INVALIDATED_KEYS · invalidateOnServerEvent   (pure, tested)
api/query-utils.ts  + cyclesFromResponse · firstQueryError                               (pure, tested)
api/queries.ts      + 17 thin hooks (useCycles, useMetrics, useTimeline, useIssueDiff, …)
utils/spinner.ts    spinHoldMs                                                          (pure, tested)
hooks/useSpinner.ts useSpinner(busy, delay) · useSpinOnce(onSettled).track(run)
IssueDetail/diff-utils.ts  + diffFreshness                                              (pure, tested)
```

---

## 1. Query keys are the design (`api/query-keys.ts`)
`queryKeys` moved out of `queries.ts` into its own module, which `queries.ts` re-exports. The keys are where the interesting decisions live, so they are pure and tested without React.

**Prefix nesting is deliberate.** The parameterised keys sit under the prefixes SSE invalidates, so one `invalidateQueries({ queryKey: prefix })` covers them all:
- `["cycles", id, "progress"]` sits under `["cycles"]`
- `["timeline", limit]` sits under `["timeline"]`

**Per-issue keys are version-keyed (user decision).** Each one carries the issue field its data depends on:

| Data | Key ends with | Refetches when |
|---|---|---|
| history, artifacts | `updated_at` | anything on the issue changes |
| local worktree | `status` | `start` creates it or `merge` removes it |
| commits, diff, mergeability | `head_sha` | the branch moves |

When SSE refreshes the issues list and that field changes, the key changes and the data refetches. An unrelated SSE event changes nothing. This matters because the diff and mergeability endpoints shell out to git, and the alternative (invalidate the open issue's data on every event) would rerun them for every comment on any issue. Superseded versions just age out with the default `gcTime`. It also matches exactly what the old `useEffect` deps did, so behaviour didn't change.

**`SSE_INVALIDATED_KEYS`** lists issues, inbox, cycles, metrics, activity and timeline. `invalidateOnServerEvent(qc)` invalidates them all.
- Only *mounted* queries refetch; the rest are just marked stale and refetch when next observed.
- It deliberately leaves out:
  - `user` and `config`: SSE events don't change them
  - `instances`: polled
  - the version-keyed issue data: refreshed through its keys
- The tests seed a real `QueryClient` and assert `isInvalidated` per key. That makes the inclusion and exclusion list a checked contract rather than a convention.

**The server broadcasts on every draft write and from both the DB and git watchers.** That's why invalidating on SSE can replace the old "refetch when `issues` changes" effects in Cycles and Dashboard.

## 2. The hooks (`api/queries.ts`)
The hooks are thin on purpose: a key, a fetcher and, where needed, options. Each one returns what its call site needs: `Cycle[]`, `data ?? null`, or the whole result when the component needs `isPending` or `refetch`. That keeps the component diffs small.

The empty defaults are module-level constants (`NO_DAYS`, `NO_ACTIVITY`…), so a not-yet-loaded result has a stable identity and doesn't break downstream `useMemo`s.

Non-default options, each matching old behaviour:
- **`useMetrics`/`useActivity`:** `refetchInterval: 30_000` plus `refetchOnWindowFocus: true`. The user chose to keep polling, because time-based metrics such as aging and cycle time change without events.
- **`useInstances`:** 10s polling plus focus, as before.
- **`useTimeline(limit)`:** `placeholderData: keepPreviousData`, so "Load more" (a new key) keeps the list on screen instead of flashing the spinner.
- **`useIssueHistory`, `useLocalWorktree`:** `keepPreviousData` too, matching the old retained `useState`.
- **`useCommitDiff(id, sha | null)`, `useCommitDetail(sha, open)`:** `enabled` queries. Commit content is immutable, so a reopened commit diff or popover is served from cache. Previously this was imperative and only cached per component instance.
- **`useFetchArtifact()`:** returns `(issue, file) => qc.fetchQuery(…)` for the download button. A click is still imperative, but it shares the cache with the spec and walkthrough tabs.

`createQueryClient`'s defaults (`retry: false`, no refetch on focus, `staleTime: Infinity`) are untouched, for the reasons in the xpo-de3808 walkthrough.

**An import-cycle gotcha:** `queries.ts` imports `useToast` from `components/ui/ToastContext`, not from the `components/ui` barrel. ContextMenu lives in `ui/` and now imports `queries`. Going through the barrel would create a cycle (`ui/index → ContextMenu → queries → ui/index`) that only works by accident of evaluation order.

## 3. Per-component migration
| Component | Change |
|---|---|
| Cycles | Two effects collapsed into `useCycles()`. The second effect refetched on every `issues` change and is now covered by SSE. `loading` = `isPending`. ProgressChart uses `useCycleProgress`. |
| Backlog | `cycleMap` is a `useMemo` over `useCycles().cycles`. |
| ContextMenu, PropertySidebar | `useCycles()`. One request is shared with whatever view is open. |
| Dashboard | `useMetrics()`/`useActivity()` replace the interval-plus-focus effect. |
| Sidebar | `useInstances()` replaces the interval-plus-focus effect. |
| Timeline | `useTimeline(limit)`. The commit SHA popover uses `useCommitDetail(sha, open)`. |
| IssueDetail | The `artifactContent` map and its reset effects become `useArtifact(issue, file, tab !== "details")`. The worktree effect becomes `useLocalWorktree(issue)`. |
| ActivityTimeline | `useIssueHistory(issue)` |
| ArtifactList | Download goes through `useFetchArtifact`. |
| MergeView | Five queries replace one `Promise.all`; details below. |

Render branches (spinners, empty states, "errors mean empty data") were kept as they were, except for MergeView's uncommitted diff.

**MergeView's queries:** commits, the branch diff, mergeability, the walkthrough artifact and the uncommitted diff are now separate queries.
- `loading` is true while any of them is pending.
- `error` comes from `firstQueryError([...])`, the same helper the Refresh feedback uses. The uncommitted query only counts when it has *no* data, so a failed poll never swaps the diff for the error screen.

## 4. Uncommitted diff: polling and entry
The uncommitted diff polls every 3s, only on the Files tab with the Uncommitted scope: `refetchInterval: pollUncommitted ? 3000 : false`.

**TanStack gotcha:** turning `refetchInterval` on doesn't fetch; it only starts the timer. The old code refreshed immediately on entering the view.
- **Fix:** a `showView(tab, scope)` function replaces the direct tab and scope setters. On entering the uncommitted view it records `enteredAt` and calls `refetch()`.
- **Why not an effect:** I tried a `useEffect` first. Doing the refetch in the event handler needs no effect at all.

## 5. Staleness feedback (review: "we see an outdated diff")
The user pointed out two ways the old behaviour, which I had preserved, showed outdated content silently:
- on entry, the diff fetched when MergeView opened stayed up until the refetch landed
- after a failed poll, the last diff stayed up indefinitely

The user chose **option B**: keep the cached diff, so nothing blanks, but make staleness visible.

`diffFreshness({ dataUpdatedAt, errorUpdatedAt, enteredAt })` is pure and tested:
- **`failed`** if the latest error is newer than the latest data. A failed entry refetch counts as failed, not refreshing.
- **else `refreshing`** if the data predates entering the view.
- **else `fresh`.**

Routine polls after entry always produce data newer than `enteredAt`, so they never read as "refreshing". That's what keeps the indicator from flickering every 3s.

**`failed` → `StaleDiffNotice`:** a warning icon plus "Couldn't refresh, showing changes as of HH:MM", placed to the left of the Refresh button. Clicking Refresh retries.

How it got there, through review:
1. A full-width banner in ViewContainer's `header` slot, with its own Retry button.
2. A "Refreshing…" text label next to the scope toggle. The toggle is absolutely centered (`left-1/2 -translate-x-1/2`), so any sibling in its container widens the container and shifts the toggle.
3. The rule since then: **status text lives after the `flex-1` spacer, next to the Refresh button, where it grows into empty space.**
4. The banner was replaced by the compact inline notice once the Refresh button became the retry.

## 6. A Refresh button that visibly spins (review: "too fast locally")
The text label became a spinning icon. Three separate problems hid the spin.

**1. Local fetches take milliseconds.**
`spinHoldMs({ busyMs, visibleMs, delay, period })` is pure and tested. It returns how long to keep spinning after the work settles: `null` if the work finished inside `delay` and never showed, otherwise enough to finish the current rotation. The spin always ends on a whole turn, never mid-rotation.

There are two hooks:
- **`useSpinner(busy, 150)`** for the automatic entry refetch. It appears only if the refetch takes over 150ms, so fast fetches show nothing rather than a one-frame flicker.
- **`useSpinOnce().track(run)`** for clicks. It always gives at least one full turn (1s, Tailwind's `animate-spin` period).

**2. The re-render blocked the first frame.**
A trace showed the spin class applied 4ms after the click. Two long tasks followed (98ms and 83ms, more in other runs): MergeView re-rendering the entire diff because `isFetching` flipped. The first spin frame couldn't paint until about 190ms in, and a stop timed from the click landed about 0.2 turns short.

Two changes fixed it:
- **Spin state lives in a small `RefreshButton`,** so toggling it doesn't re-render MergeView.
- **`track(run)` takes the work as a function** and calls it only after the first spinning frame has painted (`requestAnimationFrame` → `setTimeout`). It measures rotations from that frame's timestamp, which is when the CSS animation starts.

Verified on the animation's own clock: `getAnimations()[0].currentTime` read exactly 1000 at the stop.

The underlying cost (a full diff re-render on every poll and refresh, which predates this story) is filed as **xpo-5839aa**.

**3. Reduce Motion.**
The user's macOS has Reduce Motion on (`com.apple.universalaccess reduceMotion = 1`), and the icon used `motion-safe:animate-spin`, so it never animated for them. The user decided the spinner is status feedback, not decoration. It now uses plain `animate-spin`, like the app's other loading spinners; this was the app's only reduced-motion usage.

**Testing note:** the Chrome that the DevTools MCP drives also reports `prefers-reduced-motion: reduce`. I had to force the animation with an injected style before switching to plain `animate-spin`.

## 7. Outcome feedback (review: "success / fail feedback when the spin ends")
The user chose the icon morph plus a toast on failure.
- `refreshData` resolves to `firstQueryError` of the five refetch results. `refetch()` never throws; it resolves with a result that carries `status` and `error`.
- `useSpinOnce(onSettled)` reports the last run's result when the spin actually stops, so the feedback lines up with the end of the rotation rather than with the network.

`RefreshButton` then shows one of:
- **Success:** a green `Check` for 1.5s.
- **Failure:** a red `AlertTriangle` for 1.5s, plus a toast: "Couldn't refresh: <reason>".

All three icons are the same size, so the layout never moves. A new click cancels a showing flash. Automatic refreshes report nothing, so polls stay silent.

`firstQueryError` is pure and tested: it returns the first failure's `ApiError` message, "Failed to load" for non-API errors, or null, and it skips `false` entries.

---

## Non-obvious things for future readers
- **Don't add the version-keyed issue data to `SSE_INVALIDATED_KEYS`.** It would rerun git-backed endpoints on every event. Its keys already change when the issue does.
- **`refetchInterval` doesn't fetch on enable.** If you need "fresh on entry", refetch in the event that enters.
- **Nothing should sit next to MergeView's centered scope toggle.** Status UI goes after the `flex-1` spacer.
- **`useSpinOnce.track` defers the work by about one frame on purpose.** Don't "optimise" that away while MergeView's render is expensive (xpo-5839aa).
- **react-hooks v7 is strict:** no synchronous `setState` in effects, no `Date.now()` in render, no ref reads in render. That's why spin and timer state only change inside timers and event handlers, and the timing logic is a pure function.

## Follow-ups
- **xpo-5839aa:** MergeView re-renders the whole diff on every poll or refresh (100–200ms long tasks).
- **xpo-6e757d** (pre-existing): mutations, and removing `onRefresh`.
- **Offered, not done:** the scope toggle still shifts about 2px when switching, because the active option is `font-medium`. This predates this story.

---

## Acceptance criteria
- [x] No component calls an `api/client` fetch function inside `useEffect`. Evidence: the grep over `web/src` for `fetchX(` outside `client.ts` and `queries.ts` finds only `fetchArtifact(...)` in ArtifactList's click handler, which is the `useFetchArtifact` hook's function, not an effect.
- [x] An SSE event refreshes any mounted cycles, metrics, activity or timeline data. Evidence: `invalidateOnServerEvent` tests (3). In the browser, with Timeline mounted, posting an issue comment refetched `/api/timeline` within 3s, and the new entry appeared (40 → 41 events). Unmounted views were not requested.
- [x] `fetchCycles` is requested once per invalidation, not once per mounted consumer. Evidence: navigating Cycles → Issues → Dashboard → Timeline made exactly one `/api/cycles` request, shared by Cycles and Backlog. ContextMenu and PropertySidebar use the same `useCycles` key, so they share that cache entry; I didn't capture a separate context-menu trace.
- [x] Dashboard still polls every 30s and on focus; Sidebar instances poll every 10s and on focus. Evidence: `DASHBOARD_POLL` and `useInstances` options. In the browser, `/api/instances` was re-requested after about 10s.
- [x] Timeline "Load more" keeps existing entries visible while it loads. Evidence: `useTimeline` uses `placeholderData: keepPreviousData`, and `loading` is `isPending`, which stays false while placeholder data shows. On the user's tophat list.
- [x] MergeView: the uncommitted diff polls every 3s only on Files + Uncommitted; Refresh reloads everything; a reopened commit diff is cached. Evidence: in the browser, a request log showed polls about 3s apart only in that view. `refreshData` refetches all five queries. `useCommitDiff` is keyed by an immutable sha.
- [x] MergeView: entering the uncommitted scope refetches immediately. The icon spins only if that takes over 150ms and then completes its rotation; a manual Refresh always spins at least one full turn, including with Reduce Motion on; routine polls never spin it. Evidence:
  - fast entry: never spun
  - 400ms entry: spun 993ms
  - manual click at local speed: animation running at 6ms, `currentTime` exactly 1000 at the stop, in a browser with `prefers-reduced-motion: reduce`
  - polls sampled every 100ms for 4s: never spun
- [x] MergeView: a failed uncommitted fetch keeps the diff and shows "Couldn't refresh, showing changes as of HH:MM" beside Refresh, which retries. The next success clears it, and the layout never shifts. Evidence: with a forced 500, the notice read "…as of 01:25 PM" and about 550 diff cells stayed rendered. Refresh after recovery cleared it. The Refresh button stayed at x=1809 in every state.
- [x] IssueDetail: spec and walkthrough tabs, history, worktree and git data refresh when `updated_at`, `status` or `head_sha` change. Evidence: 5 `queryKeys` tests covering key changes and non-changes per field. In the browser, history, worktree (VSCode link) and the spec tab loaded, and MergeView reloaded after `head_sha` moved with the checkpoint commit.
- [x] MergeView: when a manual Refresh's spin ends, the icon flashes a green check, or a red alert plus a "Couldn't refresh: <reason>" toast, then returns. Evidence:
  - success trace: spin → check at 1017ms → idle at 2518ms
  - failure trace: spin → alert at 1108ms → idle at 2515ms, toast "Couldn't refresh: boom" (screenshot)
  - the user confirmed it works
- [x] TDD: `cyclesFromResponse`, `diffFreshness`, `spinHoldMs`, `firstQueryError` and `SSE_INVALIDATED_KEYS` were tested first. Evidence: each test file was run red before the implementation. That's 26 new tests across `query-keys.test.ts`, `query-utils.test.ts`, `MergeView.test.ts` and `spinner.test.ts`.
- [x] `make test` passes, with no regression. Evidence: exit 0, with 506 frontend tests plus the Go suite and lint. The user tophatted ("works now").
