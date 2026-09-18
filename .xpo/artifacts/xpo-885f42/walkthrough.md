## Walkthrough: Add --json input/output to link command

## What changed

One file: `cmd/exponential/link.go`.

The `link` command now supports `--json` mode, accepting the same payload shape as the
MCP `link` tool and emitting the same output.

## How it works

### JSON path

```
stdin → jsonio.DecodeStrict → LinkToolInput → validate source/target/type
→ client.GetIssue (both) → self-link check → duplicate check
→ client.UpdateIssue → LinkOutput JSON
```

The JSON path mirrors the MCP handler in `tools.go` exactly, including two guards the
CLI previously lacked:
- **Self-link check:** rejects `source == target`
- **Duplicate-link check:** rejects if the same `(target, kind)` pair already exists

All errors use `exitJSONError` (from xpo-d1a06b).

### Arg handling

`Args` changed from `cobra.ExactArgs(2)` to `cobra.RangeArgs(0, 2)` because in `--json`
mode, no positional args are needed. The non-JSON branch validates `len(args) == 2`
manually and shows help if wrong.

`MarkFlagRequired("type")` was removed because `--type` is not needed in JSON mode (the
type field is in the payload). The non-JSON branch still validates that the type
normalizes to a known dependency kind.

### Non-JSON path

Unchanged behavior — positional args + `--type` flag, plain-text output.
