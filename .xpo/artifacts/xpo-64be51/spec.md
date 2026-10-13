# Spec: Merge view handles an issue merged while the view is open

## What
If an issue is merged while its merge view is open, the view switches in place to a read-only **Merged** view. The view keeps the same diff, commits, walkthrough and conversation, now read from the merge commit, and shows a "Merged" status instead of the Merge button. You no longer see a raw `{"error":"no branch found for this issue"}` screen.

Merged issues also get a secondary **View changes** button in the issue details, which opens the same Merged view.

API error messages across the UI also stop showing raw JSON.

## Why
Merging from the agent chat while the merge view stays open is now a normal flow. Today the view switches to a raw JSON error. Seeing what was merged is useful after the fact too.

## Root cause
1. After the merge, the issue update arrives without `branch_stats`. The query keys include `head_sha` (`web/src/api/query-keys.ts:32-34`), so they change and all four branch queries refetch. Each refetch gets a 404.
2. `MergeView` checks `loading` and `error` before `!bs`, so the error screen wins.
3. `ApiError` uses the raw response body as its message (`web/src/api/client.ts:12-14`).

## How

### Server
1. **`merged` flag on issues**: `IssueResponse` gets `merged: true` when the issue has a MERGE event. The DONE transition and the MERGE event are written together, so the flag and `status: DONE` arrive in the same issue update.
2. **Finding the merge commit** (`exponential.FindMergeCommit`): uses the issue's latest MERGE event.
   - If the payload's `merge_sha` is set (older events) and resolves, use it.
   - Otherwise, take the first commit on the default branch's first-parent history after `base_sha` that adds this issue's MERGE event to `.xpo/issues.db`: `git log --first-parent --reverse <base>..<default> -S'"id":"<id>","type":"MERGE"' -- .xpo/issues.db`. Merge amends the bookkeeping into the merge commit (or, for ff, the branch tip), so this works for every strategy.
   - Not found → error.
3. **`GET /api/issues/{id}/merged`** returns `{merge_sha, base_sha, branch, strategy, merged_at, merged_by, merged_on_behalf_of, commits, diff}`:
   - `commits`: `git log --no-merges <base>..<merge>`. Squash gives the one squash commit, merge gives the branch's commits, ff gives the fast-forwarded commits.
   - `diff`: `<base>...<merge>`. The UI already hides `.xpo/` files, so the bookkeeping doesn't show.
   - 404 with a readable message if the issue has no MERGE event or the merge commit can't be found.
   - Commit diffs use the existing `/api/issues/{id}/commits/{sha}/diff`, which doesn't need a branch.

### Web
4. **Readable API errors** (`client.ts`): `ApiError` reads the body as JSON. If the body is an object with a string `error` field, that field becomes the message. Otherwise the message is the body as now, or `"<status> <statusText>"` if the body is empty. The raw body stays in `.body`.
5. **Modes** (`MergeView.tsx`):
   | Condition | View |
   |---|---|
   | `merged && status === DONE` | **Merged view** |
   | else, `branch_stats` set | Branch view (as today) |
   | else | No-branch screen: warning icon + "No branch found for this issue. It may have been deleted or renamed." + "Back to issue" |

   Because the mode comes from the issue, an open branch view switches to the Merged view in place when the merged issue update arrives.
6. **Queries** (`queries.ts`):
   - The branch hooks (`useIssueCommits`, `useIssueDiff`, `useMergeability`) only run in branch mode: `enabled` is false for merged issues or issues without `branch_stats`. This also stops the 3s uncommitted poll.
   - New `useMergedChanges(issue, enabled)` runs only in merged mode.
   - Loading and error states are computed only from the queries of the current mode, since disabled queries stay pending.
7. **Merged view**:
   - The merge bar shows `main ← <branch>` as before. In the middle is a green **Merged** status with the short merge SHA, who merged it and when. The Merge button and strategy menu are gone.
   - Tabs are Walkthrough, Commits, Files changed and Conversation. There is no All/Uncommitted toggle, and Files always shows the merged diff.
   - Refresh refetches the merged changes.
   - An open merge confirmation dialog closes.
   - If `/merged` returns 404: a merge icon, "This issue has been merged", the readable reason ("Its changes can't be shown: …") and a "Back to issue" button.
8. **"View changes" button** (`PropertySidebar.tsx`): shows when `gitActions && issue.merged && status === DONE`, styled as a secondary button (surface + border, like Start Work next to Review Changes), and opens the merge view.

## Unchanged
- Merging from the view's own Merge button still closes the view and shows the "Branch merged" toast (`onMerged`).
- The view never closes on its own, so the user isn't pulled away from what they're reading.
- Remote mode still hides git actions, including View changes.

## Edge cases
- **A 404 arrives before the issue update:** the branch view may briefly show a readable error. When the update arrives, the mode switches to merged.
- **Merged with `keep_branch`:** `branch_stats` may still be set, but `merged && DONE` wins, so the Merged view shows.
- **Reopened after a merge:** the status isn't DONE, so the branch view or the no-branch screen shows.
- **Merged more than once:** the latest MERGE event is used.
- **Bookkeeping amend failed during the merge:** the MERGE event lands in a later commit, so the search finds that commit and its range may include unrelated commits. This is rare. It is accepted and documented.
- **Set DONE by hand, never merged:** `merged` is false, so there's no View changes button. If the view is open, it shows the no-branch screen or the branch view.

## Acceptance criteria
- [ ] With the merge view open, merging the issue elsewhere (chat, CLI, MCP) switches the view in place to the Merged view, with the merged diff, commits and a "Merged" status, not an error.
- [ ] A merged DONE issue has a secondary "View changes" button that opens the Merged view.
- [ ] If the merge commit can't be found, the view shows "This issue has been merged" with a readable reason and "Back to issue".
- [ ] An issue that isn't merged and has no branch shows the no-branch screen.
- [ ] `ApiError.message` is the `error` field for JSON error bodies. Non-JSON and empty bodies behave as before. `.body` keeps the raw text.
- [ ] Branch requests (commits, diff, mergeability, uncommitted poll) are not made in merged or no-branch mode.
- [ ] Go tests cover `FindMergeCommit` for squash, merge and ff, the `/merged` endpoint (200 and 404), and the `merged` flag.
- [ ] Web tests cover the `ApiError` parsing, mode selection, the no-branch, merged-fallback and Merged-view rendering, and the View changes button.
- [ ] `make test` passes.
