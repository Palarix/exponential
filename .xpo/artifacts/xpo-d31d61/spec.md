# Spec: Add `--json` output to `inbox` command

## What

Add a `--json` flag to `cmd/exponential/inbox.go`. Define an `InboxOutput` envelope
type in `jsonio` with an `InboxEntry` item type that mirrors `exponential.InboxItem`
with RFC3339 string timestamps (consistent with all other jsonio types).

## Flow

1. Add `InboxOutput` and `InboxEntry` types to `jsonio/output.go`.
2. Add `ToInboxEntries()` converter function.
3. Add `--json` flag to inbox command: when set, convert items and encode to stdout.

## Decisions

1. **New `InboxEntry` type** — `exponential.InboxItem` uses `time.Time` for `CreatedAt`.
   Following the pattern of all other jsonio types, we format as RFC3339 string.
2. **`Payload` stays `interface{}`** — the payload varies by event type. JSON marshaling
   handles this naturally.
