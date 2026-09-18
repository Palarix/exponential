# Spec: Align history --json with MCP envelope format

## What

Replace JSONL output with a single JSON envelope. Use the rich `TimelineEntry` shape
(matching the CLI's existing timeline data) for both CLI and MCP surfaces.

## Flow

1. Add `TimelineEntry` type to `jsonio/output.go` — mirrors `model.TimelineEntry` with
   RFC3339 string timestamp. Fields: `kind`, `timestamp`, `issue_id`, `issue_title`,
   `event_type`, `payload`, `created_by`, `on_behalf_of`, `source`, `sha`, `message`,
   `author`, `branch`.
2. Change `HistoryOutput` from `{Events []EventSummary}` to `{Timeline []TimelineEntry}`.
3. Add `ToTimelineEntries()` converter from `[]model.TimelineEntry`.
4. Update MCP `history` handler to convert `issue.Events` → `[]TimelineEntry` (all
   `kind: "issue_event"`, no commit fields).
5. Update CLI `history.go` to wrap entries in `HistoryOutput` and encode as single JSON.
6. Update flag help text from "JSONL" to "JSON".

## Decisions

1. **Rich shape everywhere** — `EventSummary` was too minimal (no payload, no on_behalf_of).
   The web UI already uses richer shapes via HTTP. Aligning MCP to the same rich shape.
2. **`ShowOutput.Events` stays as `[]EventSummary`** — show just lists event types as
   metadata, doesn't need the full timeline. Only `HistoryOutput` changes.
