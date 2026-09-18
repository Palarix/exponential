# Walkthrough: Align rationale --json with shared output types

## What changed

One file: `cmd/exponential/rationale.go`.

The `--json` path previously encoded the raw `exponential.RationaleSearchResult` directly.
Now it converts each `exponential.RationaleResult` to `jsonio.RationaleHit` and wraps in
`jsonio.RationaleOutput` before encoding. The JSON output is byte-identical since the
field names and types match exactly — this is a type alignment, not a schema change.

Added `jsonio` import; all other logic unchanged.
