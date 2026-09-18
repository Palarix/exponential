## Spec: Add `walkthrough` CLI command

## What

Add a `xpo walkthrough` command mirroring `xpo spec` — default action is **read with
glamour rendering**, write/delete are subcommands, `--json` provides full MCP-compatible I/O.

## Why

Same rationale as the `spec` command: having a `read` subcommand breaks with the convention
of other commands where the primary action is the default. `xpo walkthrough <ID>` should
just show the walkthrough.

## How

New file: `cmd/exponential/walkthrough.go`. Identical structure to `cmd/exponential/spec.go`,
substituting:

- `spec` → `walkthrough` in command names, descriptions, and help text
- `client.ReadSpec` → `client.ReadWalkthrough`
- `client.WriteSpec` → `client.WriteWalkthrough`
- `client.DeleteSpec` → `client.DeleteWalkthrough`
- `jsonio.SpecToolInput` → `jsonio.WalkthroughToolInput`
- `jsonio.SpecOutput` → `jsonio.WalkthroughOutput`
- `spec.md` → `walkthrough.md` in path construction

### Command structure

```
xpo walkthrough <ID>              → glamour-rendered walkthrough
xpo walkthrough <ID> --json       → WalkthroughOutput JSON (read)
xpo walkthrough --json            → full MCP JSON I/O from stdin (all operations)
xpo walkthrough write <ID>        → pipe content to stdin, writes walkthrough
xpo walkthrough delete <ID>       → deletes the walkthrough
xpo walkthrough                   → shows help
```

## Acceptance Criteria

1. `xpo walkthrough <id>` prints glamour-rendered walkthrough content
2. `xpo walkthrough <id> --json` emits `WalkthroughOutput` JSON
3. `echo '{"operation":"write",...}' | xpo walkthrough --json` writes walkthrough
4. `echo '{"operation":"read",...}' | xpo walkthrough --json` reads as JSON
5. `echo '{"operation":"delete",...}' | xpo walkthrough --json` deletes
6. `xpo walkthrough write <id>` reads from stdin and writes
7. `xpo walkthrough delete <id>` deletes the walkthrough
8. `make test` passes
