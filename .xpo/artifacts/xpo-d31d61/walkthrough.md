# Walkthrough: Add `--json` output to `inbox` command

## What changed

Two files:

### `internal/jsonio/output.go`

Added `InboxOutput` envelope (`{Items []InboxEntry}`) and `InboxEntry` type with fields
mirroring `exponential.InboxItem`: `IssueID`, `IssueTitle`, `Type` (as string),
`Payload` (interface{}), `CreatedAt` (RFC3339 string), `CreatedBy`, `OnBehalfOf`.

No converter function was added to jsonio because `InboxItem` lives in `internal/exponential`,
and importing that from jsonio would add a heavy dependency. The conversion is done inline
in the command file instead.

### `cmd/exponential/inbox.go`

Added `inboxJSONFlag` and `--json` flag. When set, converts each `exponential.InboxItem`
to `jsonio.InboxEntry` (formatting `CreatedAt` as RFC3339, casting `Type` to string),
wraps in `InboxOutput`, and encodes to stdout. Empty inbox emits `{"items": []}`.

The `--json` path runs before the "No new activity" check, consistent with other commands.
The `--clear` flag takes precedence over `--json` (checked first).
