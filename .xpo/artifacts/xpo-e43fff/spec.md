# Spec: Add `--json` output to `comments` command

## What

Add a `--json` flag to `cmd/exponential/comments.go`. Refactor the command to use
`Client.GetIssue()` instead of raw `storage.ReadEvents()` + `ProjectIssues()`, then
serialize comments through `jsonio.ToCommentSummaries()`.

## Why

The current implementation bypasses the client layer and reads raw events from storage.
This is inconsistent with other commands and duplicates issue projection logic. Using
`GetIssue()` is simpler and gives us `.Comments` directly.

## Flow

1. Add `CommentsOutput` type to `jsonio/output.go`: `{Comments []CommentSummary}`.
2. Refactor `comments.go` to use `exponential.NewClient(cfg).GetIssue(id)`.
3. Add `--json` flag: when set, convert via `jsonio.ToCommentSummaries()`, wrap in
   `CommentsOutput`, encode to stdout.

## Decisions

1. **New `CommentsOutput` envelope** — MCP embeds comments inside `ShowOutput`, but CLI
   needs a standalone envelope for `comments --json`. Keeps the pattern consistent with
   `ListOutput`, `HistoryOutput`.
2. **`GetIssue` not `FindIssue`** — comments don't need children or archive fallback.
