# Walkthrough: update { links } source_id and clearing links

## What was broken
The update path accepted `links` but never set `source_id` on the resulting `model.Dependency` values, unlike `add` and `link`. The rows it wrote looked like `{"source_id":"", ...}`. Because an explicit empty list was converted to nil, `update { links: [] }` also looked like "no fields set", so the broken rows couldn't be cleared either.

## How the fix fits together

### 1. Stamp and validate at the single choke point: `buildUpdate`
Every update, whether it comes from MCP, CLI `--json`, the web API handler or RemoteTransport (which the server executes locally), goes through `LocalTransport.buildUpdate` after the issue ID is resolved. The new `normalizeUpdateDependencies(id, deps)` runs there, before `pruneUnchangedFields`:
- It sets `SourceID = id` on every entry.
- It rejects `TargetID == id` ("cannot link an issue to itself").
- It rejects a repeated (target, kind) pair ("duplicate link …"). It does this with a set keyed on the whole `Dependency` struct, which works because SourceID has just been made identical for every entry.

Running this before pruning matters. The pruner compares the payload with the projected issue, and both now carry the same SourceID, so resubmitting an unchanged list is still a no-op.

`ValidateUpdatePayload` was deliberately left alone: it doesn't know the issue ID.

### 2. `links: []` means "clear"
`encoding/json` already decodes `"links": []` into an empty, non-nil slice. The information was being thrown away by `LinksToDependencies`, which tested `len(links) == 0`. It now tests `links == nil`. Downstream code already handled an empty list correctly:
- `UpdatePayloadEmpty` checks for nil, so the payload counts as set.
- `UpdatePayload.Dependencies` has no `omitempty`, so `[]` is stored as `[]` and reads back as non-nil.
- The projection and collapse (`mergeUpdatePayloads`, `pruneNoopUpdates`) both check `!= nil`.

`labels: []` already worked for the same reason. It now has tests so it stays that way.

### 3. Repair existing data on read
`ProjectIssues` calls `fillDependencySourceIDs` on each surviving issue, which fills an empty `SourceID` with the issue's ID. Because the projection feeds every reader (`show`, `list`, the web API, `link`'s duplicate check, start gating), the legacy rows look correct everywhere and the log doesn't need a migration. The function copies the slice before changing it, so event payloads kept in `issue.Events` are never mutated in place.

### 4. Documenting start gating
Agents had assumed `depends_on` gates `start`. The MCP `link` description (now the `linkToolDescription` constant, so a test can check it), the `links` jsonschema tags on `AddInput`/`UpdateInput`, and the CLI `link` long help all now say that only `blocked_by` on the issue being started prevents `start`. The `update` schema also documents `[]` to clear.

## Decisions
- **Repair on read, not a log migration.** It's idempotent, needs no rewrite of `issues.db`, and the next write to the issue's deps persists correct rows anyway.
- **Reject duplicates rather than dedupe.** This matches `link`. Legacy data can't contain duplicates that would trip the check, because `link` has always rejected (target, kind) repeats.

## Acceptance criteria
- [x] `update { links }` stores `source_id` = the issue's ID (MCP, CLI `--json`). Evidence: `TestUpdateIssue_DependenciesGetSourceID` asserts that no stored UPDATE event contains `"source_id":""`; `TestUpdateLinksSetsSourceIDAndClears` covers MCP end to end. CLI `--json` uses the same `ToUpdatePayload` → `UpdateIssue` path.
- [x] `update { links: [] }` clears all links. Evidence: `TestUpdateInputEmptyLinksClears` (JSON decode → non-empty payload), `TestUpdateIssue_EmptyDependenciesClears`, and the second half of `TestUpdateLinksSetsSourceIDAndClears`. Also checked by the user.
- [x] `update { links }` rejects self-links and duplicates. Evidence: `TestUpdateIssue_RejectsSelfLink`, `TestUpdateIssue_RejectsDuplicateLinks`.
- [x] Existing empty-`source_id` rows are projected with the owner's ID. Evidence: `TestProjectIssues_FillsEmptyDependencySourceID` covers both legacy CREATE and legacy UPDATE rows.
- [x] Descriptions state that only `blocked_by` gates `start`. Evidence: `TestLinkSchemasDocumentStartGating`, `TestLinkToolDocumentsStartGating`, and the CLI help text in `cmd/exponential/link.go`.
- [x] Tests for all of the above; `make test` passes (lint, 369 frontend tests, Go suite).
