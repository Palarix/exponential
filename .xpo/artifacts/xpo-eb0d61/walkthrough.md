## Walkthrough: Fix `comment --json` to accept MCP `CommentToolInput` payload

## What changed

One file: `cmd/exponential/comment.go`.

## The bug

`comment --json` decoded stdin into `jsonio.CommentInput` (body only). The MCP tool
sends `jsonio.CommentToolInput` (id + body). Since `DecodeStrict` rejects unknown
fields, piping an MCP payload failed on the `id` field.

## The fix

### Decode target

Changed from `jsonio.CommentInput` to `jsonio.CommentToolInput`. Both have `Body`;
`CommentToolInput` additionally has `ID`.

### ID resolution

Three-tier fallback matching the pattern used by `link --json` and `merge --json`:
1. If `input.ID` is non-empty, use it (MCP parity — ID in payload)
2. Otherwise fall back to `args[0]` (CLI convenience — ID as positional arg)
3. If neither is present, emit JSON error

### Args relaxation

`Args` changed from `cobra.RangeArgs(1, 2)` to `cobra.RangeArgs(0, 2)`. In JSON mode
with ID in the payload, no positional args are needed. The non-JSON `default` branch
now explicitly checks for a missing ID before complaining about missing text.
