## Spec: Refactor `comment --json` to use shared MCP input schema

## What

Refactor the CLI `comment` command's `--json` path so that:
1. Input deserialization uses `jsonio.CommentInput` directly (drop the `inputs` shim import)
2. Output emits `jsonio.CommentOutput` JSON to stdout instead of plain text
3. Errors emit structured JSON to stdout instead of plain text

## Why

The `comment` command is the last CLI command still importing the legacy `inputs` shim.
Aligning it with the MCP tool makes CLI and MCP interchangeable for automation and
unblocks removing the `inputs` package entirely.

Additionally, in `--json` mode callers expect machine-parseable output for both success
and error cases. Currently all `--json` CLI commands emit plain-text errors — this issue
fixes `comment` and introduces a reusable `ErrorOutput` type for the others.

## How

### 1. Add `ErrorOutput` to `internal/jsonio/output.go`

```go
type ErrorOutput struct {
    Error string `json:"error"`
}
```

A minimal, generic error envelope. CLI commands emit this to stdout when `--json` is set
and an error occurs, then exit non-zero. The MCP server does not use this type (it has
its own error handling at the protocol level).

### 2. Refactor `cmd/exponential/comment.go`

**Imports:** Replace `internal/inputs` with `internal/jsonio`; add `encoding/json` and `os`.

**JSON error helper:** Add a file-local helper:

```go
func commentJSONError(msg string) {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    enc.Encode(jsonio.ErrorOutput{Error: msg})
    os.Exit(1)
}
```

**`--json` input path:** Change `inputs.CommentInput` → `jsonio.CommentInput` and
`inputs.DecodeStrict` → `jsonio.DecodeStrict`.

**Error paths (when `commentJSONFlag` is true):** Every error that would normally be
returned as a Go error or printed as plain text should instead call the JSON error
helper. This covers:
- stdin read failure
- JSON decode failure
- empty body
- issue not found
- AddComment failure

**Success path:** After `client.AddComment` succeeds, emit `jsonio.CommentOutput{ID: issueID}`
as indented JSON to stdout and return (skip the plain-text message).

### Flow (--json path)

```
stdin → jsonio.DecodeStrict → CommentInput.Body → validate non-empty
→ client.GetIssue (verify exists) → client.AddComment → CommentOutput JSON

Any error → ErrorOutput JSON + exit 1
```

### Non-goals

- Retrofitting JSON errors onto `add`/`update` — that's a separate follow-up.
- No changes to the non-JSON paths (positional arg, stdin pipe, `-` explicit stdin).

## Acceptance Criteria

- `echo '{"body":"test"}' | xpo comment xpo-123 --json` deserializes using `jsonio.CommentInput`
- Success output matches `jsonio.CommentOutput` schema: `{"id":"xpo-123"}`
- Error output matches `jsonio.ErrorOutput` schema: `{"error":"..."}`
- Errors exit non-zero
- Plain-text path unchanged
- `make test` passes
- No remaining imports of `internal/inputs` in `comment.go`
