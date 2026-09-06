# Walkthrough: Worktree-aware diff endpoints + mnemonicPrefix normalization

## What changed

Two related bugs in `internal/exponential/review.go` and `internal/server/handlers.go` prevented the merge view from showing file lists and diffs for worktree-based issues.

## Bug 1: Diff/files endpoints ignore worktree directory

`GetWorkingTreeDiffText()` and `ListWorkingTreeFilesChanged()` ran `git diff HEAD` without `-C <dir>`, targeting the server's CWD (the hub checkout on main) instead of the issue's worktree. Same class of bug as xpo-67a1c4, which fixed `computeBranchStats` but missed the review endpoints.

**Fix:** Added a `dir` parameter to both functions and the underlying `listFilesChangedFromDiff`. When non-empty, all git commands prepend `-C <dir>`. The handlers in `handlers.go` now resolve the worktree path via `FindWorktreeForBranch` before calling these functions. When `FindWorktreeForBranch` returns `""` (non-worktree branch), the empty string is passed through and no `-C` flag is added — preserving the original behavior.

## Bug 2: `diff.mnemonicPrefix` breaks diff parsing

When the user has `diff.mnemonicPrefix = true` in their git config, `git diff` uses context-dependent prefixes (`c/`/`w/` instead of `a/`/`b/`). The frontend's `parseDiffByFile` regex matches ` b/(.+)$` to extract filenames, which silently fails with non-standard prefixes.

**Fix:** Added `--src-prefix=a/` and `--dst-prefix=b/` flags to all git commands that produce raw diff text: `GetWorkingTreeDiffText`, `GetDiffText`, and `GetCommitDiffText`. This normalizes output regardless of user git config. The `--numstat`/`--name-status` variants are unaffected since they don't use path prefixes.

## Key decisions

- Fixed in the backend rather than making the frontend parser more flexible. Normalizing at the source is more robust — there are multiple prefix variants (`c/`, `w/`, `i/`, `o/`) and future diff consumers shouldn't need to handle them all.
- Subsumes xpo-90b533 (marked as duplicate), which addressed only bug 1.
