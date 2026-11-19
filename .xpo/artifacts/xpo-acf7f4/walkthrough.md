# Walkthrough: `changes` field on `show --json` events

## What was built

Each event in `show --json` and the MCP `show` tool (with `include_events`) now has a `changes` object. It lists the fields that the event changed, with the value before and after:

```json
{"type": "UPDATE", "created_by": "…", "created_at": "…",
 "changes": {
   "assignee": {"to": "A <a@x>"},
   "status": {"from": "PLANNED", "to": "DOING"}
 }}
```

## Why a replay is needed

An UPDATE payload stores only the new value of each field that the update set. Labels and dependencies hold the full new list. No stored data has the previous value. The issue description pointed at `EventsToTimelineEntries` as existing diff logic, but that function only copies the payload. So the "from" side comes from a new replay of the issue's own events.

## How the pieces fit

- `internal/jsonio/changes.go` holds `FieldChange` and `eventChanges`.
- `eventChanges` walks the events in order. The CREATE event seeds an `issueState`. Each UPDATE goes through `applyUpdate`, which compares every set field with the state, records a change when the value differs, and then applies it. A MERGE records `status` to `DONE`.
- `scalar` is a generic helper for the from/to fields. It leaves an empty value (`""` or `0`) out, so a cleared field has only `from`.
- `setDiff` gives the added and removed items for labels (by value) and dependencies (by target and kind). A reorder of the same labels is not a change.
- `ToEventSummaries` in `internal/jsonio/output.go` calls `eventChanges` once and puts each result on its `EventSummary`. Both the CLI `show` and the MCP `show` use `ToEventSummaries`, so both get the field.
- Payloads are decoded through a JSON round-trip (`decodePayload`). Events read from the log carry map payloads, and events built in code carry typed structs. The round-trip handles both.

## Decisions

1. **Keys use the `ShowOutput` field names.** `estimate` becomes `story_points`, so a reader can match a change to the issue field.
2. **`sort_order` and `children_moved` are left out.** `sort_order` is an opaque key that changes on every drag. `children_moved` records a bulk operation, not a field of this issue.
3. **CREATE has no `changes`.** There is no earlier state.
4. **Description shows the full text**, and **milestone shows IDs only** (user decisions).
5. **Assignee goes through `identity.ResolveAssignment`**, as in `ProjectIssues`. An agent that assigns itself while working for a user shows the user. Without this, `changes.assignee.to` would differ from the `assignee` that `show` reports.
6. **The replay does not reuse `ProjectIssues`.** That function projects every issue and adds derived data. The replay needs only one issue's fields, step by step.

## Non-obvious points

- If a new field is added to `UpdatePayload`, add it to `issueState` and `applyUpdate` too. Otherwise its changes do not show.
- An UPDATE before any CREATE (not possible in a valid log) gets no `changes`.

## Acceptance criteria

- [x] An UPDATE that changes the status has `changes.status` with `from` and `to`. Evidence: `TestEventChanges_StatusFromTo`, `TestEventChanges_ReplaysEarlierUpdates`.
- [x] An UPDATE that changes labels has `changes.labels` with `added` and `removed`. Evidence: `TestEventChanges_LabelsAddedRemoved`; dependencies in `TestEventChanges_DependenciesAddedRemoved`.
- [x] A field that the update set to its current value is not in `changes`. Evidence: `TestEventChanges_UnchangedFieldsLeftOut`.
- [x] An event with no shown changes has no `changes` key in the JSON. Evidence: `TestEventChanges_NoChangesKeyOmitted`.
- [x] A MERGE event has `changes.status` with `to: "DONE"`. Evidence: `TestEventChanges_MergeSetsDone`.
- [x] CREATE and COMMENT events have no `changes`. Evidence: `TestEventChanges_NoChangesKeyOmitted`, `TestShowEventsIncludeChanges`.
- [x] An agent assignee shows as the user the agent works for. Evidence: `TestEventChanges_AgentAssigneeResolvedToPrincipal`.
- [x] The MCP `show` tool returns the same `changes` field. Evidence: `TestShowEventsIncludeChanges` in `internal/mcpserver/tools_test.go`.
- [x] `make test` passes. Evidence: full run exits 0 (lint, 1777 frontend tests, Go tests).
