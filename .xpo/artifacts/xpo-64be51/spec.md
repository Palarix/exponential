# Spec: Merge view handles an issue merged while the view is open

## What
If an issue's branch disappears while the merge view is open, the view shows a clear state instead of a raw `{"error":"no branch found for this issue"}` screen:
- the issue is DONE → "This issue has been merged"
- otherwise → a readable "no branch found" message

API error messages across the UI also stop showing raw JSON.

## Why
Merging from the agent chat while the merge view stays open is now a normal flow. The view already has a "Branch merged" screen, but it can't be reached during this race.

## Root cause
1. After the merge, the issue update arrives without `branch_stats`. The query keys include `head_sha` (`web/src/api/query-keys.ts:32-34`), so they change and all four branch queries refetch. Each refetch gets a 404.
2. `MergeView` checks `loading` and `error` before `!bs` (`MergeView.tsx:322-358`), so the error screen wins.
3. `ApiError` uses the raw response body as its message (`web/src/api/client.ts:12-14`).

## How
1. **Check the branch first** (`MergeView.tsx`). The render order becomes:
   1. no `branch_stats` → merged or no-branch screen
   2. loading → "Loading..."
   3. error → error screen
   4. normal view

   The no-branch check must come before `loading`, because of change 4.
2. **Tell "merged" apart from other missing branches** (`MergeView.tsx`):
   | `issue.status` | Screen |
   |---|---|
   | `DONE` | Merge icon + "This issue has been merged", with a "Back to issue" button (`onClose`) |
   | anything else | Warning icon + "No branch found for this issue. It may have been deleted or renamed.", with a "Back to issue" button |

   `merge` always sets the issue to DONE, so DONE plus no branch means it was merged.
3. **Readable API errors** (`client.ts`): `ApiError` reads the body as JSON. If the body is an object with a string `error` field, that field becomes the message. Otherwise the message is the body as now, or `"<status> <statusText>"` if the body is empty. The raw body stays in `.body`. Nothing in `web/src` parses `err.message` as JSON.
4. **Don't fetch without a branch** (`queries.ts`): `useIssueCommits`, `useIssueDiff` and `useMergeability` get `enabled: !!issue.branch_stats`. This also stops the 3s uncommitted poll. `MergeView` is the only place that uses these hooks.

## Unchanged
- Merging from the view's own Merge button still closes the view and shows the "Branch merged" toast (`onMerged`).
- The view does not close on its own when the branch disappears, so the user isn't pulled away from a diff they're reading.

## Edge cases
- **A 404 arrives before the issue update (`branch_stats` still set):** the error screen shows briefly, now with a readable message. When the issue update arrives, change 1 shows the merged screen. No extra handling is needed.
- **Opening the merge view for an issue with no branch:** goes straight to the no-branch screen, without making a request.
- **A failed uncommitted poll while the branch still exists:** unchanged. The last diff stays on screen with the stale notice.

## Acceptance criteria
- [ ] With the merge view open, merging the issue elsewhere (chat, CLI, MCP) shows "This issue has been merged" with a "Back to issue" button, not an error.
- [ ] A non-DONE issue whose branch is gone shows the no-branch message, not the merged screen.
- [ ] `ApiError.message` is the `error` field for JSON error bodies. Non-JSON and empty bodies behave as before. `.body` keeps the raw text.
- [ ] Without `branch_stats`, the merge view makes no commits, diff or mergeability requests.
- [ ] Unit tests cover the `ApiError` parsing, both no-branch screens, and the order of no-branch before loading and error (`MergeView.test.ts`, `client` tests).
- [ ] `make test` passes.
