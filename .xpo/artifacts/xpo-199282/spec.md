# Spec: Refactor `update --json` to use shared MCP input schema

## What

Refactor the CLI `update` command's `--json` path and add `--json` output to the
status alias commands (`done`, `planned`, `blocked` with ID).

## Why

The `update --json` path uses the `inputs` shim, skips `ValidateUpdatePayload()`,
and outputs plain text instead of structured JSON. Aligning with the MCP tool makes
CLI and MCP interchangeable for automation.

The `done`, `planned`, and `blocked <id>` aliases produce plain text messages — adding
`--json` gives them structured `jsonio.UpdateOutput` output.

## Flow

### Step 1: Refactor `update --json` path

In `cmd/exponential/update.go`:
- Replace `inputs.UpdateInput` → `jsonio.UpdateInput`, `inputs.DecodeStrict` → `jsonio.DecodeStrict`, `inputs.UpdatePayloadEmpty` → `jsonio.UpdatePayloadEmpty`
- Remove `inputs` import, add `jsonio` and `encoding/json` imports
- Add `client.ValidateUpdatePayload(&payload)` after `ToUpdatePayload()` (before the empty check)
- Replace plain text output (`for _, msg := range msgs`) with `jsonio.UpdateOutput{ID: id, Messages: msgs}` JSON encoding

### Step 2: Add `--json` to `done` command

Add `doneJSONFlag` and `--json` flag. When set, emit `jsonio.UpdateOutput` JSON
instead of printing messages line by line.

### Step 3: Add `--json` to `planned` command

Same pattern as `done`.

### Step 4: Add `--json` to `blocked <id>` path

The `blocked` command already has `--json` for its list mode. When called with an ID
(status-update mode), emit `jsonio.UpdateOutput` JSON when `--json` is set.

### Note on `start`

The `start` command is excluded — it uses `client.StartWork()` (not `UpdateIssue`)
and has its own `jsonio.StartOutput` type. It will be handled in a separate issue if needed.

## Acceptance Criteria

- [ ] `echo '<MCP update payload>' | xpo update <id> --json` uses `jsonio.UpdateInput`
- [ ] `update --json` output is `jsonio.UpdateOutput` JSON
- [ ] `ValidateUpdatePayload` is called (parity with MCP path)
- [ ] `inputs` import removed from `update.go` (use `jsonio` directly)
- [ ] `xpo done <id> --json` emits `jsonio.UpdateOutput` JSON
- [ ] `xpo planned <id> --json` emits `jsonio.UpdateOutput` JSON
- [ ] `xpo blocked <id> --json` emits `jsonio.UpdateOutput` JSON
- [ ] `make test` passes
