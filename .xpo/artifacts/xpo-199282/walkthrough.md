# Walkthrough: Refactor `update --json` to use shared MCP input schema

## What changed

Two files: `cmd/exponential/update.go` and `cmd/exponential/blocked.go`.

The `update --json` path now mirrors the MCP `update` handler:

```
stdin → jsonio.DecodeStrict → UpdateInput.ToUpdatePayload() → ValidateUpdatePayload → UpdateIssue → UpdateOutput JSON
```

The status alias commands (`done`, `planned`, `blocked <id>`) gained `--json` flags
that emit `jsonio.UpdateOutput` instead of plain text messages.

## Files

### `cmd/exponential/update.go`

- **Import swap**: `inputs` → `jsonio` (drops backward-compat shim)
- **Validation**: Added `client.ValidateUpdatePayload(&payload)` after `ToUpdatePayload()`
  and the empty-payload check. This adds title length, description size, assignee format,
  parent/dep resolution, priority range, estimate, and cycle validation — matching the
  MCP path.
- **JSON output**: Replaced `for _, msg := range msgs { fmt.Println(msg) }` with
  `jsonio.UpdateOutput{ID, Messages}` JSON encoding.
- **`done` command**: Added `--json` flag. When set, emits `UpdateOutput` JSON.
- **`planned` command**: Added `--json` flag. Same pattern.

### `cmd/exponential/blocked.go`

The `--json` flag already existed for the list mode (no args). Now it also applies to
the status-update mode (with ID arg), emitting `jsonio.UpdateOutput` JSON.

## Key decisions

**`start` excluded.** The `start` command uses `client.StartWork()` — not `UpdateIssue` —
and has its own `jsonio.StartOutput` type with branch/worktree fields. It's a meta-command
that happens to update status as part of a larger chain, not a status alias.

**Validation ordering.** `ValidateUpdatePayload` runs after the empty-payload check.
This matches the MCP handler's ordering — no point validating a payload we're going
to reject as empty anyway.