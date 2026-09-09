## What was built

A scope toggle in the MergeView Files tab that switches between "All changes" (committed branch diff vs main) and "Uncommitted" (working tree vs HEAD). When the uncommitted view is active, the UI auto-refreshes every 3 seconds so users can follow an agent's in-progress work in real time.

## How it works

### Backend

Two new functions in `internal/exponential/review.go` handle the uncommitted diff with untracked file support:

- **`GetWorkingTreeFullDiffText(dir)`** — runs `git diff HEAD` for tracked changes, then appends synthetic unified-diff entries for each untracked file (via `git ls-files --others --exclude-standard`). Binary files (null byte in first 512 bytes) and files over 1 MB are skipped. The synthetic entries use standard diff format (`new file mode 100644`, `--- /dev/null`, `+++ b/{path}`, hunk header, `+` lines) so `parseDiffByFile` on the frontend handles them identically to real git diffs.

- **`ListWorkingTreeAllFilesChanged(dir)`** — merges `ListWorkingTreeFilesChanged` (tracked changes) with untracked files, adding them as status `A` entries with line-counted insertions.

Both endpoints (`/api/issues/{id}/diff` and `/api/issues/{id}/files`) now accept an optional `?scope=uncommitted` query parameter. A shared `resolveWorkingDir(branch)` helper in `handlers.go` resolves the working directory — worktree path if one exists, or `""` if the branch is the currently checked-out branch — enabling the toggle in both worktree and branch mode. Returns 404 if neither applies.

A key fix during review: the original handlers had a `Commits == 0 && HasUncommitted` fallback that made the default (non-scoped) diff endpoint return working-tree changes instead of committed branch changes. This caused "All changes" and "Uncommitted" to show identical content when all work was uncommitted. The fallback was removed — "All changes" now always returns the committed branch-vs-main diff via `git diff base...branch`.

### Frontend

**`web/src/api/client.ts`** — `fetchIssueDiff` and `fetchIssueFiles` accept an optional `scope?: "uncommitted"` parameter, appending `?scope=uncommitted` to the URL when set.

**`web/src/components/IssueDetail/MergeView.tsx`** — `MergeView` manages three new state variables: `diffScope` (`"all"` | `"uncommitted"`), `uncommittedDiff`, and `uncommittedLoading`. A `refreshUncommitted` callback fetches the uncommitted diff with an in-flight guard (`refreshingRef`) to prevent overlapping requests. A `useEffect` sets up a 3-second `setInterval` when `diffScope === "uncommitted"` AND `activeTab === "files"`, clearing on scope/tab change or unmount. `activeFileDiffs` selects between the committed and uncommitted parsed diffs based on scope.

`FilesTab` renders a segmented toggle ("All changes" / "Uncommitted") above the file filter, using the same button group pattern as the Unified/Split toggle in `DiffViewer`. Switching scope clears `selectedFile`. When the uncommitted scope is active and the diff is empty (and not loading), a centered "No uncommitted changes" empty state replaces the two-panel layout.

## Key decisions

- **Query parameter, not new endpoints** — `?scope=uncommitted` is a modifier on the same resource, keeping the API surface small.
- **Synthetic diffs for untracked files** — read-only approach, no `git add -N` that could interfere with the agent's index.
- **3-second polling** — fast enough for real-time feel, with in-flight guard to prevent request pileup. Only active when the uncommitted view is showing.
- **Removed the `Commits == 0` fallback** — the toggle makes the old workaround unnecessary and actively harmful (both views showed the same data).

## Acceptance criteria

- [x] Segmented toggle ("All changes" / "Uncommitted") appears in Files tab left panel, above the file filter
- [x] Toggle is always visible when the issue has a branch (worktree or branch mode)
- [x] "All changes" shows committed branch-vs-main diff only
- [x] "Uncommitted" shows only uncommitted and untracked changes (working tree vs HEAD)
- [x] Works in both worktree mode and branch mode (via `resolveWorkingDir`)
- [x] Untracked files appear with full content as additions (synthetic diff)
- [x] File tree filters to only dirty/untracked files when "Uncommitted" is selected
- [x] Empty state ("No uncommitted changes") when worktree is clean
- [x] Auto-refresh every 3 seconds when "Uncommitted" is active
- [x] Auto-refresh stops on scope/tab change or unmount
- [x] Switching scopes clears selected file
- [x] 4 Go tests (full diff, binary skip, empty worktree, all-files listing) and 3 frontend tests (synthetic diff parsing, mixed diffs, line numbering)
