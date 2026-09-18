## Walkthrough: Remove `internal/inputs` backward-compat shim

## What changed

The `internal/inputs` package — a backward-compatibility shim of type aliases and
function wrappers over `internal/jsonio` — has been deleted. Its test coverage was
migrated to `internal/jsonio/`, which previously had no tests.

## Details

### `internal/mcpserver/tools_test.go`

Replaced `internal/inputs` import with `internal/jsonio`. All ~35 references to
`inputs.AddInput`, `inputs.UpdateInput`, `inputs.LinkInput` changed to their `jsonio`
equivalents. Since the shim was just type aliases, the types are identical — no
behavioral change.

### `internal/jsonio/jsonio_test.go` (new)

All 17 tests from `internal/inputs/inputs_test.go` migrated here with the package
declaration changed from `inputs` to `jsonio`. The tests call the functions directly
(not through wrappers), so they work unchanged. Coverage includes:

- `DecodeStrict` — unknown field rejection
- `AddInput.ToCreatePayload` — full field mapping, missing title, invalid status, invalid/missing link fields
- `UpdateInput.ToUpdatePayload` — partial updates, empty payload, invalid status
- `ValidateStatus` — all valid statuses + invalid values
- `UpdatePayloadEmpty` — zero payload, status, labels, dependencies
- `CommentInput` — body decoding with newlines
- Markdown round-trip — special characters, code blocks, quotes
- `NormalizeDependencyKind` — all forms including camelCase, UPPER_CASE

### `internal/inputs/` (deleted)

Both `inputs.go` (30 lines — type aliases + function wrappers) and `inputs_test.go`
(244 lines) removed. No production code imported this package; it was kept alive only
by the MCP server test file.
