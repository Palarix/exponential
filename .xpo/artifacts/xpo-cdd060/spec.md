## Spec: Add `--json` input/output to `start` command

## What

Add a `--json` flag to the `start` CLI command that reads `jsonio.StartToolInput` from
stdin and emits `jsonio.StartOutput` on success / `jsonio.ErrorOutput` on failure.

## Why

The `start` command is one of the remaining commands without `--json` support, breaking
the unified JSON I/O contract that lets scripts and agents use the CLI with the same
payload shapes as the MCP tools.

## How

Single file change: `cmd/exponential/update.go`.

### Flow (--json path)

```
stdin → jsonio.DecodeStrict → StartToolInput → validate id
→ handle mode override → client.StartWork(id, force)
→ StartOutput JSON
```

### Changes

1. Add `var startJSONFlag bool` alongside existing `startForce` / `startMode` vars.
2. Register the flag: `startCmd.Flags().BoolVar(&startJSONFlag, "json", false, "Read a structured payload as JSON from stdin")` in `init()`.
3. In the `startCmd.Run` function, add early dispatch: `if startJSONFlag { runStartJSON(); return }`.
4. When `--json` is set, change `Args` validation from `cobra.ExactArgs(1)` to allow zero args (the ID comes from stdin JSON). Use `cobra.MaximumNArgs(1)` so both `xpo start --json` (stdin) and `xpo start xpo-123 --json` (positional, for output-only) work, but the JSON path ignores positional args.
5. Write `runStartJSON()` following the merge command pattern:
   - `readStdinExplicit()` → `jsonio.DecodeStrict` into `StartToolInput`
   - Validate `input.ID != ""` → `exitJSONError`
   - Handle `input.Mode` (same switch as the interactive path and MCP handler)
   - Call `client.StartWork(input.ID, input.Force)`
   - Encode `jsonio.StartOutput{ID, Branch, WorktreePath, Messages}` to stdout

### What stays the same

- Interactive (non-JSON) path is untouched.
- `jsonio.StartToolInput` and `jsonio.StartOutput` already exist — no type changes.
- `readStdinExplicit`, `exitJSONError`, `jsonio.DecodeStrict` are reused as-is.

## Acceptance Criteria

1. `echo '{"id":"xpo-123"}' | xpo start --json` reads input and emits `StartOutput` JSON.
2. `echo '{"id":"xpo-123","force":true}' | xpo start --json` honours the force flag.
3. `echo '{"id":"xpo-123","mode":"branch"}' | xpo start --json` honours the mode override.
4. Errors emit `jsonio.ErrorOutput` via `exitJSONError`.
5. `make test` passes.
