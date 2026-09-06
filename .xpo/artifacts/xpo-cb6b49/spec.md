# Spec: `xpo merge` should tolerate untracked files on the hub checkout

## What

The pre-merge cleanliness check in `cmd/exponential/merge.go` (lines 60-72) calls `IsWorkingTreeCleanIgnoringXpo()`, which rejects **any** non-`.xpo/` dirty file — including untracked files that have no bearing on the merge. This causes `xpo merge` to fail when the hub has scratch files like `idea.md`, and the generic error message ("commit or stash them first") sends agents into a destructive stash-thrash loop.

## Why

This is a recurring workflow failure. Users discuss ideas with AI on `main`, producing scratch files. When they later merge a feature worktree, the untracked files block the merge unnecessarily. Agents, reading the generic error, start stashing on both `main` and the worktree — corrupting `.xpo/issues.db` (which flows hub-ward by design) and polluting git history.

## How

### Change 1: Relax the cleanliness check

Replace `IsWorkingTreeCleanIgnoringXpo()` with a smarter check that categorizes dirty files:

1. **`.xpo/` files** — always ignored (existing behavior, correct)
2. **Untracked files (`??`)** — ignored by the pre-check. Git merge itself will error if the incoming branch introduces a file that collides with an untracked file, and that error is specific and actionable. Our pre-check adds nothing here.
3. **Modified tracked files** — these are the only ones that can cause merge conflicts. Check whether any overlap with files changed in the merge branch (`git diff --name-only <base>...<branch>`). If they don't overlap, they're safe — ignore them. If they do overlap, fail with a specific error.

New function signature:

```go
// HubCleanForMerge checks whether the hub working tree is safe to merge
// the given branch. Returns nil if clean enough, or an error listing the
// specific conflicting files.
func HubCleanForMerge(branch string) error
```

### Change 2: Agent-safe error messages

When the check does fail, the error must list the specific files and give a narrow action:

```
Error: these tracked files have local changes that conflict with the incoming branch:
  - src/config.go (modified locally, also changed on xpo-cb6b49-fix-merge)
Commit or remove these changes before merging.
Do NOT stash — .xpo/issues.db must not be stashed.
```

The "Do NOT stash" line is intentional — agents read error messages as instructions.

### Change 3: Remove the old functions

- Delete `IsWorkingTreeClean()` (only used in non-worktree mode, same problem)
- Delete `IsWorkingTreeCleanIgnoringXpo()`
- Replace both call sites in `cmd/exponential/merge.go` with `HubCleanForMerge(branch)`

For the non-worktree path, `HubCleanForMerge` still works — untracked files are harmless, and conflicting tracked files are the only real concern.

### Change 4: Update `MergeIssue()` merge failure message

In `internal/exponential/merge.go` line 160, the merge-failure fallback message should also be agent-safe:

```
merge failed: <git error>
Do NOT stash or reset. Resolve the conflict in the listed files, then re-run xpo merge.
```

## Flow

1. User calls `xpo merge xpo-cb6b49`
2. Issue resolved, branch stats checked (existing logic, unchanged)
3. **New**: `HubCleanForMerge(branch)` runs:
   a. `git status --porcelain` on hub — collect non-`.xpo/` dirty files
   b. Filter out untracked (`??`) — those are fine
   c. If no modified tracked files remain → pass
   d. If modified tracked files remain → `git diff --name-only <base>...<branch>` to get branch's changed files
   e. Intersect modified-locally with changed-on-branch
   f. If intersection is empty → pass
   g. If intersection is non-empty → return error listing the conflicting files
4. Strategy prompt, merge execution, cleanup (existing logic, unchanged)

## Acceptance Criteria

- [ ] `xpo merge` succeeds when hub has untracked files (e.g. `idea.md`) that don't exist in the merge branch
- [ ] `xpo merge` succeeds when hub has untracked files that also exist in the merge branch but are identical
- [ ] `xpo merge` succeeds when hub has modified tracked files that aren't touched by the merge branch
- [ ] `xpo merge` fails with a specific, file-listing error when hub has modified tracked files that conflict
- [ ] Error messages never suggest stashing; they explicitly warn against it
- [ ] `.xpo/issues.db` is never at risk of being stashed or discarded
- [ ] `IsWorkingTreeClean()` and `IsWorkingTreeCleanIgnoringXpo()` are removed
- [ ] Existing tests updated; new tests cover the relaxed check
- [ ] Non-worktree merge path also uses the new check
