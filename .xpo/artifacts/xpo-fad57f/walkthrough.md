## Summary

The mergeability check was comparing working-directory state against the branch's changed files, falsely reporting merge conflicts when uncommitted changes overlapped. This was replaced with `git merge-tree --write-tree`, which compares only committed refs. A second guard was added to block merges when the worktree has uncommitted or untracked files that would be silently destroyed on cleanup.

## What changed

### Backend — `internal/exponential/merge.go`

The monolithic `HubCleanForMerge` was split into focused functions:

- **`CheckMergeConflicts(branch)`** — runs `git merge-tree --write-tree <base> <branch>` to test whether committed refs merge cleanly. Parses `CONFLICT` lines from the output to extract file names. Working-directory state is completely irrelevant.
- **`HubRequireCleanTree()`** — checks for dirty tracked files in the hub (excluding `.xpo/` and untracked). Simple guard with a clear "uncommitted changes — commit or discard before merging" message. No branch parameter needed.
- **`HubDirtyTrackedFiles()`** — returns the list of dirty tracked hub files for the mergeability endpoint to use as warnings.
- **`WorktreeDirtyFiles(branch)`** / **`WorktreeRequireClean(branch)`** — finds the worktree for the branch via `FindWorktreeForBranch`, checks `git status --porcelain` for any dirty or untracked files (excluding `.xpo/`). Returns nil if no worktree exists (branch-only mode). This prevents the silent data loss that occurred when `git worktree remove --force` destroyed uncommitted work.
- **`HubCleanForMerge(branch)`** — kept as a convenience for CLI and MCP callers. Delegates to all three checks: merge conflicts → hub clean → worktree clean.

### Backend — `internal/server/handlers.go`

**`handleMergeability`** now returns structured blockers instead of raw error strings:

```json
{
  "can_merge": false,
  "blockers": [{"message": "3 merge conflicts with main", "files": ["shared.txt", ...]}],
  "warnings": ["uncommitted changes: foo.go — commit before merging"]
}
```

Merge conflicts and dirty worktree are blockers. Dirty hub files are warnings (they don't prevent the merge from being *possible*, just from being *executed*).

**`handleMergeIssue`** runs all three checks independently with distinct HTTP 409 error messages.

A `pluralize` helper was added to `server.go` for the blocker messages.

### Frontend — `web/src/api/client.ts`

New `MergeBlocker` interface (`{message, files?}`) and `warnings` field on `Mergeability`.

### Frontend — `web/src/components/IssueDetail/MergeView.tsx`

- **Zero-commit guard**: `canMerge` factors in `hasCommits`. With zero commits the status shows "No commits to merge" and the button is disabled.
- **Blocker summary**: the amber status message is now a clickable button showing the first blocker's `message`.
- **Detail modal**: clicking opens the project's `Modal` component (portal-rendered, focus-trapped, Escape-dismissable) with the full blocker list, scrollable file list per blocker, and remediation instructions.
- **Subtitle**: shows "Resolve blockers to enable merging" when blocked, instead of "Merging will close this issue".

### Frontend — `web/src/components/IssueDetail/PropertySidebar.tsx`

The sidebar button always reads "Review Changes" instead of conditionally showing "Merge Branch".

### Tests — `internal/exponential/merge_test.go`

- **`CheckMergeConflicts`**: clean merge, committed conflict (divergent `shared.txt`), dirty tree ignored (the key bug scenario).
- **`HubRequireCleanTree`**: clean tree, untracked allowed, `.xpo/` allowed, modified tracked file blocked with correct message.
- **`WorktreeRequireClean`**: clean worktree, untracked file blocked, modified tracked blocked, `.xpo/` allowed, no-worktree passthrough.
- **`HubCleanForMerge`** (legacy): adapted to verify delegation behavior.

## Key decisions

- **`HubRequireCleanTree` blocks on any dirty tracked file**, not just files overlapping with the branch. Simpler and safer — if you're about to merge, commit everything first.
- **`WorktreeRequireClean` also catches untracked files**, unlike the hub check. Untracked files in the hub are harmless, but untracked files in the worktree are destroyed by `git worktree remove --force`.
- **Structured blockers** (`{message, files}`) rather than raw error strings. This enabled the short-summary + detail-modal UX pattern without client-side string parsing.
- **Used the project's `Modal` component** instead of hand-rolled positioning, which avoids stacking context issues with the tab content area below the merge status bar.