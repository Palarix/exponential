## Spec: Remove `internal/inputs` backward-compat shim

## What

Delete `internal/inputs/` and migrate its remaining consumers and test coverage.

## Why

The shim has zero production consumers since the JSON I/O epic migrated all CLI
commands to import `jsonio` directly. Only `internal/mcpserver/tools_test.go` still
references it. Keeping it adds confusion about which package to import.

## How

Three steps, all mechanical:

### 1. Update `internal/mcpserver/tools_test.go`

Replace `"github.com/palarix/exponential/internal/inputs"` import with
`"github.com/palarix/exponential/internal/jsonio"`. Change all `inputs.XxxInput` →
`jsonio.XxxInput` and `inputs.DecodeStrict` → `jsonio.DecodeStrict`.

### 2. Migrate tests from `internal/inputs/inputs_test.go` to `internal/jsonio/`

Create `internal/jsonio/jsonio_test.go`. Move all tests, changing the package
declaration from `inputs` to `jsonio`. The test functions exercise:
- `DecodeStrict` (valid, unknown fields, malformed)
- `AddInput.ToCreatePayload` (valid, missing title, with status, with links)
- `UpdateInput.ToUpdatePayload` (with status validation)
- `ValidateStatus` (valid, invalid)
- `UpdatePayloadEmpty` (empty, non-empty)
- `CommentInput` decoding
- Markdown round-tripping in descriptions

Since these tests call the functions directly (not through the shim), they work
unchanged after the package rename.

### 3. Delete `internal/inputs/`

Remove `internal/inputs/inputs.go` and `internal/inputs/inputs_test.go`.

## Acceptance Criteria

- No Go file imports `internal/inputs`
- `internal/inputs/` directory removed
- `internal/jsonio/` has test files with equivalent coverage
- `make test` passes
