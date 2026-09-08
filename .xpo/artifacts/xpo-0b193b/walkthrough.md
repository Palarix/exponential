# Walkthrough: Fix spec/walkthrough live-update in issue detail

## What changed

**Single file:** `web/src/components/IssueDetail/IssueDetail.tsx`

Added a 3-line `useEffect` that clears the artifact content cache when `issue.updated_at` changes.

## The problem

The SSE pipeline works correctly end-to-end: writing an artifact appends an ARTIFACT event to `issues.db`, the file watcher detects the change, broadcasts an SSE event, and `App.tsx` re-fetches the issue list with updated timestamps. However, `IssueDetail` cached artifact content in local state (`artifactContent`) and never invalidated it — the fetch effect had an early-return guard (`if (artifactContent[filename] !== undefined) return`) that prevented re-fetching, and the cache was only cleared on `issue.id` change (navigating to a different issue).

## The fix

```tsx
useEffect(() => {
  setArtifactContent({});
}, [issue.updated_at]);
```

This effect runs whenever the issue's `updated_at` timestamp changes (which happens on any SSE-triggered re-fetch). It clears the entire `artifactContent` map, so the next time the fetch effect runs for the active tab, it sees `undefined` and triggers a fresh fetch.

## Why a separate effect

The existing `issue.id` effect (line 80-88) resets broader UI state: active tab back to "details", editing fields, optimistic values, comments. An `updated_at` change should only invalidate cached content without disrupting the user's current view — if they're looking at the spec tab, they should stay on the spec tab and see the updated content, not get bounced back to the details tab.
