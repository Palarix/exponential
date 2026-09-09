# Walkthrough: Highlight uncommitted files in MergeView file tree

## Overview

This change gives users immediate visibility into which files have uncommitted changes when reviewing a branch in MergeView. Dirty files get amber-colored icons in the file tree, and the file icon color semantics were corrected from diff-line-based to file-status-based.

## What changed

### Backend: `internal/server/handlers.go`

The `handleMergeability` handler already called `WorktreeDirtyFiles()` and stuffed the result into a blocker's `Files` array. The change extracts the dirty file list into a variable and adds it as a top-level `dirty_files` field in the JSON response. This avoids coupling the frontend to the blocker message structure. When the worktree is clean, `dirty_files` is an empty `[]string{}` (never null).

### Frontend types: `web/src/api/client.ts`

Added `dirty_files: string[]` to the `Mergeability` interface — a flat list of file paths with uncommitted or untracked changes in the worktree.

### Utility: `web/src/components/IssueDetail/diff-utils.ts`

Added `hasDirtyDescendant(dirPath, dirtyFiles)` — checks whether any file in a `Set<string>` is a descendant of a directory path. Uses `dirPath + "/"` as the prefix to avoid false matches on similarly-named directories (e.g. `src` does not match `src-old/file.ts`). Exported for potential future use; not currently called in the component after review feedback removed directory-level amber propagation.

### Component: `web/src/components/IssueDetail/MergeView.tsx`

Three changes here:

1. **`MergeView` (top-level)**: Builds a `dirtyFilesSet: Set<string>` via `useMemo` from `mergeability.dirty_files` and passes it down through `FilesTab`.

2. **`FileTreeView`**: Accepts a new `dirtyFiles: Set<string>` prop. The file icon color logic was rewritten from diff-line-content-based (has `+` lines only → green) to file-status-based using diff headers:
   - Amber (`text-amber-500`) — `dirtyFiles.has(node.path)`, with "Uncommitted changes" tooltip
   - Green (`text-green-500`) — diff contains `new file mode` header
   - Red (`text-red-500`) — diff contains `deleted file mode` header
   - Muted — everything else (modified files)
   
   Amber takes priority. Directory nodes keep their default muted text color — only file icons are colored.

3. **Inline warning removal**: The merge bar previously rendered `mergeability.warnings[0]` as amber text listing all dirty files. This was removed since the file tree now conveys the same information visually. The contextual hint text (e.g. "Resolve blockers to enable merging") now always shows. The blocker detail modal still lists dirty files for the full picture.

### Tests: `web/src/components/IssueDetail/MergeView.test.ts`

Five new tests for `hasDirtyDescendant`: direct child match, nested descendant match, no match, empty set, and the prefix false-positive guard (`src` vs `src-old`).

## Key decisions

- **Amber on icons only, not directories**: Initial implementation colored directory text/chevrons amber when they contained dirty descendants. Review feedback asked for only file icons to be colored — directories stay muted. The `hasDirtyDescendant` utility remains exported for xpo-6211f5 or future use.

- **File-status-based coloring over line-content-based**: Review feedback identified a semantics mismatch — the original green/red coloring used diff line content (only additions → green), but file tree convention is green=new file, red=deleted. Fixed to use diff header lines (`new file mode` / `deleted file mode`) instead.

- **Top-level `dirty_files` field**: Rather than requiring the frontend to parse the blocker structure, dirty files are surfaced as a dedicated response field. Cleaner contract and no coupling to blocker message format.

- **Uncommitted diff toggle deferred**: During review, the idea of an "Uncommitted" vs "All changes" toggle emerged. Filed as xpo-6211f5 since it needs a new backend endpoint.
