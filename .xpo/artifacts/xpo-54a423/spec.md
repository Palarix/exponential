# Spec: MergeView "Files changed" count ignores uncommitted changes

## What
The "Files changed" tab badge and the "All Changes" file view don't account for uncommitted changes, showing 0 and empty respectively when a worktree has only uncommitted (no committed) changes. Additionally the scope selector ("All Changes" / "Uncommitted") is buried inside the file tree sidebar and doesn't influence the badge count.

## Why
When an agent is actively working in a worktree but hasn't committed yet, the Review pane is the primary way for a user to monitor progress. Showing "0 Files changed" and an empty "All Changes" view makes it look like nothing happened.

## Design model
GitHub-PR-style scoping:
- **"All Changes"** = all committed changes on branch vs base (the PR diff). Falls back to uncommitted diffs when no commits exist yet.
- **"Uncommitted"** = working tree changes only (staged + unstaged vs HEAD).
- **Scope is first-class**: controls the badge count, file tree, and diff viewer.
- **Future extension**: generalize to a "since \<commit\>" selector for incremental review. Not in scope for this fix.

## How

### 1. Always fetch uncommitted diff eagerly
Add `fetchIssueDiff(issue.id, "uncommitted")` to the initial `Promise.all` in `refreshData`. This makes uncommitted diffs available immediately for both the fallback and the toggle.

### 2. activeFileDiffs with fallback
- `"uncommitted"` → `uncommittedFileDiffs`
- `"all"` with committed diffs → `fileDiffs`
- `"all"` with no committed diffs → `uncommittedFileDiffs` (fallback so the view isn't empty)

### 3. Dynamic tab badge
Change `count: files.length` → `count: activeFileDiffs.size`. Remove the now-unused `files` state, `fetchIssueFiles` call, and `FileInfo` import.

### 4. Elevate scope selector
Move the segmented control out of `FilesTab`'s sidebar header into a secondary toolbar just below the tab bar, spanning the full content width. This positions scope as a top-level control rather than a sidebar detail. The file tree sidebar loses the scope toggle and keeps only the search filter.

## Acceptance Criteria
- [ ] When a worktree has only uncommitted changes, "Files changed" badge shows the correct count
- [ ] When switching scope, the badge updates to reflect the active scope's file count
- [ ] When "All Changes" is selected and there are no commits, uncommitted changes are shown
- [ ] When there are commits, "All Changes" shows only committed changes
- [ ] The scope selector sits below the tab bar, visually above the file tree + diff area
- [ ] No regressions: committed-only worktrees still show correct counts and diffs
