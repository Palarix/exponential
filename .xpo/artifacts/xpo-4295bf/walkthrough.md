# Walkthrough: Show review view for branches with uncommitted changes

## What changed and why

Previously, the merge/review view was only accessible when a branch had at least one commit ahead of the base branch. This meant that if you checked out a branch, started editing files, but hadn't committed yet, you could see the branch indicator badge but had no way to review your working-tree changes through the UI. This change makes the review flow available as soon as there are any changes — committed or not.

## Model changes

**`internal/model/types.go`** — The `BranchStats` struct gained a new boolean field:

```go
HasUncommitted bool `json:"has_uncommitted"`
```

This flag tells the frontend whether the stats represent uncommitted working-tree changes rather than committed branch deltas. It's only set when `Commits == 0` and the branch is the currently checked-out branch.

**`web/src/api/types.ts`** — The TypeScript interface mirrors the new field as `has_uncommitted: boolean`.

## Backend: detecting uncommitted changes

**`internal/exponential/branches.go`** — The core change is in `computeBranchStats()`. Previously it always ran `git diff --shortstat base...branch` to get file/line stats. Now:

1. If `commits > 0`, it behaves exactly as before (committed diff stats).
2. If `commits == 0` AND the branch is the current branch (`branch == CurrentBranch()`), it calls the new `fillUncommittedStats()` function instead.

`fillUncommittedStats()` runs two git commands:
- `git diff --shortstat HEAD` — detects unstaged changes
- `git diff --shortstat --cached` — detects staged changes

It combines both into the stats fields and sets `HasUncommitted = true` if either has changes. This means the `BranchBadge` and sidebar button can reflect the working-tree state.

Why check `CurrentBranch()`? If the branch isn't checked out, there's no working tree to inspect — the uncommitted state only makes sense for the active branch.

## Backend: serving working-tree diffs

**`internal/exponential/review.go`** — Two new functions:

- `GetWorkingTreeDiffText()` — runs `git diff HEAD` to get a unified diff of all staged + unstaged changes
- `ListWorkingTreeFilesChanged()` — delegates to a new shared helper `listFilesChangedFromDiff("HEAD")`

The existing `ListFilesChanged(branch, base)` was refactored to also use `listFilesChangedFromDiff(base + "..." + branch)`. This avoids duplicating the `--numstat` / `--name-status` parsing logic.

**`internal/server/handlers.go`** — Two handlers were updated:

- `handleGetIssueDiff`: checks `issue.BranchStats.Commits == 0 && HasUncommitted` — if true, returns `GetWorkingTreeDiffText()` instead of `GetDiffText(branch, base)`
- `handleGetIssueFiles`: same check, returns `ListWorkingTreeFilesChanged()` instead of `ListFilesChanged(branch, base)`

The `handleGetIssueCommits` handler naturally returns an empty list when there are no commits (the git log produces nothing), so no change was needed there. The `handleMergeability` handler already returns `can_merge: false` with a "Working tree has uncommitted changes" blocker — also no change needed.

## Frontend: sidebar button

**`web/src/components/IssueDetail/PropertySidebar.tsx`** — The visibility condition changed from:

```tsx
issue.branch_stats.commits > 0
```

to:

```tsx
issue.branch_stats.commits > 0 || issue.branch_stats.has_uncommitted
```

The label dynamically switches between "Merge Branch" (when there are commits to merge) and "Review Changes" (when there are only uncommitted changes). This sets the right expectation — you're reviewing, not merging.

## Frontend: MergeView adjustments

**`web/src/components/IssueDetail/MergeView.tsx`** — Three changes:

1. **Subtitle text**: The line under the mergeability indicator changes from "Merging will close this issue" to "Commit your changes to enable merging" when `commits === 0 && has_uncommitted`. This tells the user what action to take.

2. **Commits tab empty state**: When the commits array is empty, instead of showing an empty sidebar with "Select a commit to view its changes", the tab now renders a centered message with a commit icon: "No commits yet — Uncommitted changes are shown in the Files tab". This guides the user to the right tab.

3. **Merge button stays disabled**: No code change needed here — the existing mergeability endpoint already returns `can_merge: false` when the working tree is dirty, and the merge button respects that.

## Frontend: BranchBadge

**`web/src/components/ui/BranchBadge.tsx`** — The tooltip now has three states:

1. Has commits: shows commit count, files, insertions/deletions (unchanged)
2. Has uncommitted (no commits): shows "uncommitted changes: N files, +X -Y"
3. Neither: shows "no commits yet" (unchanged)

Additionally, when `has_uncommitted && commits === 0`, a small amber dot renders inside the badge to visually signal dirty state without taking up much space.

## What was NOT changed

- The mergeability endpoint (`handleMergeability`) already blocks merges when the working tree is dirty
- No new API endpoints were added — the existing `/diff`, `/files`, `/commits` endpoints serve both modes
- No changes to the merge execution flow — the merge button is simply disabled in the uncommitted state
- The `applyBranchInference` status inference logic is unchanged — it only infers DOING from remote branches, not from uncommitted local changes
