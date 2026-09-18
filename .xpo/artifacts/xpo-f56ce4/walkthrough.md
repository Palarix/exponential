# Walkthrough: Align history --json with MCP envelope format

## What changed

Four files:

### `internal/jsonio/output.go`

- Added `TimelineEntry` type mirroring `model.TimelineEntry` with RFC3339 string timestamp.
  All fields from the model are preserved: `kind`, `timestamp`, `issue_id`, `issue_title`,
  `event_type`, `payload`, `created_by`, `on_behalf_of`, `source`, `sha`, `message`,
  `author`, `branch`.
- Changed `HistoryOutput` from `{Events []EventSummary}` to `{Timeline []TimelineEntry}`.
- Added `ToTimelineEntries()` converter from `[]model.TimelineEntry` (for CLI).
- Added `EventsToTimelineEntries()` converter from `[]model.Event` (for MCP — converts
  events to timeline entries with `kind: "issue_event"`, preserving payload and metadata).

### `internal/mcpserver/tools.go`

History handler now calls `jsonio.EventsToTimelineEntries()` instead of
`jsonio.ToEventSummaries()`. The MCP `history` tool response is richer — includes
`payload`, `on_behalf_of`, and `source` fields that were previously dropped.

### `cmd/exponential/history.go`

Replaced `ui.RenderTimelineJSON(entries, os.Stdout)` (JSONL) with building a
`jsonio.HistoryOutput` envelope and encoding as a single pretty-printed JSON object.
Flag help text updated from "JSONL" to "JSON envelope".

### `CHANGELOG.md`

Cleaned up the `[Unreleased]` section: merged duplicate `### Added` and `### Fixed`
headers, recategorized misplaced items (xpo-76e2a1 init items moved from Fixed to
Added/Changed), added entries for all JSON I/O stories completed in this session.

## Key decision

**Rich timeline shape everywhere** — the MCP `EventSummary` (3 fields) was too minimal
and unused by the web UI (which has its own richer HTTP endpoints). Adopting the full
`TimelineEntry` shape for both CLI and MCP gives agents access to payloads, delegation
info, and source metadata. This is a breaking change for the MCP `history` tool response
(field renamed from `events` to `timeline`, entry shape expanded) and for CLI consumers
of `history --json` (JSONL → single JSON envelope).
