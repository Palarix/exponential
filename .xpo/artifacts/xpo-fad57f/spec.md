## What

The mergeability check (`HubCleanForMerge`) compared working-directory state against the incoming branch's changed files. Uncommitted changes that overlapped with branch files falsely reported merge conflicts — even when committed refs merged cleanly. Additionally, the worktree could be silently deleted on merge with untracked files still in it.

## Why

Users saw "local changes conflict with the incoming branch" when checking mergeability before committing. This was misleading — the committed branch tip and main may have no conflict at all. Separately, `git worktree remove --force` on merge silently destroys any uncommitted or untracked files in the worktree.

## How

### 1. `CheckMergeConflicts(branch string) ([]string, error)`

Uses `git merge-tree --write-tree <base> <branch>` to test committed refs. Unaffected by working-directory state. Returns conflicting file names.

### 2. `HubRequireCleanTree() error`

Simple guard for dirty tracked files in the hub (excluding `.xpo/` and untracked). Message: "uncommitted changes — commit or discard before merging."

### 3. `WorktreeDirtyFiles(branch string) []string` / `WorktreeRequireClean(branch string) error`

Checks the worktree for the branch for any uncommitted or untracked files (excluding `.xpo/`). Blocks merge to prevent silent data loss on worktree removal.

### 4. `handleMergeability` — structured response

Returns structured blockers `{message, files[]}` instead of raw error strings. Merge conflicts and dirty worktree files are blockers; dirty hub files are warnings.

### 5. `handleMergeIssue` — separate checks

Checks merge conflicts, hub cleanliness, and worktree cleanliness independently with distinct error messages.

### 6. Frontend — blocker detail modal

Short clickable blocker summary in the merge status bar. Clicking opens a Modal dialog with the explanation, file list, and remediation instructions. Merge button disabled when there are zero commits.

### 7. "Review Changes" button rename

The sidebar button that opens the merge view always reads "Review Changes" (was "Merge Branch" when commits existed).

## Acceptance Criteria

- [x] `GET /api/issues/{id}/mergeability` returns `can_merge: true` when committed refs merge cleanly, regardless of working-directory state
- [x] Uncommitted hub changes appear as `warnings`, not `blockers`
- [x] Actual merge conflicts between committed refs appear as structured `blockers`
- [x] Dirty worktree (uncommitted or untracked files) appears as a structured `blocker`
- [x] The merge action refuses when the working tree or worktree is dirty, with clear messages
- [x] Merge button disabled with "No commits to merge" when zero commits ahead
- [x] Blocker detail modal uses the project's Modal component with file list
- [x] All existing tests pass (adapted)
- [x] New tests cover `CheckMergeConflicts`, `HubRequireCleanTree`, and `WorktreeRequireClean`
- [x] `make test` passes