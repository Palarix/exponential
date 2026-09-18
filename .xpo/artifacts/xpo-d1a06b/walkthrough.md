## Walkthrough: Standardize JSON error output across all CLI --json commands

## What changed

Every CLI command with a `--json` flag now emits `jsonio.ErrorOutput` JSON to stdout on
failure instead of plain text, and exits non-zero. This covers errors at three levels:
cobra arg validation, `PersistentPreRunE`, and command handlers.

## Infrastructure

### `internal/jsonio/output.go`

`ErrorOutput` was introduced by xpo-aa4529. No changes here — just consumed.

### `cmd/exponential/stdin.go`

`exitJSONError(err)` was introduced by xpo-aa4529. This issue extracted the writing
logic into `writeJSONError(w io.Writer, err error)` so the JSON encoding can be unit
tested without calling `os.Exit`.

### `cmd/exponential/main.go`

The hardest part of this issue. Errors from cobra (wrong arg count, unknown flags) and
`PersistentPreRunE` (no project, version mismatch) happen before any command handler
runs, so per-command flag checks can't catch them.

Solution:

1. `argsContainJSONFlag()` scans `os.Args` for `--json` or `--json=true`, stopping at
   `--`. This is a raw-arg scan because cobra may not have parsed flags when it fails.

2. In `main()`, if JSON mode is detected, `SilenceErrors` and `SilenceUsage` are set on
   the root command *before* `Execute()`. This prevents cobra from printing plain-text
   errors or usage — only done conditionally so non-JSON paths are unchanged.

3. After `Execute()` returns an error, `main()` calls `exitJSONError(err)` if JSON mode
   is active.

4. The "not a project" check in `PersistentPreRunE` previously called `printNotAProject()`
   + `os.Exit(0)` — a plain-text banner that couldn't be caught by `main()`. Now it
   returns a descriptive error when JSON mode is active, which flows through `main()`'s
   centralized handler.

Design decision: `argsContainJSONFlag` does not replicate pflag's full boolean parsing
(`--json=1`, `--json=TRUE`, last-value-wins for repeated flags). These forms are
technically valid but not used in practice by anyone piping `--json` output.

## Per-command changes

All follow the same pattern — wrap each error path with a JSON flag check:

```go
if err != nil {
    if xxxJSONFlag {
        exitJSONError(err)
    }
    // existing plain-text error handling unchanged
}
```

### Mutation commands (input + output)

**`add.go`** — 6 error paths. The duplicate-check warning is notable: in JSON mode it
emits an `ErrorOutput` with the duplicate IDs and exits non-zero, instead of styled
terminal text + exit 0. A duplicate that exits 0 with unparseable output is a caller bug.

**`update.go`** — 6 error paths in the `--json` block. The "no changes in payload" case
now emits an error in JSON mode (empty update = caller mistake). The `done` and `planned`
subcommands had their JSON flag check moved before the error check — previously the error
fired first and the flag was never consulted.

### Read commands (output only)

**`list.go`**, **`show.go`**, **`comments.go`**, **`rationale.go`**, **`pulse.go`** — 1
error path each. All had the same structural issue: the error fired before the `--json`
flag check. Fixed by adding a flag check at each error site.

**`history.go`** — 3 error paths (FindIssue, ListIssues, parseDuration). Also fixed
a mixed-output bug: the `"Note: This issue is archived."` plain-text line was printed
before the `--json` check, contaminating stdout when JSON mode was active. Now suppressed
when `historyJSON` is true.

**`inbox.go`** — 2 error paths (SetInboxLastRead, GetInbox).

**`blocked.go`** — 2 error paths (list mode ListIssues, mark mode UpdateIssue).

## Test strategy

34 tests in `json_error_test.go`, split into three tiers:

**Unit tests (6):** `argsContainJSONFlag` edge cases (bare flag, `=true`, `=false`, no
flag, after `--`, mixed flags). `writeJSONError` format verification (valid JSON, wrapped
errors, indentation, no extra fields).

**Integration tests — valid project (20):** Build the real CLI binary in `TestMain`, create
a minimal `.xpo` project per test, and run commands with inputs designed to trigger errors.
Covers: bad JSON decode, missing required fields, empty payloads, not-found issue IDs,
invalid `--since` values, empty comment bodies, cobra arg validation (missing args),
no-project errors, `--json=true` assignment syntax, and duplicate detection.

**Integration tests — broken storage (5):** Create a project where `issues.db` is a
directory (causes I/O errors). Tests: `list`, `pulse`, `inbox`, `blocked` (list mode),
`history` (global mode).

Every integration test verifies: non-zero exit code, stdout is valid
`jsonio.ErrorOutput` JSON, and (where applicable) the error message contains the
expected substring.
