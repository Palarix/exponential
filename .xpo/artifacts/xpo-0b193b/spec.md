# Fix: spec/walkthrough don't live-update in issue detail

## What
Artifact content (spec.md, walkthrough.md) cached in `IssueDetail` is never invalidated when the file changes on disk.

## Root cause
In `IssueDetail.tsx` line 108, the guard `if (artifactContent[filename] !== undefined) return;` prevents re-fetching once content is cached. The `artifactContent` state is only cleared on `issue.id` change (line 87), not when `issue.updated_at` changes — so the stale cached content persists even though the SSE pipeline correctly delivers the update event.

The SSE pipeline itself works fine:
1. `writeArtifact()` writes the file and appends an ARTIFACT event to `issues.db`
2. `WatchDB` detects the change → broadcasts SSE `issue_updated`
3. `App.tsx` re-fetches all issues → `issue.updated_at` changes
4. `IssueDetail`'s artifact fetch effect short-circuits at the cache guard

## Fix
Clear `artifactContent` when `issue.updated_at` changes, so the next render of the active tab triggers a fresh fetch.

Add a separate `useEffect` with `[issue.updated_at]` dependency that calls `setArtifactContent({})`. This must be a separate effect from the `issue.id` reset (line 80-88) because `issue.id` changes should also reset UI state (active tab, editing fields, etc.), while `updated_at` changes should only invalidate the content cache.

## Acceptance criteria
- [ ] Editing spec.md on disk (via CLI or agent) updates the spec tab in real-time
- [ ] Editing walkthrough.md on disk updates the walkthrough tab in real-time
- [ ] Navigating to a different issue still resets the cache and tab
- [ ] No unnecessary re-fetches when `updated_at` hasn't changed
- [ ] `make test` passes
