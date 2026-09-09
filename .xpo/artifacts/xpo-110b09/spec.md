# Spec: Highlight uncommitted files in MergeView file tree

## What

Add amber visual indicators to file icons in the MergeView file tree for files with uncommitted or untracked changes in the worktree. Also fix the file icon color semantics to use file-level status (new/deleted/modified) instead of diff line content.

## Why

Currently the MergeView shows a blocker message when there are uncommitted changes, but the user has to read the blocker detail to find out which files are dirty. Highlighting them directly in the file tree makes the information visible at a glance — the user can see exactly which files need attention without extra clicks.

## How

### Backend (handlers.go)

Add a top-level `dirty_files` string array to the mergeability response so the frontend can consume it directly without parsing blocker internals.

### Frontend API types (client.ts)

Add `dirty_files: string[]` to the `Mergeability` interface.

### FileTreeView component (MergeView.tsx)

Add a `dirtyFiles: Set<string>` prop. Apply amber to file icons only (not directory text/chevrons). Use file-level status from diff headers for coloring:

1. **Amber** — file has uncommitted changes (`dirtyFiles.has(node.path)`)
2. **Green** — new file (`new file mode` in diff header)
3. **Red** — deleted file (`deleted file mode` in diff header)
4. **Muted** — modified (everything else)

Add a tooltip on amber files: "Uncommitted changes".

### Inline warning removal

Remove the inline warning text from the merge bar that previously listed all dirty files — redundant now that the file tree conveys this information visually. The blocker detail modal still shows the full list.

### Utility (diff-utils.ts)

Add `hasDirtyDescendant(dirPath, dirtyFiles)` for potential future use in directory-level propagation.

## Acceptance Criteria

- [x] Mergeability API response includes `dirty_files` array with paths of uncommitted/untracked files
- [x] Files in the file tree that appear in `dirty_files` render with amber icon color (`text-amber-500`)
- [x] Amber takes priority over other icon colors
- [x] File icon colors use file-level semantics: green=new, red=deleted, muted=modified
- [x] Directory text/chevrons remain default muted color (no amber propagation to directories)
- [x] Inline warning text removed from merge bar
- [x] Clean files retain their existing coloring (no regression)
- [x] `dirty_files` is an empty array (not null/undefined) when the worktree is clean

## Out of scope

- Uncommitted vs committed diff toggle view (filed as xpo-6211f5)
- Changing the blocker modal or warning text (already handled by xpo-fad57f)
- Amber highlight in the Commits tab
- File-level dirty indicators outside MergeView
