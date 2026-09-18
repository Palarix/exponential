# Walkthrough: Add `--json` output to `comments` command

## What changed

Two files:

### `internal/jsonio/output.go`

Added `CommentsOutput` envelope type: `{Comments []CommentSummary}`. Follows the same
pattern as `ListOutput`, `HistoryOutput` — a top-level wrapper for a list of typed entries.

### `cmd/exponential/comments.go`

**Refactored issue lookup**: replaced `storage.ReadEvents()` + `exponential.ProjectIssues()`
+ map lookup with a single `exponential.NewClient(cfg).GetIssue(args[0])` call. This
eliminates the direct storage dependency and uses the same client path as other commands.
The `storage` import was removed entirely.

**Added `--json` flag**: when set, converts `issue.Comments` via `jsonio.ToCommentSummaries()`,
wraps in `jsonio.CommentsOutput`, and encodes to stdout. Empty comments produce
`{"comments": []}` (explicit empty array, not null) for consistent machine-readable output.

The terminal rendering path is unchanged — same lipgloss styling and formatting.
