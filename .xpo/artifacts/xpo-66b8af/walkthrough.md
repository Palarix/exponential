## Walkthrough: Cosmetic consistency pass for --json flags

## What changed

14 files touched across 5 consistency items. All mechanical, no behavioral changes
except items 3 and 5.

### 1. Flag variable naming (history.go, pulse.go)

Renamed `historyJSON` → `historyJSONFlag` and `pulseJSON` → `pulseJSONFlag`. All other
commands already used the `xxxJSONFlag` convention.

### 2. Help text normalization (12 files)

Two standard patterns applied:
- **Mutation (input+output):** `"Read a structured payload as JSON from stdin"` — add, update, comment, link, merge
- **Read (output-only):** `"Output as JSON"` — list, show, comments, history, rationale, inbox, blocked, pulse, done, planned

Previously varied between "Read a full issue payload...", "Output as JSON matching MCP ... schema", "Output blocked issues as JSON", "output raw metrics as JSON" (lowercase), etc.

### 3. inbox --clear --json success path (inbox.go)

When `--clear` succeeds with `--json` set, now emits `{"cleared": true}` as JSON
instead of the plain-text `"Inbox marked as read."`. Previously the `--json` flag was
silently ignored for the `--clear` success path.

### 4. update --json empty-payload error (update.go)

Changed from `"no changes in payload"` to `"no fields set: provide at least one field to update"` to match the MCP handler's wording. Updated the corresponding integration
test assertion.

### 5. list --json status filter validation (list.go)

Added upfront validation of status filter values via `jsonio.ValidateStatus` when
`listJSONFlag` is true. Previously invalid statuses (e.g. `--status TODO`) silently
returned empty results. Now emits a JSON error identifying the invalid value.
