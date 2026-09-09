## What

Add a scope toggle to the MergeView Files tab that lets the user switch between viewing the full branch diff ("All changes" — branch vs main) and only uncommitted/untracked changes ("Uncommitted" — working tree vs HEAD). When the uncommitted view is active, the file tree and diff pane update to show only dirty files, and the view auto-refreshes on a short interval so the user can follow an agent's work in real time.

## Why

When an agent is actively making changes in a worktree (or on the checked-out branch), the MergeView mixes committed work with in-progress edits. The user can't easily see what just changed. An uncommitted-only view isolates the live delta, turning MergeView into a real-time follow-along tool during agent sessions.

Depends on xpo-110b09 (dirty file highlighting), which already surfaces `dirty_files` from the mergeability response.

## Acceptance Criteria

- [ ] A segmented toggle ("All changes" / "Uncommitted") appears in the Files tab left panel, above the file filter
- [ ] Toggle is always visible when the issue has a branch (worktree or branch mode)
- [ ] "All changes" (default) shows the existing branch-vs-main diff — no behavior change
- [ ] "Uncommitted" shows only uncommitted and untracked changes (working tree vs HEAD)
- [ ] Works in both worktree mode (agent worktree) and branch mode (branch is currently checked out)
- [ ] Untracked (new) files appear in the uncommitted diff with full content as additions
- [ ] File tree filters to only dirty/untracked files when "Uncommitted" is selected
- [ ] When "Uncommitted" is active and there are no uncommitted changes, a centered empty state reads "No uncommitted changes"
- [ ] When "Uncommitted" is active, the view auto-refreshes every 3 seconds
- [ ] Auto-refresh stops when the user switches back to "All changes" or leaves the Files tab
- [ ] Switching scopes clears stale state (selected file resets)
- [ ] Tests cover the new backend query parameter and the new frontend components

## Flow

### Backend

1. **`internal/server/handlers.go` — `handleGetIssueDiff`**: read optional `scope` query parameter. When `scope=uncommitted`, resolve the working directory (worktree via `FindWorktreeForBranch`, or `""` if the branch is the current branch) and return `GetWorkingTreeFullDiffText(dir)` (new function). Return 404 if neither worktree nor current branch applies. When `scope` is absent/empty, keep current behavior.

2. **`internal/server/handlers.go` — `handleGetIssueFiles`**: same `scope=uncommitted` parameter. When set, return `ListWorkingTreeAllFilesChanged(dir)` (new function). Same working-directory resolution logic.

3. **`internal/server/handlers.go` — new helper `resolveWorkingDir(branch string) (string, bool)`**: returns `(worktreePath, true)` if a worktree exists, `("", true)` if the branch is the current branch, `("", false)` otherwise. Shared by both handlers.

4. **`internal/exponential/review.go` — new `GetWorkingTreeFullDiffText(dir string) string`**: combines `git diff HEAD` output with synthetic diff entries for untracked files. For each untracked file (from `git ls-files --others --exclude-standard`, with `-C dir` when non-empty):
   - Read the file content
   - Generate a unified diff header (`diff --git a/{path} b/{path}`, `new file mode 100644`, `--- /dev/null`, `+++ b/{path}`) followed by `@@ -0,0 +1,{linecount} @@` and all lines as `+` additions
   - Skip binary files (null byte in first 512 bytes)
   - Skip files > 1 MB
   - Filter out `.xpo/` paths

5. **`internal/exponential/review.go` — new `ListWorkingTreeAllFilesChanged(dir string) []FileStat`**: combines `ListWorkingTreeFilesChanged(dir)` with untracked files from `git ls-files --others --exclude-standard`, adding them as status `A` entries with line-counted insertions.

### Frontend

6. **`web/src/api/client.ts`**: add optional `scope` parameter to `fetchIssueDiff` and `fetchIssueFiles`:
   - `fetchIssueDiff(issueId: string, scope?: "uncommitted")`
   - `fetchIssueFiles(issueId: string, scope?: "uncommitted")`
   Append `?scope=uncommitted` to the URL when the parameter is set.

7. **`web/src/components/IssueDetail/MergeView.tsx` — `MergeView`**:
   - Add state: `diffScope: "all" | "uncommitted"` (default `"all"`)
   - Add state: `uncommittedDiff: string`, `uncommittedLoading: boolean`
   - Add `refreshUncommitted()` callback that fetches `fetchIssueDiff(id, "uncommitted")`
   - Set up a 3-second `setInterval` that calls `refreshUncommitted()` when `diffScope === "uncommitted"` AND `activeTab === "files"`. Clear interval on scope/tab change or unmount. Guard against overlapping requests (skip tick if previous fetch is in flight).
   - Compute `activeFileDiffs`: when `"all"`, use existing `fileDiffs`; when `"uncommitted"`, parse `uncommittedDiff`
   - Pass `diffScope`, `onScopeChange` down to `FilesTab`

8. **`web/src/components/IssueDetail/MergeView.tsx` — `FilesTab`**:
   - Accept new props: `diffScope`, `onScopeChange`, `uncommittedLoading`
   - Render a segmented toggle above the file filter input (inside the left panel header area). Use the same segmented button group pattern as the Unified/Split toggle in DiffViewer (lines 1043-1057).
   - When scope changes, clear `selectedFile`
   - When uncommitted scope is active and `fileDiffs` is empty (and not loading), render a centered empty-state message instead of the two-panel layout

## Decisions

- **Query parameter on existing endpoints, not new endpoints.** Keeps the API surface small. The `scope=uncommitted` parameter is a modifier on the same resource. Alternative: separate `/api/issues/{id}/uncommitted-diff` endpoint — rejected as unnecessary proliferation.

- **Synthetic diff for untracked files, not `git add -N`.** `git add -N` modifies the index, which could interfere with the agent's work. Generating synthetic diff entries is read-only and safe. Alternative: `git diff --no-index /dev/null <file>` per file — rejected as slower (one process per file).

- **3-second polling interval.** Fast enough to feel real-time. Alternatives: WebSocket/SSE (complex, overkill for a local tool), filesystem watch (platform-dependent). Only active when the uncommitted view is showing.

- **Toggle always visible when branch exists.** Works for both worktree mode (dedicated working directory) and branch mode (branch is currently checked out). No need for a `has_worktree` flag — the backend resolves the working directory from whichever mode applies, and returns empty when neither does.

- **No localStorage persistence for scope.** "All changes" is always the correct default. Persisting "Uncommitted" could show a confusing empty view.

- **Uncommitted diff includes both staged and unstaged changes.** `git diff HEAD` shows everything not yet committed. This is the correct semantic — the user wants the full uncommitted delta.

## Edge Cases

- **HIGH — Binary files in untracked set**: skip binary files in synthetic diff (detect null bytes in first 512 bytes). Show in file list with "(binary)" note but no diff content.
- **MEDIUM — Large untracked files**: cap synthetic diff at 1 MB per file. Show "File too large to display" placeholder.
- **MEDIUM — Branch is neither worktree nor current branch**: `resolveWorkingDir` returns `false`, endpoint returns 404. Frontend can hide the toggle or show a disabled state — but this scenario is uncommon (the user is viewing MergeView for an active issue).
- **LOW — Race between polling and agent commits**: files may disappear from uncommitted view between polls. This is correct — they moved to committed.
- **LOW — Rapid polling with slow diff**: guard against overlapping requests — skip a poll tick if the previous fetch is still in flight.
