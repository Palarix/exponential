## Walkthrough: Refactor `comment --json` to use shared MCP input schema

## What changed

Three files, all small and focused.

### `internal/jsonio/output.go`

Added `ErrorOutput` — a minimal JSON error envelope:

```go
type ErrorOutput struct {
    Error string `json:"error"`
}
```

This is the canonical type for CLI `--json` error output. The MCP server doesn't use it
(it has protocol-level errors). Follow-up issue xpo-d1a06b will retrofit this onto `add`,
`update`, and all read commands.

### `cmd/exponential/stdin.go`

Added `exitJSONError(err)` alongside the existing stdin helpers. It encodes an
`ErrorOutput` to stdout with 2-space indentation and calls `os.Exit(1)`. Placed here
(rather than in `comment.go`) because xpo-d1a06b will need it from every `--json`
command — no move required later.

### `cmd/exponential/comment.go`

The `--json` path now mirrors the MCP `comment` handler:

```
stdin → jsonio.DecodeStrict → CommentInput.Body → validate non-empty
→ client.GetIssue → client.AddComment → CommentOutput JSON
```

Specific changes:
1. **Import swap:** `internal/inputs` → `internal/jsonio` (+ `encoding/json`, `os`).
   `comment.go` was the last consumer of the `inputs` shim.
2. **Input decoding:** `inputs.CommentInput` / `inputs.DecodeStrict` →
   `jsonio.CommentInput` / `jsonio.DecodeStrict`. Same types under the hood (the shim
   was a re-export), but now imported directly.
3. **Error paths:** All five error cases in `--json` mode (stdin read, JSON decode, empty
   body, issue not found, AddComment failure) call `exitJSONError` instead of returning a
   Go error. This emits `{"error": "..."}` to stdout and exits non-zero.
4. **Success path:** Emits `jsonio.CommentOutput{ID: issueID}` as indented JSON instead
   of `fmt.Printf("Comment added to %s\n", issueID)`.
5. **Non-JSON paths:** Completely unchanged — positional arg, `-` stdin, piped stdin all
   behave identically.

## Key decisions

- **`exitJSONError` in `stdin.go`, not `comment.go`:** Avoids a file move when xpo-d1a06b
  adds JSON error output to remaining commands. The stdin helper file is already imported
  everywhere that has `--json` support.
- **`ErrorOutput` as a separate type, not fields on `CommentOutput`:** Keeps success
  output types clean for the MCP server (which never emits error fields in the structured
  output). Some output types (`SpecOutput`, `WalkthroughOutput`, `ArtifactOutput`)
  already have inline `OK`/`Error` fields — those can migrate to `ErrorOutput` in a
  future cleanup if desired, but that's out of scope here.
