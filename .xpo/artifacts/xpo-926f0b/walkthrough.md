# Walkthrough: Review Changes for worktrees with only new files

## What was built

Review Changes was hidden when a worktree's only changes were new files that git doesn't track yet. That's common now: Work Mode agents (xpo-00f0e5) often start by creating files.

1. **Branch stats count untracked files.** `fillUncommittedStats` (`internal/exponential/branches.go`) only ran `git diff --shortstat HEAD` and `--cached`, which ignore untracked files. Such a worktree reported `has_uncommitted: false`, and issue details hid Review Changes.
2. **The dialog opens where the changes are.** "All changes" is the committed branch diff. With no commits yet it was empty, for edited files as much as new ones. The dialog now opens on "Uncommitted" in that case.
3. **One primary button, on top.** Fixing the stats put Review Changes next to Continue Work, two accent buttons stacked. Review Changes is now the primary, top button, and the work button turns secondary.

## How the pieces fit

- **`listUntrackedStats(dir)`** (`internal/exponential/review.go`) returns the untracked files as added files, each with its line count as insertions. It builds on `listUntrackedFiles`, which already uses `git ls-files --others --exclude-standard` and skips `.xpo/`.
  - Both the branch stats and the dialog's file list (`ListWorkingTreeAllFilesChanged`) now use it, so the button and the dialog agree.
- **`fillUncommittedStats`** adds each untracked file to `files_changed` and `insertions`, and sets `has_uncommitted`. It still runs only when the branch has no commits ahead of base; with commits, Review Changes shows anyway.
- **`initialDiffScope(branch_stats)`** (`web/src/components/IssueDetail/diff-utils.ts`) returns "uncommitted" when `commits === 0 && has_uncommitted`, else "all". `MergeView` uses it as the starting `diffScope`. The `DiffScope` type moved there with it.
- **`PropertySidebar`** computes `showReview` once. It renders Review Changes first, and gives Start Work or Continue Work the secondary style while `showReview` is true.

## Decisions

- **New files' lines count as insertions,** matching the dialog's file list.
- **The default scope changed, not what the scopes mean.** "All changes" stays the committed diff, which is what merges.
- **One primary action, always on top.** A stack reads top-down. Once a worktree has changes, reviewing them is the next step.

## Acceptance Criteria

- [x] A worktree with only an untracked file reports `has_uncommitted: true`, counts the file in `files_changed`, and adds its lines to `insertions`. *`TestFillUncommittedStats_UntrackedOnly`. Live: upe-000050 reported 1 file, +1, `has_uncommitted: true`.*
- [x] Untracked and edited files together are both counted. *`TestFillUncommittedStats_UntrackedAndEdited` (2 files, 2 insertions).*
- [x] Gitignored files and files under `.xpo/` are not counted. *`TestFillUncommittedStats_UntrackedOnly` (with `debug.log` and `.xpo/state`); `TestFillUncommittedStats_Clean`.*
- [x] Issue details shows Review Changes for such an issue. *Live on upe-000050.*
- [x] The Review Changes dialog opens on "Uncommitted" when the branch has no commits and has uncommitted changes, and lists the new file with its diff. *`initialDiffScope` test. Live: it opened on Uncommitted with `hello.txt` and "+hello world".*
- [x] With commits, the dialog still opens on "All changes". *`initialDiffScope` test.*
- [x] When issue details shows Review Changes, it is the primary button, and Start Work or Continue Work is secondary; without Review Changes, the work button stays primary. *`PropertySidebar.test.tsx` "button priority" tests. Live.*
- [x] The primary action is always the top button. *`PropertySidebar.test.tsx` checks the order. Live: Review Changes above Continue Work.*
- [x] `make test` passes. *Run before commit.*
