## Walkthrough: Add `walkthrough` CLI command

## What changed

New file: `cmd/exponential/walkthrough.go`.

A new top-level `xpo walkthrough` command mirroring the `xpo spec` command (xpo-d9d6f2)
— default action is read with glamour rendering, write/delete are subcommands, `--json`
provides full MCP-compatible I/O.

## How it works

Identical structure to `cmd/exponential/spec.go`, substituting walkthrough-specific types
and client methods throughout:

- `client.ReadWalkthrough` / `WriteWalkthrough` / `DeleteWalkthrough`
- `jsonio.WalkthroughToolInput` / `jsonio.WalkthroughOutput`
- `walkthrough.md` in path construction

### Command structure

```
xpo walkthrough <ID>              → glamour-rendered walkthrough (default)
xpo walkthrough <ID> --json       → WalkthroughOutput JSON (read)
xpo walkthrough --json            → full MCP JSON I/O from stdin
xpo walkthrough write <ID>        → pipe content to stdin
xpo walkthrough delete <ID>       → deletes the walkthrough
xpo walkthrough                   → shows help
```

## Key decisions

- **Exact mirror of `spec` command:** Both commands share the same UX pattern — this was
  a deliberate user decision to keep the CLI consistent. The user explicitly asked for
  `xpo walkthrough <ID>` as the default read action (no `read` subcommand), matching `xpo spec`.
