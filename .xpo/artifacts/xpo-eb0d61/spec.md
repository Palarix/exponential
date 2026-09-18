## Spec: Fix `comment --json` to accept MCP `CommentToolInput` payload

## What

Change `comment --json` to decode into `jsonio.CommentToolInput` (which has `id` + `body`)
instead of `jsonio.CommentInput` (body only).

## Why

`DecodeStrict` rejects unknown fields. The MCP payload includes `id`, so piping it into
the CLI fails. This breaks the "one contract" principle.

## How

Single file change: `cmd/exponential/comment.go`.

1. Change decode target from `jsonio.CommentInput` to `jsonio.CommentToolInput`
2. ID resolution: if `input.ID` is non-empty, use it; otherwise fall back to `args[0]`
3. Body comes from `input.Body` (same field name on both types)
4. Change `Args` from `cobra.RangeArgs(1, 2)` to `cobra.RangeArgs(0, 2)` — in `--json`
   mode with ID in payload, no positional args are needed
5. In the `--json` branch, if neither `input.ID` nor `args[0]` provides an ID, emit
   an error

## Acceptance Criteria

- MCP payload with `id` + `body` accepted (no positional arg needed)
- Payload with `body` only + positional arg still works
- Empty `id` in both payload and args → JSON error
- `make test` passes
