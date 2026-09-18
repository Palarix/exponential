## Walkthrough: Add `spec` CLI command

## What changed

New file: `cmd/exponential/spec.go`.

A new top-level `xpo spec` command where the default action is **read with glamour
rendering** — `xpo spec <ID>` just shows the spec, no `read` subcommand needed.

## How it works

### Command structure

```
xpo spec <ID>              → glamour-rendered spec to terminal (default action)
xpo spec <ID> --json       → SpecOutput JSON (read)
xpo spec --json            → full MCP JSON I/O from stdin (write/read/delete)
xpo spec write <ID>        → pipe content to stdin, writes spec
xpo spec delete <ID>       → deletes the spec
xpo spec                   → shows help
```

The parent command (`specCmd`) uses `cobra.MaximumNArgs(1)` and its `RunE` dispatches
three ways based on `--json` and whether a positional arg is present:

1. `--json` + no arg → `runSpecJSON()` (full stdin dispatch)
2. `--json` + arg → read spec, emit `SpecOutput` JSON
3. no `--json` + arg → read spec, glamour render, print
4. no `--json` + no arg → `cmd.Help()`

### Glamour rendering

Uses the same pattern as `internal/ui/details.go:115-124`:

```go
renderer, _ := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),
    glamour.WithWordWrap(min(termWidth-4, 76)),
)
rendered, _ := renderer.Render(content)
```

Falls back to raw content if glamour fails.

### JSON dispatch (`runSpecJSON`)

Mirrors the MCP handler (`toolset.spec` in `tools.go:341-385`):

```
stdin → jsonio.DecodeStrict → SpecToolInput → validate issue_id
→ switch operation:
    write  → validate content → client.WriteSpec → SpecOutput{OK, IssueID, Path}
    read   → client.ReadSpec → SpecOutput{OK, IssueID, Path, Content}
    delete → client.DeleteSpec → SpecOutput{OK, IssueID, Path}
```

### Write subcommand

Requires piped stdin (`isStdinPiped()` check). Calls `client.WriteSpec()` directly.
No `--file` flag — keeps it simple since the MCP/JSON path handles structured input.

## Key decisions

- **No `read` subcommand:** The user explicitly requested `xpo spec <ID>` as the default
  read action, matching the ergonomics of `xpo show <ID>`. This breaks from the original
  issue description which had `xpo spec read <ID>`.
- **Glamour for terminal rendering:** Reuses the existing `glamour` dependency already
  used in `internal/ui/details.go` for issue descriptions. Auto-detects light/dark theme
  and wraps at `min(termWidth-4, 76)`.
- **`--json` on parent, not subcommands:** Same pattern as the `artifact` command — the
  MCP tool uses a single entry point with an `operation` field, so the CLI mirrors that
  with `--json` on the parent command for full MCP-compatible I/O.
