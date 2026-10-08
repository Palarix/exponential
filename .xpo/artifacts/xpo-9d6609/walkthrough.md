# Walkthrough: Links are bidirectional

## What was built and why

Links used to count only for the issue that stored them. If `A blocks B` was stored on A, B didn't know it was blocked: `start B` went through, drive picked B, and `show B` didn't list the link. Now a link is one relationship, stored once and true from both sides. Projection derives the inverse on the other issue, and every reader and writer works on that combined view.

## How the pieces fit together

### 1. Two lists on `model.Issue`

- `Dependencies` stays as the **owned** rows: what this issue's events stored. Events, `storage/collapse.go` and `pruneUnchangedFields` all work on this list and didn't change.
- `Links` (new) is the **combined view**. It isn't stored; projection computes it.
- `Dependency.Derived` (`json:"derived,omitempty"`) marks an entry that's stored on the other issue.

Keeping `Dependencies` as the storage truth, rather than redefining it, means event replay and collapse need no changes. The derived flag can't leak into storage, because `normalizeUpdateDependencies` clears it on every write.

### 2. Projection: `deriveLinks` (`projection.go`)

`deriveLinks` runs at the end of `ProjectIssues`, after deleted issues are filtered out:
1. Each issue's `Links` starts as a copy of its `Dependencies`.
2. For every owned row `X kind Y` where Y is a live issue, it adds `{source: Y, target: X, kind: InverseKind(kind), derived: true}` to Y, unless Y already has that (target, kind). Owned rows were added first, so a relationship recorded from both sides keeps the owned row.

IDs are iterated in sorted order, so the order of `Links` is deterministic. `relates_to` maps to itself through `InverseKind`'s default case. Links from deleted issues disappear because deleted issues aren't in the map any more.

### 3. Readers switched to `Links`

| Reader | File |
|---|---|
| Start-gating (`blocked_by`) | `internal/exponential/update.go` |
| Drive `hasUnresolvedBlockers` | `internal/exponential/drive.go` |
| MCP `show` | `internal/mcpserver/tools.go` |
| CLI `show --json` | `cmd/exponential/show.go` |
| TUI details | `internal/ui/details.go` |
| Web/remote API (`DependencyResponse.Derived`) | `internal/server/responses.go` |

The remote transport (`apiIssueToModel`) receives the combined list from the API. It puts everything in `Links` and the entries that aren't derived in `Dependencies`, so a remote client sees the same two lists a local one does.

### 4. Writers: `reconcileLinks` (`update.go`)

The key semantic change: **`UpdatePayload.Dependencies` is the issue's full set of relationships**, not just its owned rows. This is what lets callers send back whatever `show` gave them. That's how the web sidebar and `link` already worked.

`buildUpdate` calls `reconcileLinks` after normalization and before pruning:
- **Desired link stored on the other side** (and not owned here): it's dropped from the owned payload and stays where it is.
- **A row stored on another issue X that targets this issue and is no longer desired:** it's removed from X with a separate UPDATE event on X containing X's remaining rows. All other issues are scanned, not just derived entries, so a relationship stored on *both* sides is fully removed.
- **Everything else** becomes the owned list.

Reconciliation runs before `pruneUnchangedFields`/`payloadEmpty` and returns its events even when the primary payload ends up empty. An update that only removes a derived link must still write.

`link` (MCP and both CLI paths) now appends to `src.Links` and checks for duplicates against it. "Already exists" therefore covers the inverse, and the derived links that come back in the full set are left untouched, not removed.

### 5. Web

The sidebar already sent `issue.dependencies` back in full on add and remove, so with the API returning the combined view it needed no changes. Removing a derived relation now works through reconciliation. `collectEdges` already canonicalized and deduplicated edges; a test now covers the owned-plus-derived case. The TS `Dependency` type gained `derived?`.

## Key decisions

- **Derive on read, don't store both rows.** There's no migration, nothing to keep in sync, and existing data picks up the new behaviour immediately.
- **One `dependencies` output field with a `derived` flag**, rather than a separate `incoming` list. A single list round-trips through `update { links }`.
- **Gating stays kind-based.** `blocked_by` (owned, or derived from `blocks`) gates `start`. Drive also skips `depends_on` (owned, or derived from `dependency_of`), as it did before. Which kinds gate didn't change.
- **Compatibility:** an older remote client that sends only owned rows would remove derived links. That's accepted, since the client and server ship together.
- **Broadcasts:** only the edited issue is broadcast, not the other issue whose row was removed. This matches the existing parent-cascade events.

## Acceptance criteria

- [x] `show B` (MCP, CLI, API) lists `blocked_by A` with `derived: true` when `A blocks B` is stored on A. Evidence: `TestShowListsDerivedLink`, `TestProjectIssues_DerivesInverseLinks`, `TestIssueToResponse_IncludesDerivedLinks`.
- [x] A relationship stored from both sides appears once per issue. Evidence: `TestProjectIssues_DedupesBothSides`.
- [x] `start B` is rejected while A is open when `A blocks B` is stored only on A. Evidence: `TestUpdateIssue_DerivedBlocksGatesStart`.
- [x] Drive treats B as blocked in the same situation, and likewise for `A dependency_of B`. Evidence: `TestHasUnresolvedBlockers_Derived` (blocks, dependency_of).
- [x] `link B A blocked_by` reports "already exists" when `A blocks B` is stored on A. Evidence: `TestLinkRejectsInverseOfExisting`.
- [x] `update { id: B, links: [] }` removes `A blocks B` from A. Evidence: `TestUpdateIssue_EmptyLinksRemovesDerived`, `TestUpdateIssue_RemovingLinkRemovesBothSides`.
- [x] `update { id: B, links: <current combined list> }` is a no-op (no events). Evidence: `TestUpdateIssue_SendingCombinedLinksBackIsNoop`.
- [x] The `derived` flag is never persisted in event payloads. Evidence: `TestUpdateIssue_DerivedFlagNotPersisted`.
- [x] Web: no duplicate graph edges, and removing a derived link in the sidebar removes it. Evidence: the vitest case "emits one edge when the API returns a link and its derived inverse"; the sidebar's full-list save is covered by the reconciliation tests and was approved in user testing.
- [x] Tests cover the above; `make test` passes.

Additional coverage: `TestLinkKeepsDerivedLinksOnSource`, `TestUpdateIssue_AddingLinkKeepsDerived`, `TestProjectIssues_RelatesToIsSymmetric`, `TestProjectIssues_SkipsLinksFromDeletedIssues`, `TestAPIIssueToModel_SplitsDerivedLinks`.

## Follow-up

- xpo-9d5d00 (`unlink`) can now be a thin wrapper: compute `Links` minus the one entry and send it through `update`. Reconciliation removes it from whichever side stores it.
