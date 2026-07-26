# Spec: Show review view for branches with uncommitted changes

## Problem

When a user checks out a branch named after an issue and starts editing files, the backlog view shows a branch indicator (via `BranchBadge`) but provides no way to open the merge/review view. The "Merge Branch" button in `PropertySidebar` is gated on `branch_stats.commits > 0`, so uncommitted work is invisible to the review flow.

## Requirements

1. **Detect uncommitted changes on the issue branch.** When the issue's branch is the currently checked-out branch, has 0 commits ahead of base, and the working tree has changes (staged or unstaged), populate `BranchStats` with the working-tree delta stats and set a new `has_uncommitted` flag.
2. **Show "Review Changes" entry point.** The sidebar button should appear when `branch_stats` exists and either `commits > 0` OR `has_uncommitted` is true. Label: "Review Changes" when only uncommitted changes exist; "Merge Branch" when there are commits.
3. **MergeView with disabled merge.** When opened for an uncommitted-only branch:
   - The diff/files tabs show the working-tree diff (`git diff HEAD` for the current branch).
   - The commits tab shows an empty state message.
   - The merge bar shows a warning: "Changes must be committed before merging" and the merge button is disabled.
   - The mergeability endpoint returns `can_merge: false` with the uncommitted-changes blocker (this already happens).
4. **BranchBadge enhancement.** When `has_uncommitted` is true and `commits == 0`, the badge tooltip should say "uncommitted changes" instead of "no commits yet". The badge itself can show a dot or indicator to signal dirty state.

## Design

### Model layer

**`internal/model/types.go`** — add field to `BranchStats`:

```go
type BranchStats struct {
    Branch         string `json:"branch"`
    HeadSHA        string `json:"head_sha"`
    Commits        int    `json:"commits"`
    FilesChanged   int    `json:"files_changed"`
    Insertions     int    `json:"insertions"`
    Deletions      int    `json:"deletions"`
    HasUncommitted bool   `json:"has_uncommitted"`
}
```

**`web/src/api/types.ts`** — mirror the field:

```typescript
export interface BranchStats {
  branch: string;
  head_sha: string;
  commits: number;
  files_changed: number;
  insertions: number;
  deletions: number;
  has_uncommitted: boolean;
}
```

### Backend changes

**`internal/exponential/branches.go`** — in `computeBranchStats()`, after computing committed stats, if `commits == 0` and the branch is the currently checked-out branch, run `git diff --shortstat HEAD` to detect uncommitted changes. If there are any, set `HasUncommitted = true` and populate `FilesChanged`/`Insertions`/`Deletions` from the working-tree diff.

New helper `computeUncommittedStats(stats *BranchStats)`:
- Only called when `stats.Commits == 0` and `stats.Branch == CurrentBranch()`
- Runs `git diff --shortstat HEAD` (unstaged) + `git diff --shortstat --cached` (staged)
- Combines the results into the stats fields
- Sets `HasUncommitted = true` if any changes detected

**`internal/exponential/review.go`** — new functions for working-tree diffs:

- `GetWorkingTreeDiffText() string` — runs `git diff HEAD` to capture both staged and unstaged changes
- `ListWorkingTreeFilesChanged() []FileStat` — runs `git diff --numstat HEAD` + `git diff --name-status HEAD`

**`internal/server/handlers.go`** — modify the diff/files/commits handlers:

- `handleGetIssueDiff`: when `issue.BranchStats.Commits == 0` and `HasUncommitted`, return `GetWorkingTreeDiffText()` instead of `GetDiffText(branch, base)`
- `handleGetIssueFiles`: same pattern, use `ListWorkingTreeFilesChanged()`
- `handleGetIssueCommits`: when commits == 0, return empty array (already does, but verify)
- `handleMergeability`: already returns `can_merge: false` when working tree is dirty — no change needed

### Frontend changes

**`web/src/components/IssueDetail/PropertySidebar.tsx`** — change the merge button condition:

```tsx
{issue.branch_stats && 
 (issue.branch_stats.commits > 0 || issue.branch_stats.has_uncommitted) && 
 issue.status !== "DONE" && onOpenMerge && (
  <button onClick={onOpenMerge} ...>
    <GitMerge size={16} />
    {issue.branch_stats.commits > 0 ? "Merge Branch" : "Review Changes"}
  </button>
)}
```

**`web/src/components/IssueDetail/MergeView.tsx`** — handle the uncommitted-only state:

- The existing mergeability check already disables the merge button when can_merge is false
- The existing blocker display already shows "Working tree has uncommitted changes"
- The subtitle under the merge bar should change from "Merging will close this issue" to "Commit your changes to enable merging" when uncommitted-only
- The commits tab should show a friendly empty state when `commits.length === 0`

**`web/src/components/ui/BranchBadge.tsx`** — update tooltip text:

- When `has_uncommitted && commits === 0`: tooltip says "uncommitted changes" with the file/line stats
- Optionally show a small dot indicator on the badge to signal dirty state

## Acceptance criteria

- [ ] A branch with 0 commits but uncommitted changes shows "Review Changes" button in sidebar
- [ ] Clicking it opens MergeView with the working-tree diff visible in the files tab
- [ ] Merge button is disabled with "Working tree has uncommitted changes" blocker
- [ ] Commits tab shows empty state when there are no commits
- [ ] BranchBadge tooltip reflects uncommitted status
- [ ] A branch with commits continues to work exactly as before ("Merge Branch" label, full merge flow)
- [ ] A branch with both commits AND uncommitted changes shows committed diff (existing behavior) with merge disabled (existing behavior via mergeability)
- [ ] `make test` passes
- [ ] `make build` succeeds

## Out of scope

- Staging individual files from the review view
- Committing from the review view
- Showing a split between staged vs unstaged changes
