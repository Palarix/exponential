# Walkthrough: MergeView "Files changed" count ignores uncommitted changes

## What changed

The MergeView's "Files changed" tab badge, file list, and diff viewer now all respond to the active scope ("All changes" vs "Uncommitted"), and the scope selector has been promoted to the tab bar.

## Why

When a worktree had only uncommitted changes (no commits yet), the Review pane showed "Files changed 0" and an empty "All Changes" view — making it look like nothing happened. The root cause was that the badge count came from `fetchIssueFiles()`, which only returns committed files, and the "All Changes" diff view only showed committed diffs.

## How the pieces fit together

### Data flow changes (`MergeView.tsx`)

**Eager uncommitted fetch**: The initial `refreshData` call now fetches the uncommitted diff alongside the committed data (`fetchIssueDiff(issue.id, "uncommitted")` added to the `Promise.all`). Previously, uncommitted diffs were only fetched on-demand when the user toggled to "Uncommitted" scope.

**`activeFileDiffs` with fallback**: The computed value that drives the file tree and diff viewer:
- `"uncommitted"` scope → `uncommittedFileDiffs` (parsed from uncommitted diff)
- `"all"` scope with committed diffs → `fileDiffs` (parsed from committed diff)
- `"all"` scope with no committed diffs → falls back to `uncommittedFileDiffs`, so the view isn't empty on a fresh worktree

**Dynamic badge**: The "Files changed" tab badge changed from `files.length` (static committed count) to `activeFileDiffs.size` (reactive to scope). This made the `files` state (`FileInfo[]`), the `fetchIssueFiles` call, and the `FileInfo` type import all unused — they were removed.

### UI changes

**Scope selector promoted**: The "All changes / Uncommitted" segmented control moved from inside `FilesTab`'s sidebar header into the tab bar itself, absolutely centered (`left-1/2 -translate-x-1/2`). It only renders when the "Files changed" tab is active. Font size bumped from `text-xs` to `text-sm` to match the tab labels.

**Loading spinner removed**: The `RefreshCw` spinner that flashed next to the scope selector on every poll cycle was removed. Since uncommitted diffs are now eagerly loaded, the spinner added no value and was distracting.

**`FilesTab` simplified**: The component no longer receives `diffScope`, `onScopeChange`, or `uncommittedLoading` props. It receives an `isEmpty` boolean instead, computed by the parent.

## Acceptance criteria

- [x] When a worktree has only uncommitted changes, "Files changed" badge shows the correct count — verified: badge shows `activeFileDiffs.size` which reflects uncommitted files when no commits exist
- [x] When switching scope, the badge updates to reflect the active scope's file count — verified: badge derives from `activeFileDiffs` which switches on `diffScope`
- [x] When "All Changes" is selected and there are no commits, uncommitted changes are shown — verified: fallback logic in `activeFileDiffs` uses `uncommittedFileDiffs` when `fileDiffs.size === 0`
- [x] When there are commits, "All Changes" shows only committed changes — verified: `fileDiffs.size > 0` takes priority, no change to committed-only behavior
- [x] The scope selector sits in the tab bar, centered — verified: absolutely positioned with `left-1/2 -translate-x-1/2`, only visible on files tab
- [x] No regressions — all 263 frontend tests pass, lint clean, all Go tests pass
