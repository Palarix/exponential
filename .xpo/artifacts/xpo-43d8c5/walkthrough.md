## Walkthrough: Add `--json` input/output to `artifact` command

## What changed

One file: `cmd/exponential/artifact.go`.

The `artifact` command now supports `--json` mode on the parent command, accepting the same
payload shape as the MCP `artifact` tool and emitting the same output. All four operations
(add, read, delete, list) are accessible via the `operation` field in the JSON input.

## How it works

### Design: flag on the parent, not subcommands

Unlike simpler commands (merge, start) where `--json` sits on a single command, artifact
has four subcommands. The MCP tool unifies them behind a single `operation` field, so the
CLI `--json` flag lives on the parent `artifactCmd`. When `--json` is not set and no
subcommand is given, the parent falls through to `cmd.Help()` — preserving existing behavior.

### JSON path (`runArtifactJSON`)

```
stdin → jsonio.DecodeStrict → ArtifactToolInput → validate issue_id
→ client.GetIssue (resolve ID) → switch operation:
    add    → validate filename + content → client.AddArtifact → ArtifactOutput{OK, IssueID, Path}
    read   → validate filename → client.ReadArtifact → ArtifactOutput{OK, IssueID, Path, Content}
    delete → validate filename → client.DeleteArtifact → ArtifactOutput{OK, IssueID, Path}
    list   → client.ListArtifacts → jsonio.ToArtifactEntries → ArtifactOutput{OK, IssueID, Artifacts}
```

This mirrors the MCP handler (`toolset.artifact` in `internal/mcpserver/tools.go:433-497`)
exactly — same validation order, same error messages, same output shapes. The only difference
is the CLI path doesn't call `t.broadcast()` (CLI doesn't have SSE subscribers).

### Path construction

Uses `filepath.Join(storage.XpoDir(), "artifacts", issueID, filename)` for the `Path` field,
matching the MCP handler's approach.

## Key decisions

- **Parent command, not subcommands:** The MCP tool uses a single entry point with an `operation`
  field. Putting `--json` on each subcommand would have required four separate JSON handlers
  with different input types, diverging from the MCP contract. The parent-level approach keeps
  a single handler that exactly matches the MCP tool.
- **No broadcast calls:** The MCP handler calls `t.broadcast("ARTIFACT", issueID)` for add/delete
  mutations. The CLI has no SSE infrastructure, so these are omitted — consistent with how other
  CLI `--json` handlers skip broadcast.
