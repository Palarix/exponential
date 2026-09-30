# Spec: merge fails with misleading conflict error when run from inside the issue's worktree

## What

Make `MergeIssue` (`internal/exponential/merge.go`):

1. **Independent of the process working directory:** every git call targets the hub checkout.
2. **Record only after the commit:** MERGE/DONE events are appended only after the code commit exists. A failed merge never writes events, so there is nothing to undo.
3. **Honest in its errors:** git's real output is shown, and the conflict advice appears only for real conflicts.

## Why

The MCP server can be running with its working directory inside the issue's worktree. Elysium does this after `link_worktree`, and the server stays there after `unlink_worktree`. Today, the squash and commit run in that working directory. The squash merges the branch into itself, and `git commit` then exits 1. By that point MERGE and DONE are already appended and staged in the hub, so the issue ends up DONE with no merge commit and an error that points at a nonexistent conflict. Seen in practice: `srv-ebf547` in servaint.

The root problem is ordering: the code commit, the step most likely to fail (hooks, nothing to commit, conflicts), runs after the permanent records are written. Undoing those records afterwards isn't an option. `issues.db` is shared by every agent on the hub. Restoring a saved copy, or rewriting the file to remove lines, would destroy events that other agents wrote in the meantime.

## Invariants

- **I1: the log only grows.** `MergeIssue` never truncates, restores, or rewrites `issues.db` to undo anything. Events other agents append at any point during the merge survive, whether the merge succeeds or fails.
- **I2: DONE implies merged.** A MERGE/DONE event exists only if base contains the code commit.
- **I3: a failed merge leaves the hub clean.** No merge in progress, nothing staged, and the issue keeps its status, branch, and worktree.
- **I4: one commit per merge (squash/no-ff).** The code and the xpo bookkeeping still land in a single commit, as today (for example, 65b9e10).

## How

### 1. Hub-anchored git

Add `hubGit(args ...string) *exec.Cmd`, which runs `git -C storage.HubRoot() ...`. Use it for every git call in the merge path: merge, commit, amend, add, abort/reset, `resolveRef`, `deleteBranch`, and in branch mode the checkout of base. `WorktreeRemove` and `WorktreeList` in `git.go` also switch to `-C hub`. That's safe for their other caller, `start.go`. Use `-C` and never `os.Chdir`: the working directory is shared by the whole process, and the MCP and web servers handle requests concurrently.

### 2. New order inside `WithGitLock`

| Step | squash | no-ff | ff | On failure |
|---|---|---|---|---|
| a. Preconditions (hub on base, or branch mode checks out base) | ✓ | ✓ | ✓ | return the error; nothing has changed |
| b. `merge` | `--squash` | `--no-ff --no-commit` | `--ff-only` | `merge --abort` → `reset --merge` (hub). Error shows git's output. Conflict advice only if the output contains `CONFLICT`. **No events written.** |
| c. Code commit | `commit -m msg` | `commit -m msg` | (none: the branch tip is the commit) | abort/reset (hub). Error: `commit failed — nothing was merged, <id> is still <STATUS>` plus git's output (for example, hook output). **No events written.** |
| d. Record | append MERGE + DONE | same | same | Code is already on base; do **not** reset it. Error: `merged as <sha> but failed to record: <err>`, with the recovery step (`update <id> status=DONE`). |
| e. Bookkeeping | `add .xpo/issues.db .xpo/artifacts/<id>`, then `commit --amend --no-edit --no-verify` | same | `add ...`, then `commit -m "xpo: merge <id>" --no-verify` | `reset -q -- .xpo` (unstage only; the working file isn't touched). Merge **succeeds** with a warning: the events are recorded but uncommitted and will go in with the next commit, which is the normal per-merge cadence. |
| f. Cleanup | remove worktree, delete branch | same | same | warning (unchanged) |

Notes:
- **no-ff:** `--no-commit` plus an explicit `commit` makes all three strategies share one commit step whose output is captured. Previously the `--amend` error was silently ignored.
- **Branch mode (not worktrees):** the existing `issues.db` union rewrite for squash/ff stays where it is, before step d. It is a rewrite, so it doesn't satisfy I1, but branch mode is being removed in xpo-863802. Not in scope here.
- **`MergePayload.MergeSHA`** stays `""` as it is today. A commit can't contain its own SHA.

### 3. Error surfacing

`runGitMerge` and a new `runHubCommit` both return `*mergeError{err, output}` with the combined output. The top-level message is built by what failed (merge conflict, merge other, commit, record) instead of one fixed conflict message.

## Concurrency analysis

- **Two merges, or a merge and a start:** serialized by `git.lock`, as today.
- **Another agent writing events during a merge:** its appends go to the hub's `issues.db`. The merge only appends (step d) and stages whatever is on disk (step e), so other agents' uncommitted events get committed along with it, as today. The failure paths never touch the file, so I1 holds.
- **Known remaining race:** `AppendEventCollapsed` reads, modifies, and renames `issues.db` without a lock, so two concurrent writers can lose an event whether or not a merge is running. It's filed as xpo-d6288f and out of scope here.

## Decisions (confirmed by user)

1. **Amend, not a separate bookkeeping commit.** This matches the manual workflow used today (one commit per merge). `MergeSHA` stays `""`.
2. **`--no-verify` on the bookkeeping commit.** Hooks already ran on the code commit in step c.

## Acceptance Criteria

- [ ] With the working directory inside the issue's worktree, squash and `--no-ff` merges succeed: one commit on base containing the code, `.xpo/issues.db`, and the artifacts; worktree removed; branch deleted; issue DONE.
- [ ] A hub `pre-commit` hook that fails: the error contains the hook's output and no conflict advice; no MERGE/DONE events for the issue; the issue is still DOING; the hub index is clean with no merge in progress; the branch and worktree still exist.
- [ ] **I1 under concurrency:** a `pre-commit` hook that appends a line to `issues.db`, standing in for another agent, and then fails leaves that line in place. The same hook with exit 0 leaves the line in place and it's included in the commit.
- [ ] Pre-existing uncommitted events from other issues are kept on failure and committed on success.
- [ ] A real content conflict still returns the conflict advice with the file list.
- [ ] Bookkeeping failure (a `prepare-commit-msg` hook that fails when the source is `commit`, which `--no-verify` doesn't skip): the merge returns success with a warning; the events are in `issues.db` but uncommitted and unstaged; the code commit is on base.
- [ ] Existing merge tests pass; `make test` is green.

## Out of scope

- xpo-d6288f: the lock around event writes.
- xpo-863802: removing branch mode and its union rewrite.
- Elysium not restarting the provider session on unlink.
- Repairing `srv-ebf547` in servaint by hand.
