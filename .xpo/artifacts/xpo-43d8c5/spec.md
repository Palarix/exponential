## Spec: Add `--json` input/output to `artifact` command

## What

Add a `--json` flag to the parent `artifact` CLI command that reads `jsonio.ArtifactToolInput`
from stdin and dispatches all four operations (add, read, delete, list) via the `operation`
field, emitting `jsonio.ArtifactOutput` on success / `jsonio.ErrorOutput` on failure.

## Why

The artifact command is the last multi-operation command without `--json` support. The MCP
`artifact` tool already uses a single entry point with an `operation` field — the CLI should
expose the same contract.

## How

Single file change: `cmd/exponential/artifact.go`.

### Approach

Add `--json` to the parent `artifactCmd` (not the subcommands). Give the parent a `RunE` that:
- If `--json` is set: read stdin, decode `ArtifactToolInput`, dispatch by `operation`
- If `--json` is not set: print help (preserving current behavior for bare `xpo artifact`)

### Flow (--json path)

```
stdin → jsonio.DecodeStrict → ArtifactToolInput → validate issue_id + operation
→ switch operation:
    add    → validate filename + content → client.AddArtifact → ArtifactOutput{OK, IssueID, Path}
    read   → validate filename → client.ReadArtifact → ArtifactOutput{OK, IssueID, Path, Content}
    delete → validate filename → client.DeleteArtifact → ArtifactOutput{OK, IssueID, Path}
    list   → client.ListArtifacts → jsonio.ToArtifactEntries → ArtifactOutput{OK, IssueID, Artifacts}
```

This mirrors the MCP handler (`toolset.artifact` in `tools.go:433-497`) exactly, minus the
broadcast calls (CLI doesn't need them).

### Path construction

The MCP handler uses `filepath.Join(storage.XpoDir(), "artifacts", issueID, filename)` for
the `Path` field. The CLI JSON handler should do the same. Import `path/filepath` and
`storage` package.

### Changes

1. Add `var artifactJSONFlag bool`
2. Add `artifactCmd.Flags().BoolVar(&artifactJSONFlag, "json", false, "Read a structured payload as JSON from stdin")` in `init()`
3. Set `artifactCmd.RunE` to handle the `--json` dispatch (show help otherwise)
4. Add `runArtifactJSON()` function mirroring the MCP handler's switch

### New imports needed

- `encoding/json` (for `json.NewEncoder`)
- `path/filepath` (for path construction)
- `github.com/palarix/exponential/internal/jsonio`
- `github.com/palarix/exponential/internal/storage`

## Acceptance Criteria

1. `echo '{"operation":"add","issue_id":"xpo-123","filename":"notes.md","content":"..."}' | xpo artifact --json` works
2. `echo '{"operation":"read","issue_id":"xpo-123","filename":"notes.md"}' | xpo artifact --json` works
3. `echo '{"operation":"delete","issue_id":"xpo-123","filename":"notes.md"}' | xpo artifact --json` works
4. `echo '{"operation":"list","issue_id":"xpo-123"}' | xpo artifact --json` works
5. Output matches MCP `ArtifactOutput` schema
6. Errors emit `jsonio.ErrorOutput` via `exitJSONError`
7. Bare `xpo artifact` (no `--json`, no subcommand) still shows help
8. `make test` passes
