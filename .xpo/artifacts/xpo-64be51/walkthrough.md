# Walkthrough: Merge view handles an issue merged while the view is open

## What was built
If an issue is merged while its merge view is open (from the agent chat, the CLI or MCP), the view used to switch to a raw `{"error":"no branch found for this issue"}` screen. Now it switches in place to a read-only **Merged** view, which shows the same diff and commits, read from the merge commit. Merged issues also get a secondary **View changes** button that opens this view at any time.

## Why it broke
- After a merge, the issue update arrives without `branch_stats`. The branch query keys include `head_sha`, so they changed and refetched.
- The branch is gone, so every refetch got a 404.
- `MergeView` checked `loading` and `error` before `!bs`, so the error screen won over the existing "Branch merged" screen.
- `ApiError` used the raw response body as its message, which is where the JSON text came from.

## How the pieces fit

### Finding the merge commit (`internal/exponential/merged.go`)
The MERGE event can't store the merge commit's SHA. `MergeIssue` writes the event first, then amends the bookkeeping (`.xpo/issues.db`) into the merge commit. For ff, it amends the branch tip. The amend changes the SHA. That's why `merge_sha` is empty in recent events.

The amend leaves a reliable mark, though: the merge commit is the commit that **adds this issue's MERGE event line** to `issues.db`. So `FindMergeCommit` runs:

```
git log --first-parent --reverse <base_sha>..<default> -S'"id":"<id>","type":"MERGE"' -- .xpo/issues.db
```

and takes the first result.
- `base_sha` comes from the event and narrows the search to commits after the merge started.
- `--first-parent` keeps it on main's own history.
- It works the same for squash, merge and ff.
- Older events that did record `merge_sha` use it directly.

`GetMergedChanges` builds the response from that commit:
- **Commits:** `git log --no-merges base..merge`. Squash gives one commit, merge gives the branch's commits without the merge commit, and ff gives the fast-forwarded commits.
- **Diff:** `base...merge`. The UI's `parseDiffByFile` already drops `.xpo/` files, so the bookkeeping doesn't show.

`ListBranchCommitsDetailed` now delegates to a small `listCommitsDetailed(args...)`, so both paths share the parsing.

### API (`internal/server`)
- `GET /api/issues/{id}/merged` returns `MergedChanges`, or 404 with a readable reason: "has not been merged" or "merge commit … not found".
- `IssueResponse.merged` is true when the issue has a MERGE event. The DONE transition and the MERGE event are written together, so the UI gets `merged` and `status: DONE` in the same issue update. The in-place switch depends on this.

### Web
- **`ApiError`** (`client.ts`): uses the `error` field of a JSON error body as its message. Otherwise it keeps the old behavior. `.body` keeps the raw text.
- **`mergeViewMode(issue)`** (`diff-utils.ts`): returns `merged` if `merged && DONE`, `branch` if there's a branch, and `no-branch` otherwise.
  - The mode comes from the issue, not from query results. So when the merged issue update arrives, the view re-renders into merged mode by itself.
  - `merged && DONE` wins even when `keep_branch` left the branch around.
- **Queries per mode** (`queries.ts`, `MergeView.tsx`):
  - The branch hooks take an `enabled` flag, and the new `useMergedChanges` is enabled only in merged mode. The 404 requests and the 3s uncommitted poll stop as soon as the issue is merged.
  - Disabled queries stay `pending` forever, so `loading` and `error` are computed only from the current mode's queries. Otherwise merged mode would be stuck on "Loading...".
- **Render order:** no-branch screen, then loading, then the merged fallback (on a `/merged` error), then the generic error, then the view. The no-branch check comes first because that mode has no queries to wait on.
- **Merged view:**
  - In the merge bar, the mergeability text and Merge button are replaced by `MergedStatus`: a green "Merged", the short SHA, who merged it (`ActorName`, so "via agent" shows) and when.
  - The All/Uncommitted toggle is hidden, and the Files tab is forced to `scope = "all"`.
  - Refresh refetches the merged changes.
  - The merge confirmation dialog only renders in branch mode, so it disappears if the issue is merged elsewhere.
  - Cached mergeability from a kept branch is ignored outside branch mode.
- **Fallback screens:** both use a shared `EmptyState` component with a "Back to issue" button.
  - "This issue has been merged" plus the reason, for a `/merged` 404.
  - "No branch found for this issue. It may have been deleted or renamed."

  The old "Branch merged" screen is gone.
- **View changes** (`PropertySidebar.tsx`):
  - It shows when `gitActions && merged && DONE`. Like Review Changes, it needs a local board.
  - It uses the secondary style (surface + border). Nothing is left to do on a merged issue, so it shouldn't look like a call to action.
  - It reuses `onOpenMerge`, so `IssueDetail` didn't change.

## Decisions
- **Look up the merge commit, don't store it.** Storing it would need a second bookkeeping commit after the amend. Searching for the event line with `-S` takes one git command and also works for every merge already in history.
- **Merged mode only for `merged && DONE`.** An issue set to DONE by hand was never merged, so it shouldn't claim to be. A merged issue that was reopened goes back to the branch view.
- **The view never closes on its own.** The user may be reading a diff. Merging from the view's own button still closes it and shows the "Branch merged" toast.
- **A short "Loading..." during the switch** is accepted. Keeping the old branch view up until the merged data arrives would briefly show a Merge button for an issue that's already merged.
- **`renderMarkup(el, client?)`** takes an optional `QueryClient`, so component tests can render loaded and error states.

## Non-obvious details
- If the bookkeeping amend fails during a merge, the MERGE event is committed with a later commit. The search then finds that commit, and its range may include unrelated commits. This is rare and accepted.
- In tests, an error seeded into the cache renders as "retry in progress" unless `retryOnMount: false`. The app's own flow (fetch, fail, re-render) isn't affected.

## Acceptance criteria
- [x] With the merge view open, merging the issue elsewhere switches the view in place to the Merged view with the merged diff, commits and "Merged" status. *Evidence:* the mode comes from `mergeViewMode(issue)` (unit-tested). The render test "shows the Merged view from the merge commit" passes. Tophatted by the user.
- [x] A merged DONE issue has a secondary "View changes" button that opens the Merged view. *Evidence:* the PropertySidebar tests check that it shows, that it isn't primary-styled, and that it's hidden in remote mode, without a merge, and when not DONE.
- [x] If the merge commit can't be found, the view shows "This issue has been merged" with a readable reason and "Back to issue". *Evidence:* render test "falls back to a merged screen with the reason". No raw JSON appears in the markup.
- [x] An issue that isn't merged and has no branch shows the no-branch screen. *Evidence:* render test "shows the no-branch screen, not an error". It renders without "Loading...".
- [x] `ApiError.message` is the `error` field for JSON error bodies. Non-JSON and empty bodies behave as before, and `.body` keeps the raw text. *Evidence:* the `ApiError message` tests in `client.test.ts`.
- [x] Branch requests are not made in merged or no-branch mode. *Evidence:* `enabled: isBranch` on the commits, diff, mergeability and uncommitted queries. `showView` and the refresh only refetch branch queries in branch mode.
- [x] Go tests cover `FindMergeCommit` for squash, merge and ff, the endpoint (200 and 404), and the `merged` flag. *Evidence:* `merged_test.go` and `merged_handlers_test.go`. Also checked against real history: xpo-61232d resolves to e613ff3.
- [x] Web tests cover ApiError parsing, mode selection, the screens and the button. *Evidence:* `client.test.ts`, `MergeView.test.ts`, `MergeView.render.test.tsx` and `PropertySidebar.test.tsx`.
- [x] `make test` passes. *Evidence:* exit 0 (844 web tests plus the Go suite). `make frontend` and `make cli` also build cleanly.
