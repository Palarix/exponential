## Walkthrough: Add `--json` input/output to `start` command

## What changed

One file: `cmd/exponential/update.go`.

The `start` command now supports `--json` mode, accepting the same payload shape as the
MCP `start` tool and emitting the same output.

## How it works

### JSON path (`runStartJSON`)

```
stdin → jsonio.DecodeStrict → StartToolInput → validate id
→ handle mode override → client.StartWork(id, force)
→ StartOutput JSON to stdout
```

The JSON path mirrors the MCP handler in `internal/mcpserver/tools.go` (`toolset.start`):
same input type, same validation, same mode-override switch, same output struct.

### Args change

`cobra.ExactArgs(1)` → `cobra.MaximumNArgs(1)`. In JSON mode the ID comes from stdin,
so no positional arg is needed. The interactive path checks `len(args) == 0` explicitly
and exits with an error message matching cobra's default, preserving the existing UX.

### Error handling

All errors in the JSON path go through `exitJSONError`, which emits
`{"error": "..."}` to stdout and exits with code 1 — the same contract as every
other `--json` command.

## Key decisions

- No new types or helpers were needed. `jsonio.StartToolInput`, `jsonio.StartOutput`,
  `readStdinExplicit`, `exitJSONError`, and `jsonio.DecodeStrict` all existed already.
- The mode-override switch is duplicated between the interactive and JSON paths (matching
  how merge duplicates its strategy switch). This keeps each path self-contained and
  readable rather than extracting a shared helper for three lines of code.
