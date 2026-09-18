## Spec: Cosmetic consistency pass for --json flags

## What

Fix five minor inconsistencies across the `--json` CLI surface.

## Items

### 1. Flag variable naming

Rename `historyJSON` → `historyJSONFlag` and `pulseJSON` → `pulseJSONFlag` to match
all other commands. Update all references in each file.

### 2. Help text normalization

Two patterns based on command category:

- **Mutation (input+output):** `"Read a structured payload as JSON from stdin"`
  Applies to: add, update, comment, link, merge
- **Read (output-only):** `"Output as JSON"`
  Applies to: list, show, comments, history, rationale, inbox, blocked, pulse, done, planned

### 3. `inbox --clear --json` success path

When `--clear` succeeds with `--json` set, emit a simple JSON object instead of plain
text. Reuse the pattern from other commands:

```go
if inboxJSONFlag {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    enc.Encode(map[string]bool{"cleared": true})
    return nil
}
```

### 4. `update --json` empty-payload error message

Change `"no changes in payload"` to `"no fields set: provide at least one field to update"`
to match the MCP handler.

### 5. `list --json` status filter validation

When `listJSONFlag` is true, validate each status filter value before calling
`client.ListIssues`. Use `jsonio.ValidateStatus` for each entry.

## Acceptance Criteria

- All `--json` flag variables use `xxxJSONFlag` naming
- All `--json` help text follows consistent patterns
- `inbox --clear --json` emits JSON
- `update --json` empty-payload error matches MCP
- `list --json` rejects invalid status filters with JSON error
- `make test` passes
