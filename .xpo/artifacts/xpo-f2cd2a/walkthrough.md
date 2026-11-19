# Walkthrough: `created_at`, `created_by` and `comment_count` in `list --json`

## What changed

`jsonio.IssueSummary` (`internal/jsonio/output.go`) has three new fields:

- `created_by`: the actor who created the issue (`i.CreatedBy`).
- `created_at`: the creation time, formatted as RFC 3339 (`i.CreatedAt`).
- `comment_count`: the number of comments (`len(i.Comments)`).

`ToIssueSummary` sets the fields. Every caller of `ToIssueSummary` returns them:
`xpo list --json`, the MCP `list` tool, `xpo blocked --json`, and the `children` of `xpo show --json`.

## Why the list path did not change

The issue asked us to confirm that the list path loads comments. `LocalTransport.ListIssues` is the
only `ListIssues` implementation. It projects the full event log with `ProjectIssuesWithConfig`.
The projection appends every COMMENT event to `issue.Comments` (`internal/exponential/projection.go`).
No event type deletes a comment, so `len(i.Comments)` is the correct count.

If a future change adds a remote transport or a list projection without comments, `comment_count`
falls to 0. `TestListCreatedAndCommentCount` catches that, because it reads real projected issues.

## Decisions

- **No `omitempty` on the new fields.** `created_at` must always be present. `created_by` matches
  `ShowOutput`, which always emits it. `comment_count: 0` means "no discussion". A reader can
  rely on a field that is always present.
- **Field order.** `created_by` and `created_at` go before `updated_at`, as in `ShowOutput`.

## Acceptance criteria

- [x] `ToIssueSummary` sets `created_at` (RFC 3339), `created_by` and `comment_count`.
  Evidence: `TestToIssueSummaryCreatedAndCommentCount` (`internal/jsonio/jsonio_test.go`).
- [x] An issue with no comments has `"comment_count": 0` in the JSON, not a missing field.
  Evidence: `TestToIssueSummaryZeroCommentsKeepsFields` checks the marshalled JSON.
- [x] The MCP `list` tool returns the three fields, and `comment_count` counts the comments on the issue.
  Evidence: `TestListCreatedAndCommentCount` (`internal/mcpserver/tools_test.go`) adds 2 comments to one issue and none to another. It checks A=2, B=0, an RFC 3339 `created_at`, and `created_by`.
- [x] `make test` passes. Evidence: lint, frontend tests and Go tests exit 0.
