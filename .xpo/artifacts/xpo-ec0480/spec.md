# Spec: Migrate per-view data fetching to TanStack Query

## What
Replace every `useState` + `useEffect` fetch of an `api/client` function with a `useQuery` hook in `api/queries.ts`, keyed by a shared `queryKeys` factory. Extend SSE invalidation so cycles, metrics, activity and timeline stay fresh while mounted.

Builds on xpo-de3808 (`createQueryClient` defaults: `retry: false`, `refetchOnWindowFocus: false`, `staleTime: Infinity`; these stay).

## Inventory (today → after)

| Call site | Today | After |
|---|---|---|
| `fetchCycles` ×5: Cycles (×2), Backlog, ContextMenu, PropertySidebar | mount fetch; Cycles also refetches on every `issues` change | `useCycles()` → `Cycle[]` (`[]` when disabled or loading), key `["cycles"]`. Backlog derives its `cycleMap` with `useMemo`. Cycles' `loading` = `isPending`. |
| `fetchCycleProgress` (Cycles ProgressChart) | fetch per `cycleId` | `useCycleProgress(id)`, key `["cycles", id, "progress"]` (prefix-invalidated with cycles) |
| `fetchMetrics`, `fetchActivity` (Dashboard) | refetch on `issues` change + 30s interval + window focus | `useMetrics()`, `useActivity()`, keys `["metrics"]`, `["activity"]`, with `refetchInterval: 30_000`, `refetchOnWindowFocus: true` |
| `fetchTimeline(limit)` | fetch on mount / limit change | `useTimeline(limit)`, key `["timeline", limit]`, `placeholderData: keepPreviousData` so "Load more" doesn't flash the spinner |
| `fetchInstances` (Sidebar) | 10s interval + focus | `useInstances()`, key `["instances"]`, `refetchInterval: 10_000`, `refetchOnWindowFocus: true` |
| `fetchIssueHistory` (ActivityTimeline) | on `issue.id`, `issue.updated_at` | `useIssueHistory(issue)` |
| `fetchArtifactContent` (IssueDetail spec/walkthrough tabs, MergeView walkthrough) | local `artifactContent` map, cleared on `updated_at` | `useArtifact(issue, filename, { enabled })`, shared by IssueDetail and MergeView |
| `fetchLocalWorktree` (IssueDetail) | on `issue.id`, `issue.status` | `useLocalWorktree(issue)` |
| `fetchIssueCommits`, `fetchIssueDiff`, `fetchMergeability` (MergeView) | one `Promise.all` on `issue.id`, `head_sha` | `useIssueCommits`, `useIssueDiff(issue, scope)`, `useMergeability`. MergeView's loading = any pending; error = first error. The Refresh button refetches all of them. |
| `fetchIssueDiff(…, "uncommitted")` poll (MergeView) | 3s interval while Files tab + uncommitted scope; failures silently ignored | same query with `refetchInterval: polling ? 3000 : false`, refetched immediately on entering that view; staleness made visible (see **Uncommitted diff freshness**) |
| `fetchCommitDiff` (MergeView, on click) | imperative | `useCommitDiff(issueId, sha \| null)`, `enabled: !!sha`, key `["issue", id, "commit", sha]` (immutable, so cached) |
| `fetchCommitDetail` (Timeline popover, on open) | imperative, cached in component state | `useCommitDetail(sha, { enabled: open })`, key `["commit", sha]` |
| `downloadArtifact` (ArtifactList, on click) | imperative | stays imperative but goes through `qc.fetchQuery` on the artifact key, sharing the cache |

## Freshness

**SSE-invalidated** (`useInvalidateOnServerEvent`): `issues`, `inbox`, `cycles` (covers progress), `metrics`, `activity`, `timeline`. These are kept as a tested constant `SSE_INVALIDATED_KEYS`. Only mounted queries refetch; the rest are just marked stale. **Not** invalidated: `user`, `config` and `instances` (polled).

The server broadcasts on every draft write and from the DB/git watchers, so this replaces the "refetch when `issues` changes" effects in Cycles and Dashboard.

**IssueDetail data** is keyed by the issue field it depends on today, so it refetches exactly when that field changes. That happens when the issues query refreshes through SSE, and not on every unrelated SSE event. The git diff calls in particular are too expensive to rerun on every event.
- history and artifacts: `["issue", id, "history" | "artifact", …, updated_at]`
- worktree: `["issue", id, "worktree", status]`
- commits, diff, mergeability: `["issue", id, "commits" | "diff" | "mergeability", …, head_sha]`

Superseded versions age out with the default `gcTime`.

### Uncommitted diff freshness (MergeView)
The old code could show an outdated uncommitted diff with no signal that it was outdated:
- when you entered the view, it showed the diff fetched when MergeView opened until the first poll returned
- after a failed poll, the last diff stayed up silently

**Decision (user, option B):** keep showing the cached diff so nothing blanks or flashes, but always make its staleness visible.

- **On entering** the uncommitted scope on the Files tab, the query refetches immediately.
- **The Refresh button's icon spins while a refresh is in flight.** That covers two cases: the refetch on entry, until a fetch started after entering succeeds, and a manual Refresh click, until all of its refetches settle. Routine 3s polls never spin it, so it doesn't flicker. Use plain `animate-spin`, not `motion-safe:`. The spin is status feedback rather than decorative motion, so it runs under reduced motion too (user decision), like the app's other loading spinners. With `motion-safe:`, users with Reduce Motion on never saw it. There is no text label: an earlier "Refreshing…" label shifted the centered scope toggle (user feedback), and a spinning icon says the same thing without taking up space.
  - **Spin timing:** local fetches finish in milliseconds, too fast to see (user feedback).
    - A **manual Refresh** always spins at least one full rotation (1s, the `animate-spin` period), so a click always gets feedback.
    - The **automatic entry refetch** spins only if it takes longer than 150ms, so fast fetches show nothing rather than a flicker.
    - Once the spin is visible, it always ends on a whole rotation, so the icon never stops mid-turn.
    - The rule is a pure, tested helper, `spinHoldMs({ busyMs, visibleMs, delay, period })`: ms to keep spinning after the work settles, or `null` for "never shown".
    - Two hooks in `hooks/useSpinner.ts` wire it to timers:
      - `useSpinner(busy, delay)` for the automatic entry refetch.
      - `useSpinOnce().track(run)` for clicks. It starts the spin in the click's render, and runs the work only after that first spinning frame has painted (`requestAnimationFrame` → `setTimeout`). It measures rotations from that frame, so the stop lands on a whole turn.
      - Why the deferral: the refetch re-renders the whole diff (~100–200ms of main-thread work on a large diff, see xpo-5839aa), which otherwise blocks the first spin frame.
    - The spin state lives in a small `RefreshButton` component, so toggling it re-renders only the button, not MergeView.
  - **Outcome feedback when a manual Refresh's spin ends** (user decision):
    - **Success:** the icon briefly becomes a green check (~1.5s), then the refresh icon again.
    - **Failure:** the icon briefly becomes a red alert (~1.5s), and an error toast reads **"Couldn't refresh: <reason>"**. The reason is the first failed query's `ApiError` message, or "Failed to load".
    - The feedback is the same size as the icon, so the layout never moves. A new click cancels a showing flash.
    - Automatic refreshes (entry refetch, polls) show no outcome feedback. A failed poll is still covered by the inline "Couldn't refresh…" notice.
    - A pure, tested helper turns the refetch results into the outcome: `firstQueryError(results): string | null`, in `query-utils.ts`. `useSpinOnce(onSettled)` reports the run's result when the spin stops, so the feedback lines up with the end of the rotation.
- **When a fetch fails** (the latest error is newer than the latest data), the diff stays and a compact warning shows to the left of the Refresh button: a warning icon and **"Couldn't refresh, showing changes as of HH:MM"**. The Refresh button is the retry, so there is no banner and no separate Retry button. The warning clears on the next successful fetch, whether from polling or Refresh. It sits in the normal flow after the `flex-1` spacer, so it never moves the centered toggle.
- **No data at all:** if the first fetch fails, the existing MergeView error screen is used, as today.
- The state comes from a pure, tested helper:

  ```ts
  diffFreshness({ dataUpdatedAt, errorUpdatedAt, enteredAt }): "fresh" | "refreshing" | "failed"
  ```

  - `failed` if `errorUpdatedAt > dataUpdatedAt`
  - else `refreshing` if `dataUpdatedAt < enteredAt`
  - else `fresh`

## Structure
- `queryKeys` gains factory functions for the parameterised keys.
- Pure, tested helpers go in `query-utils.ts`, such as `cyclesFromResponse`. Hooks stay thin wiring in `queries.ts`.
- Components lose their `useState`/`useEffect` fetch boilerplate. Their render branches stay the same: loading spinners, empty states and silent `catch` behaviour, where an error means empty data. The one exception is the uncommitted diff, covered above.

## Acceptance criteria
- [ ] No component calls an `api/client` fetch function inside `useEffect`. Check: grep.
- [ ] An SSE event refreshes any mounted cycles, metrics, activity or timeline data.
- [ ] `fetchCycles` is requested once per invalidation, not once per mounted consumer. With Backlog and the context menu open: one request.
- [ ] Dashboard still polls every 30s and on focus; Sidebar instances still poll every 10s and on focus.
- [ ] Timeline "Load more" keeps the existing entries visible while it loads.
- [ ] MergeView: the uncommitted diff still polls every 3s, but only on the Files tab with uncommitted scope; Refresh reloads everything; a commit diff opened a second time is served from cache.
- [ ] MergeView: entering the uncommitted scope refetches immediately. The Refresh icon spins only if that fetch takes over 150ms, then completes its rotation. A manual Refresh spins at least one full rotation, even on a fast local fetch and with Reduce Motion on. Routine polls don't spin it.
- [ ] MergeView: a failed uncommitted-diff fetch keeps the diff and shows "Couldn't refresh, showing changes as of HH:MM" beside the Refresh button, which retries. The next success clears it, and the layout never shifts.
- [ ] IssueDetail: the spec/walkthrough tabs, history, worktree and git data refresh when `updated_at`, `status` or `head_sha` change, as they do today.
- [ ] MergeView: when a manual Refresh's spin ends, the icon flashes a green check on success, or a red alert plus a "Couldn't refresh: <reason>" toast on failure, then returns to the refresh icon.
- [ ] TDD: the pure helpers (`cyclesFromResponse`, `diffFreshness`, `spinHoldMs`, `firstQueryError`) and `SSE_INVALIDATED_KEYS` are tested first.
- [ ] `make test` passes; no visual or behavioural regression.

## Out of scope
- Mutations and removing `onRefresh`: that is xpo-6e757d.
- `fetchPending`, and other fetches not called from `useEffect`, unless they are listed above.
- MergeView's full re-render on every poll or refresh: that is xpo-5839aa.
