## Walkthrough: Resolve all CLI JSON input IDs to canonical issue IDs

## What changed

Three files:
- `cmd/exponential/update.go` — `update --json`, `start --json`, `done --json`, `planned --json`
- `cmd/exponential/comment.go` — `comment --json`
- `cmd/exponential/blocked.go` — `blocked [id] --json`

All six CLI `--json` paths now resolve the caller-supplied issue ID to the canonical
`issue.ID` via `client.GetIssue()` before performing operations and emitting output.

## How it works

### The pattern

Each affected path was updated to follow the same pattern the MCP handlers use:

```go
issue, err := client.GetIssue(rawID)
if err != nil {
    exitJSONError(err)
}
// Use issue.ID for all subsequent operations and output
```

This ensures short hashes, prefixless IDs, and partial matches all resolve to the
canonical full ID in the JSON output.

### Per-command changes

**`update --json`**: Added `client.GetIssue(id)` after payload validation but before
`UpdateIssue`. Uses `issue.ID` in both the `UpdateIssue` call and `UpdateOutput`.

**`start --json`**: Added `client.GetIssue(input.ID)` before `StartWork`. Uses `issue.ID`
in both the `StartWork` call and `StartOutput`. Note: `StartWork` internally resolves
the ID again, but passing the canonical form is consistent and avoids the output mismatch.

**`done --json` / `planned --json`**: Restructured to branch on the JSON flag early,
resolving the ID first in the JSON path. The interactive path is untouched and continues
to pass the raw arg (internal resolution handles it).

**`comment --json`**: Was already calling `client.GetIssue(issueID)` but discarding the
result (`_`). Changed to capture `issue` and use `issue.ID` in `AddComment` and
`CommentOutput`.

**`blocked [id] --json`**: Same restructure as done/planned — JSON path branches early
with a resolve step.

## Key decisions

- **Resolve before operate, not just in output**: While the bug was primarily about
  the output containing non-canonical IDs, I also changed the operation calls to use
  `issue.ID` for consistency. The internal functions resolve again, but passing canonical
  IDs is cleaner and matches the MCP handlers exactly.
- **Restructured done/planned/blocked JSON paths**: Rather than adding a resolve step
  in the middle of shared code, I separated the JSON and interactive paths completely.
  This avoids the mixed error-handling pattern (checking `if jsonFlag` before every error)
  and makes each path self-contained.
