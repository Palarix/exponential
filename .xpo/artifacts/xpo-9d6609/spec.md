# Links are bidirectional

## What

A link is one relationship. It's stored once, on whichever issue created it, and it's true from both sides. `A blocks B` also means `B blocked_by A`. Every reader (start-gating, drive, `show` in MCP/CLI/TUI, the web API) sees the combined view. Every writer (`link`, `update { links }`, the web sidebar) works on the combined view.

## Why

Links only count for the issue that stores them. That causes two problems:
- **Wrong gating:** if `A blocks B` is stored on A, B can still be started while A is open.
- **Missing context:** `show B` doesn't list the link, and `link` lets the same relationship be recorded from both sides.

## How

### Model

- `Issue.Dependencies` stays as the **owned** rows: the storage truth, replayed from events. It isn't renamed, so events, collapse and `pruneUnchangedFields` keep working.
- New `Issue.Links []Dependency` holds the **combined view**, computed at the end of `ProjectIssues` (after deleted issues are filtered out):
  - every owned row, plus
  - for every row `X kind Y` owned by another live issue X with Y = this issue: a derived entry `{source_id: Y, target_id: X, kind: InverseKind(kind), derived: true}`. `relates_to` maps to itself.
  - Deduplicated by (target, kind). An owned row wins over a derived one.
- `Dependency` gets `Derived bool \`json:"derived,omitempty"\``. Owned rows never set it, and `normalizeUpdateDependencies` clears it, so it's never persisted.

### Readers switch to `Links`

- Start-gating in `update.go`.
- `hasUnresolvedBlockers` in `drive.go`.
- MCP `show`, CLI `show --json`, the TUI `details.go`, and server `responses.go`. `DependencyResponse` gets `derived`.
- `remote_transport.go` rebuilds both lists from the API: `Links` = everything, `Dependencies` = the entries that aren't derived.

### Writers: `UpdatePayload.Dependencies` means "the issue's full relationship set"

In `Update` (update.go), when `payload.Dependencies != nil`, reconcile against `issue.Links`, matching each entry by (target, kind):
- **Desired entry already derived** (stored on the other issue): drop it from the owned payload. It stays where it is.
- **Current derived entry missing from the desired set:** emit an UPDATE event on the owning issue that removes its row. It is broadcast like the other cascade events.
- **Everything else:** becomes the owned list, normalized as today.

This runs **before** `pruneUnchangedFields`/`payloadEmpty`. An update that only removes a derived link still produces the cross-issue event, even though the primary payload ends up empty.

- `link` (MCP plus both CLI paths) checks for existing links and appends against `src.Links`. "Already exists" now covers a relationship stored from the other side.
- Web sidebar: it already sends `issue.dependencies` back in full. With the API returning the combined view, add/remove works through reconciliation. Add `derived?: boolean` to the TS type. Removing a derived link no longer quietly does nothing.
- `collectEdges` in `useDepGraph.ts` already canonicalizes and deduplicates, so derived entries produce no duplicate edges. Add a test for that.

### Docs

The `link`/`update` tool descriptions and the CLI help say relationships are bidirectional: `A blocks B` gates B's start whichever side stores it.

## Decisions

- **Derive on read rather than storing both rows.** There's no migration, nothing to keep in sync, and existing data picks up the new behaviour immediately.
- **Combined view in the existing `dependencies` output field, with a `derived` flag**, rather than a separate `incoming` list. Agents and the web UI see one list, and writes that send it back round-trip correctly.
- **Gating stays kind-based.** `blocked_by` (owned or derived from `blocks`) gates `start`. Drive also skips `depends_on` (owned or derived from `dependency_of`), as it does today. This issue doesn't change which kinds gate.
- **Compatibility:** an older remote client that sends only owned rows in `update { links }` would remove derived links. That's acceptable: the client and server ship together.

## Edge cases

- Both sides recorded (`A blocks B` and `B blocked_by A`): each issue shows one entry, which is the owned one. Removing it from B's links removes B's row and also A's row, since both rows describe the same relationship. A relationship goes away only when no row for it remains.
- A derived link from a deleted issue doesn't appear.
- An owned link to a deleted or missing target stays visible, unchanged from today.

## Acceptance criteria

- [ ] `show B` (MCP, CLI, API) lists `blocked_by A` with `derived: true` when `A blocks B` is stored on A.
- [ ] A relationship stored from both sides appears once per issue.
- [ ] `start B` is rejected while A is open when `A blocks B` is stored only on A.
- [ ] Drive treats B as blocked in the same situation, and likewise for `A dependency_of B`.
- [ ] `link B A blocked_by` reports "already exists" when `A blocks B` is stored on A.
- [ ] `update { id: B, links: [] }` removes `A blocks B` from A.
- [ ] `update { id: B, links: <current combined list> }` is a no-op (no events).
- [ ] The `derived` flag is never persisted in event payloads.
- [ ] Web: no duplicate graph edges, and removing a derived link in the sidebar removes it.
- [ ] Tests cover the above; `make test` passes.
