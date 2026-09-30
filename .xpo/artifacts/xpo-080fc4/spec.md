# Spec: update { links } stores empty source_id and can't clear links

## What

1. When an update writes dependencies, set `source_id` to the owning issue's ID.
2. Treat `links: []` as "clear all links", not as "field not set".
3. Repair existing rows that have an empty `source_id` when they are read.
4. Reject self-links and duplicate `(target, kind)` pairs in an update's dependency list.
5. State in the tool and CLI descriptions that only `blocked_by` gates `start`.

## Why

`update { links }` currently writes malformed rows (`source_id: ""`). Those rows can't be repaired or removed: `link` says they "already exist", and `links: []` is rejected with "no fields set".

## How

### 1. Set source_id: `internal/exponential/update.go` → `buildUpdate`
After the ID is resolved and before `pruneUnchangedFields`, set `SourceID = id` on every entry of `payload.Dependencies`. `buildUpdate` is the single path that every update goes through (MCP, CLI `--json`, the web API handler, and RemoteTransport, which ends up in the server's LocalTransport), so this one change covers all of them.

In the same spot, return an error when:
- any `TargetID == id` → `cannot link an issue to itself`
- a `(TargetID, Kind)` pair appears more than once → `duplicate link <kind> <target>`

### 2. Let `links: []` clear the list
- `jsonio.LinksToDependencies`: return `nil` only when the input is `nil`. For an empty non-nil input, return `[]model.Dependency{}`. `encoding/json` already decodes `"links": []` into an empty non-nil slice, so no pointer type is needed.
- `UpdatePayloadEmpty` already checks for nil, so an empty non-nil slice counts as "set".
- Storage: `UpdatePayload.Dependencies` has no `omitempty`, so `[]` is serialized as `[]`, reads back as non-nil, and the projection applies it. Collapse (`mergeUpdatePayloads`, `pruneNoopUpdates`) also checks `!= nil`, so an empty non-nil list survives. Tests will lock this in.
- `pruneUnchangedFields`: if the list is empty and the issue already has no deps, it's pruned as a no-op. That's correct, but `update` will then report "Updated" with no event written; that's existing behavior for no-op updates.
- Labels: `labels: []` already works for the same reason (`in.Labels` is passed straight through, and `normalizeLabels` keeps an empty slice). A test covers it; no code change expected.

### 3. Repair on read: `internal/exponential/projection.go`
In `ProjectIssues`, after deps are applied for CREATE and UPDATE events, set `SourceID = issue.ID` on any dep whose `SourceID == ""`. Every read path (`show`, `list`, web API, `link`'s duplicate check, start gating) goes through the projection, so they all see correct data. The event log isn't rewritten. The next update to the issue's deps writes correct rows anyway.

### 4. Descriptions
- MCP `link`: "... Only `blocked_by` (set on the issue being started) prevents `start`; all other kinds are informational."
- MCP `update` `links` schema: "Replace the full dependency list; pass [] to remove all links. Only blocked_by gates start."
- MCP `add` `links` schema: the same note about gating.
- CLI `link` Long help: the same note.

## Acceptance criteria
- [ ] `update { links }` stores `source_id` = the issue's ID (MCP, CLI `--json`)
- [ ] `update { links: [] }` clears all links on the issue
- [ ] `update { links }` rejects self-links and duplicate (target, kind) entries
- [ ] Existing rows with an empty `source_id` are projected with the owner's ID, and `link` then works as expected
- [ ] The `link` / `update` / `add` link descriptions say that only `blocked_by` gates `start`
- [ ] Tests for all of the above; `make test` passes

## Decisions
- **Repair on read, don't migrate the log.** Normalizing in the projection is idempotent and needs no rewrite of the event log.
- **Validate in `buildUpdate`, not `ValidateUpdatePayload`.** `ValidateUpdatePayload` doesn't know the issue ID. `buildUpdate` does, and every path goes through it.
- **Reject duplicates rather than silently dedupe.** This matches `link`'s behavior. Existing data can't contain duplicates that would trip this check, because `link` already guards against them and the broken rows are normalized before they're compared.

## Out of scope
Reverse relationships in the backend `show`, an `unlink` tool, and changing which kinds gate `start`.
