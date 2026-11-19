# Add a `changes` field to the events in `show --json`

## What

Each event in `show --json` (and the MCP `show` tool) gets a `changes` object. It lists only the fields that the event changed, with the value before and after.

```json
{"type": "UPDATE", "created_by": "…", "created_at": "…",
 "changes": {
   "status": {"from": "DOING", "to": "DONE"},
   "labels": {"added": ["bug"], "removed": ["feature"]}
 }}
```

## Why

Today an event has only `type`, `created_by` and `created_at`. A reader sees that the issue changed, but not what changed. To find out, the reader must call `history` and replay the payloads.

## Correction to the issue description

The description says `EventsToTimelineEntries` already works out the differences. It does not. It copies the raw payload. An UPDATE payload stores only the new value of each field that the update set. The labels field holds the full new list. The previous values exist only as the state after the earlier events. So the "from" side needs a replay of the issue's own events.

## How

1. Add `Changes map[string]FieldChange` with `json:"changes,omitempty"` to `EventSummary`.
2. Add `FieldChange` with `From`, `To` (`any`, `omitempty`) and `Added`, `Removed` (`any`, `omitempty`).
3. In `ToEventSummaries`, replay the events in order. Keep a running state, seeded by the CREATE event.
4. For each UPDATE, compare each set payload field with the running state. Record a change only when the value differs. Then apply the field.
5. For each MERGE, record `status` from the running status to `DONE` when they differ.
6. CREATE, COMMENT, ARTIFACT and DELETE events get no `changes`.

### Fields and keys

The keys use the field names of `ShowOutput`.

| Payload field  | Key            | Shape            |
| -------------- | -------------- | ---------------- |
| `title`        | `title`        | from / to        |
| `description`  | `description`  | from / to (full text) |
| `status`       | `status`       | from / to        |
| `parent_id`    | `parent_id`    | from / to        |
| `estimate`     | `story_points` | from / to        |
| `priority`     | `priority`     | from / to        |
| `assignee`     | `assignee`     | from / to        |
| `cycle_id`     | `cycle_id`     | from / to        |
| `milestone_id` | `milestone_id` | from / to (IDs)  |
| `labels`       | `labels`       | added / removed  |
| `dependencies` | `dependencies` | added / removed  |

An empty value (`""` or `0`) is left out of `from` or `to`. So "cleared the assignee" is `{"assignee": {"from": "A <a@x>"}}`.

Dependency entries use the `model.Dependency` JSON shape, the same shape as `dependencies` in `ShowOutput`.

## Decisions

1. **`sort_order` is left out.** It is an opaque fractional key, and a reorder changes it on every drag. It is noise for a reader.
2. **`children_moved` is left out.** It is a record of a bulk operation, not a field of this issue.
3. **CREATE has no `changes`.** There is no state before it. The issue's own fields show the current state.
4. **The replay lives in `jsonio`.** It reads only the issue's own events. It does not reuse `ProjectIssues`, because that function projects all issues and adds derived data.
5. **Description shows the full text** in `from` and `to`, the same as other scalar fields (user decision).
6. **Milestone shows IDs only**, as stored in the payload. No milestone lookup (user decision).
7. **Assignee values go through `identity.ResolveAssignment`**, the same as in the projection. An agent that assigns itself for a user shows the user. So `from` and `to` match the `assignee` that `show` reports.
8. **Labels and dependencies compare as sets.** A reorder of the same labels is not a change. Dependencies compare by target and kind.

## Acceptance criteria

- [ ] An UPDATE that changes the status has `changes.status` with `from` and `to`.
- [ ] An UPDATE that changes labels has `changes.labels` with `added` and `removed`.
- [ ] A field that the update set to its current value is not in `changes`.
- [ ] An event with no shown changes has no `changes` key in the JSON.
- [ ] A MERGE event has `changes.status` with `to: "DONE"`.
- [ ] CREATE and COMMENT events have no `changes`.
- [ ] An agent assignee shows as the user the agent works for.
- [ ] The MCP `show` tool returns the same `changes` field.
- [ ] `make test` passes.
