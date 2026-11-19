# Add `created_at`, `created_by` and `comment_count` to `list --json`

## What

`jsonio.IssueSummary` gets three new fields:

| Field           | JSON tag                  | Source                                 |
| --------------- | ------------------------- | -------------------------------------- |
| `CreatedAt`     | `created_at` (always)     | `i.CreatedAt.Format(time.RFC3339)`     |
| `CreatedBy`     | `created_by` (always)     | `i.CreatedBy`                          |
| `CommentCount`  | `comment_count` (always)  | `len(i.Comments)`                      |

`ToIssueSummary` fills them in. `xpo list --json` and the MCP `list` tool both build their
output with `ToIssueSummary`, so both get the fields. So do the other callers of
`ToIssueSummary`: `xpo blocked --json` and the `children` of `xpo show --json`.

## Why

Agents and scripts that read the list need to know how old an issue is, who filed it, and
whether it has discussion. Today they have to call `show` on each issue to find out.

## How

- `internal/jsonio/output.go`: add the fields to `IssueSummary`, after `UpdatedAt`'s neighbours,
  and set them in `ToIssueSummary`.
- No change to the list path. `LocalTransport.ListIssues` projects the full event log with
  `ProjectIssuesWithConfig`, which appends every COMMENT event to `issue.Comments`. There is no
  event that deletes a comment, so `len(i.Comments)` is the comment count.

## Decisions

- **No `omitempty` on any of the three.** `created_at` must always be present (the issue says
  so). `created_by` matches `ShowOutput`, which always emits it. `comment_count: 0` is real
  information ("no discussion"), and a field that is sometimes missing is harder for a reader to
  rely on.

## Acceptance criteria

- [ ] `ToIssueSummary` sets `created_at` (RFC 3339), `created_by` and `comment_count`.
- [ ] An issue with no comments has `"comment_count": 0` in the JSON, not a missing field.
- [ ] The MCP `list` tool returns the three fields, and `comment_count` counts the comments
      added to the issue (test against a real projected issue, not a hand-built struct).
- [ ] `make test` passes.
