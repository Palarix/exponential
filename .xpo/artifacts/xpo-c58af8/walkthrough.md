# Walkthrough: Squash/FF merge overwrites main's issues.db when worktrees disabled

## Problem

When `worktrees: false`, `xpo merge` with squash or ff-only strategy loses `issues.db` events.
The `.gitattributes` file declares `merge=union` for `.xpo/issues.db`, which correctly unions
event lines during a 3-way merge (`--no-ff`). However, squash and ff-only merges don't invoke
the merge driver — they apply the branch's diff directly, replacing main's `issues.db` with the
branch's version.

The original scenario: a user cascades an epic and all sub-issues from BACKLOG → PLANNED
(uncommitted on main), then `xpo start` creates a feature branch carrying those uncommitted
events. After work completes and the branch is squash-merged, main's committed `issues.db`
(which never had the PLANNED events) overwrites the branch's version, reverting everything
to BACKLOG.

In worktree mode this never happens — the hub stays on `main` and all `issues.db` writes
target the hub's copy, so there's no divergence.

## What Changed

### `internal/exponential/merge.go`

**Pre-merge snapshot** (before `runGitMerge`): When `!useWorktrees` and the strategy is
squash or ff-only, main's `issues.db` is read into memory via `os.ReadFile`.

**Post-merge union** (after a successful merge): The post-merge `issues.db` (now the branch's
version after squash) is read, and `unionLines` combines both — all lines from main first,
then any branch-only lines not already present. The result is written back before MERGE/DONE
events are appended.

**`unionLines` helper**: Takes two `[]byte` slices (line-delimited), builds a set from the
base lines, and appends any lines from theirs not in the set. This simulates `merge=union`
semantics for append-only event logs.

The guard `!useWorktrees && opts.Strategy != MergeStrategyMerge` ensures:
- Worktree mode is completely unaffected (`mainIssuesDB` stays `nil`)
- `--no-ff` merges are unaffected (the real `merge=union` driver handles them)

### `internal/exponential/merge_test.go`

**`TestMergeIssue_SquashPreservesMainIssuesDB`**: End-to-end test that sets up a repo with
`worktrees: false`, creates diverged `issues.db` on both main and a feature branch, runs a
squash merge, and verifies all events from both sides survive — including the MERGE and DONE
events appended by the merge flow.

**`TestUnionLines`**: Unit tests for the helper covering disjoint lines, overlapping lines,
identical content, and empty inputs.
