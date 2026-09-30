# Walkthrough: merge fails with misleading conflict error when run from inside the issue's worktree

## What was broken

`MergeIssue` had two independent defects that combined badly.

1. **Git calls ran in the process working directory.** Only the preflight checks and `git add` used `-C <hub>`. The merge, commit, abort, `rev-parse`, `branch -D`, and worktree calls ran wherever the process happened to be. An MCP server started inside the issue's worktree (Elysium does this after `link_worktree`, and it stays there after `unlink_worktree`) squashed the branch into itself ("nothing to squash", exit 0), and then `git commit` exited 1.
2. **Events were written before the risky step.** MERGE and DONE were appended to `issues.db` and staged *before* `git commit` ran. When the commit failed, nothing undid them. The issue showed DONE with no merge commit, and the error was a bare `exit status 1` followed by fixed "resolve the conflict" text.

## Why "undo on failure" was rejected

The first draft of the spec saved a copy of `issues.db` and restored it on failure. That is wrong. `issues.db` is shared by every agent working on the hub, and any restore or rewrite would delete events that other agents appended in the meantime. The rule that came out of this: **`MergeIssue` never truncates, restores, or rewrites the log to undo anything.** The only way to satisfy that is to make sure there is never anything to undo, which means moving the writes after the step that can fail.

## New flow (`internal/exponential/merge.go`)

Inside `WithGitLock`, every git call now goes through `hubGit`:

1. **Preconditions:** the hub must be on base (worktree mode), or merge checks out base on the hub (branch mode).
2. **`git merge`:** `--squash`, `--ff-only`, or `--no-ff --no-commit`. On failure: `abortMerge()`, then `mergeFailure()`. No events are written.
3. **Code commit** (not for ff): `runHubCommit("-m", msg)`. User hooks run here. On failure: `abortMerge()` and `commit failed — nothing was merged, <id> is still <STATUS>`, plus git's output. No events are written.
4. **`recordMerge`:** appends MERGE and the DONE transition. The code is already on base at this point, so "DONE implies merged" always holds. If the append fails, the error explains that the code *is* merged and must not be merged again.
5. **Bookkeeping:** `git add` the `.xpo/issues.db` and artifacts, then `commit --amend --no-edit --no-verify` (a fresh `--no-verify` commit for ff). This matches the manual "one commit per merge" workflow the user already follows. If it fails, the `.xpo` changes are unstaged (`reset -q -- .xpo` leaves the working file alone) and the merge returns success with a warning. The events remain uncommitted, which is the normal state between merges.
6. **Cleanup:** remove the worktree and delete the branch, both on the hub.

### Details worth knowing

- **`hubGit` versus `os.Chdir`:** the working directory is shared by the whole process, and the MCP and web servers handle requests concurrently. Changing it would move every other in-flight request too.
- **`abortMerge` is safe with concurrent appends.** `merge --abort` and `reset --merge` reset the index. They keep files whose only differences are between the index and the working tree, so lines other agents appended to `issues.db` stay on disk.
- **`--no-ff --no-commit` plus an explicit commit** gives all strategies one commit step whose output is captured. Before, `--no-ff` committed inside `git merge`, and the following `--amend` error was ignored.
- **Why `--no-verify` on the amend** (confirmed by the user): hooks already validated the code commit. The amend only adds `.xpo/` files, and re-running lint or test hooks would double merge time.
- **`MergePayload.MergeSHA` stays `""`.** A commit can't contain its own SHA.
- **Branch mode's `unionLines` rewrite** still exists and now runs after the code commit. It rewrites the file, but branch mode is being removed in xpo-863802.
- In `git.go`, `WorktreeRemove` and `WorktreeList` use `hubGit`, and `WorktreeRemove` now includes git's stderr in its error.

## Tests (`internal/exponential/merge_cwd_test.go`)

`setupHubWorktreeRepo` mirrors `xpo start`: a hub on main, worktrees enabled, and a linked worktree under `.xpo/worktrees/`. `chdirWorktree` puts the test process inside the worktree, where the failing MCP server was running. Concurrency is simulated with git hooks that append an event line to `issues.db` while `git commit` runs. Before the fix, these tests reproduced the servaint failure exactly: `merge failed: exit status 1`, DONE, and `issues.db` staged.

## Acceptance Criteria

- [x] Merge from inside the worktree succeeds for squash, `--no-ff`, and ff: one commit holds the code and `.xpo`, the worktree is removed, the branch is deleted, and the issue is DONE. Evidence: `TestMergeIssue_FromInsideWorktree/{squash,merge,ff}`.
- [x] A failing `pre-commit` hook yields an error with the hook's output and no conflict advice. No MERGE event is written, the issue stays DOING, the hub is clean, and the branch and worktree survive. Evidence: `TestMergeIssue_CommitHookFails_WritesNoEvents/{squash,merge}`.
- [x] The log only grows, even under concurrency. A line appended by the hook survives a failed merge and is included in the commit of a successful one. Evidence: the same test above, plus `TestMergeIssue_ConcurrentAppendDuringCommit_IsCommitted`.
- [x] Pre-existing uncommitted events are kept on failure and committed on success. Evidence: the `preexisting` assertions in both tests above.
- [x] A real conflict still gives the conflict advice with the file list. Evidence: `TestMergeIssue_ConflictFromInsideWorktree`, plus the existing `TestMergeIssue_Conflict`.
- [x] If the bookkeeping amend fails, the merge succeeds with a warning, the events are uncommitted and unstaged, and the code is on base. Evidence: `TestMergeIssue_BookkeepingCommitFails_MergeSucceedsWithWarning`.
- [x] Existing merge tests pass and `make test` is green (`go vet`, eslint, 369 vitest tests, all Go packages).

## Follow-ups

- xpo-d6288f: event writes (`AppendEvent`, `AppendEventCollapsed`) have no lock between processes, so they can lose events independently of merge.
- `srv-ebf547` in servaint needs a one-off manual cleanup.
