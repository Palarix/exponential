# Review Changes for worktrees with only new files

## What

1. Branch stats count new, untracked files as uncommitted changes, so issue details shows Review Changes.
2. The Review Changes dialog opens on "Uncommitted" when the branch has no commits but has uncommitted changes, so it doesn't open empty.

## Why

- `fillUncommittedStats` (`internal/exponential/branches.go`) runs `git diff --shortstat HEAD` and `git diff --shortstat --cached`. Neither counts untracked files. A worktree whose only change is a new file reports `has_uncommitted: false`, and issue details hides Review Changes (it needs `commits > 0 || has_uncommitted`).
- Work Mode agents (xpo-00f0e5) often start by creating files, so this hides Review Changes for real work.
- The dialog's uncommitted file list and diff already include untracked files (`ListWorkingTreeAllFilesChanged`, `GetWorkingTreeFullDiffText` with synthesized diffs). But the dialog opens on "All changes", which is the committed diff only. With no commits it opens empty, for edited files as much as for new ones.

## Acceptance Criteria

- [ ] A worktree with only an untracked file reports `has_uncommitted: true`, counts the file in `files_changed`, and adds its lines to `insertions`.
- [ ] Untracked and edited files together are both counted.
- [ ] Gitignored files and files under `.xpo/` are not counted.
- [ ] Issue details shows Review Changes for such an issue.
- [ ] The Review Changes dialog opens on "Uncommitted" when the branch has no commits and has uncommitted changes, and lists the new file with its diff.
- [ ] With commits, the dialog still opens on "All changes".
- [ ] When issue details shows Review Changes, it is the primary button, and Start Work or Continue Work is a secondary button. Without Review Changes, Start Work or Continue Work stays primary.
- [ ] The primary action is always the top button: with Review Changes, it sits above Start Work or Continue Work.
- [ ] `make test` passes.

## Flow

1. Add `listUntrackedStats(dir)` in `internal/exponential`: the untracked files from `listUntrackedFiles` (which already excludes gitignored files and `.xpo/`), with their line counts. `ListWorkingTreeAllFilesChanged` uses the same line counting.
2. In `fillUncommittedStats`, add the untracked files to `files_changed` and `insertions`, and set `has_uncommitted` when there are any.
3. In `PropertySidebar.tsx`, render the work button in the secondary style when Review Changes shows.
4. In `MergeView.tsx`, start `diffScope` at "uncommitted" when `branch_stats.commits === 0 && branch_stats.has_uncommitted`.

## Decisions

- **The primary action is always on top.** A stack of actions reads top-down; the most important one comes first.
- **Review Changes outranks Continue Work.** Once a worktree has changes, reviewing them is the next step. Two primary buttons stacked made neither stand out.

- **Line counts for new files count as insertions,** like `ListWorkingTreeAllFilesChanged` already does for the file list. A binary or oversized file counts as a changed file; its lines count as read.
- **Default the dialog's scope, don't merge the two scopes.** "All changes" stays the committed diff, which is what merges. Opening on "Uncommitted" only when there is nothing committed avoids an empty first view without changing what "All changes" means.
