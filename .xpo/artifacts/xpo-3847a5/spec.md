# Spec: Refactor `add --json` to use shared MCP input schema

## What

Refactor the CLI `add` command's `--json` path so that:
1. Input deserialization uses `jsonio.AddInput` directly (drop the `inputs` shim import)
2. Validation calls `client.ValidateCreatePayload()` — the same path the MCP tool uses
3. Output emits `jsonio.AddOutput` JSON to stdout instead of plain text

## Why

The `--json` path already accepts the correct schema (via the `inputs.AddInput` alias),
but diverges from the MCP tool in two ways: it skips `ValidateCreatePayload()` (missing
title length, description size, assignee format, parent/dependency resolution, priority
range, and cycle validation), and it outputs human-readable text instead of structured
JSON. Aligning both makes CLI and MCP interchangeable for automation.

## Flow

### Step 1: Switch import from `inputs` to `jsonio`

In `cmd/exponential/add.go`, replace:
- `inputs.AddInput` → `jsonio.AddInput`
- `inputs.DecodeStrict()` → `jsonio.DecodeStrict()`
- Remove the `inputs` import, add `jsonio` import

### Step 2: Add `ValidateCreatePayload` call

After `input.ToCreatePayload()`, call `client.ValidateCreatePayload(&payload)`.
This subsumes the existing manual estimate validation (`config.ValidateEstimate`),
so remove that redundant block.

### Step 3: Emit JSON output

Replace `showConfirmation(issue)` with `jsonio.AddOutput` construction and
`json.NewEncoder(os.Stdout).SetIndent("", "  ").Encode(out)`, matching the
pattern used by `show --json`, `list --json`, etc.

The `AddOutput` struct is: `{ID, Title, Status}`.

### Step 4: Duplicate check stays

The `--force` / duplicate-check logic is CLI-specific and remains unchanged.
When `--json` mode hits a duplicate, the warning is still printed as text and
exits 0 — this is intentional: structured callers should use `--force` to skip it.

## Acceptance Criteria

- [ ] `echo '<MCP add payload>' | xpo add --json` works with `jsonio.AddInput` schema
- [ ] Output is `jsonio.AddOutput` JSON (not plain text)
- [ ] `ValidateCreatePayload` is called (parity with MCP path)
- [ ] Redundant manual estimate validation removed
- [ ] `inputs` import removed from `add.go` (use `jsonio` directly)
- [ ] `make test` passes
