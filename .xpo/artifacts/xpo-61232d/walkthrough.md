# Walkthrough: MCP update/add — position an issue before or after another issue

## What was built

Agents can now reorder issues. The MCP `add` and `update` tools take `before` or `after`, each an issue ID. The issue lands directly above or below that issue in the manual order. The CLI has the same options as `--before`/`--after` flags and as `before`/`after` fields in `--json` payloads.

Agents never see `sort_order` keys. They pass issue IDs, and xpo computes the key.

## How the pieces fit together

```
MCP add/update, CLI add/update (flags or --json)
  → jsonio.AddInput / UpdateInput      (Before, After; rejects both set)
  → Client.PositionKey(selfID, before, after)
      → GetIssue(target)                (resolves suffix, rejects self)
      → ListIssues(All: true)           (global order, all non-deleted issues)
      → keyBeside(...)                  (pure key computation)
  → payload.SortOrder = pos.Key
  → Client.AddIssue / UpdateIssue       (normal sort_order write, same as a board drag)
```

### `internal/exponential/position.go`

- `keyBeside(issues, selfID, targetID, below)` is a pure function. It collects every key except the moving issue's own key and sorts them. For `below`, the neighbor is the smallest key strictly greater than the target's key. For `above`, the neighbor is the largest key strictly smaller. It then calls `sortorder.GenerateKeyBetween` with the target and the neighbor. An empty neighbor is an open bound.
- Below, the search for "strictly greater" uses `sort.SearchStrings(keys, targetKey+"\x00")`. Base62 keys never contain `\x00`, so this finds the first key past every copy of the target's key.
- If the moving issue's current key already sits between the target and the neighbor, `keyBeside` returns that key unchanged. `pruneUnchangedFields` in `buildUpdate` then drops the field, and no event is written.
- `Position` holds the key, the resolved target ID, and the direction. `Unchanged(id)` builds the "X is already after Y" message.
- `Client.PositionKey` validates and resolves the request. It returns nil when neither field is set, so callers can always call it.

### Callers

- `mcpserver/tools.go`: `add` and `update` call `PositionKey` after payload validation and set `payload.SortOrder`. `update` accepts a position-only call in its "no fields set" check. When `UpdateIssue` returns no messages and a position was set, it replies with `pos.Unchanged(id)`.
- `cmd/exponential/add.go` and `update.go`: the same steps in `--json` mode and in flag mode. Flag-mode `update` now calls `GetIssue` first, so the moving issue's ID is resolved before it is compared with the target.
- `jsonio.UpdatePayloadEmpty` now counts `SortOrder`.

### Agent guidance

- The `add`/`update` tool descriptions are now constants (`addToolDescription`, `updateToolDescription`), so a test can check them.
- `GenerateMCPToolsRef` (in `agents.go`) mentions `before`/`after` in the `add` and `update` rows. The checked-in `references/mcp-tools.md` files for Claude, Codex and OpenCode were regenerated with `WriteAgentSkill`. Two Claude skill files with local edits were left alone.

## Key decisions

- **The key is computed in `Client`, not in `LocalTransport.buildUpdate`.** `Client` reads through `GetIssue`/`ListIssues`, so the feature works with the local and the remote transport. The event model and the server do not change. The cost is a small read-then-write race, which the board drag also has.
- **The neighbor comes from the global order.** It is the closest key of all issues, so the issue sits next to the target in every filtered view: backlog groups, board columns, and sub-issue lists. This follows xpo-b92fb0, which made `sort_order` one global rank.
- **Terminal issues count as neighbors.** DONE/CANCELED/DUPLICATE issues share the key space. If we skipped them, a key could fall on the wrong side of one, and that would show after a reopen.
- **Duplicate keys are skipped.** Legacy data can hold equal keys, and `GenerateKeyBetween` rejects equal bounds.
- **Status change plus position.** `buildUpdate` only appends to the end of the new status group when `payload.SortOrder` is nil. An explicit position therefore wins.
- **No-op message.** A repeated call writes nothing and says so. The remote transport always replies "Updated X", because the server does not report no-ops.

## Acceptance criteria

- [x] `update { id: X, after: Y }` puts X directly below Y. Evidence: `TestUpdatePositionOnly` (MCP); `TestKeyBeside_BelowPicksGlobalNeighbor`.
- [x] `update { id: X, before: Y }` puts X directly above Y. Evidence: `TestUpdatePositionOnly`; `TestKeyBeside_AbovePicksGlobalNeighbor`.
- [x] `add` with `after`/`before` creates the issue at that position. Evidence: `TestAddWithPosition` (MCP), `TestPositionCLI_Add` (CLI).
- [x] `before`/`after` accept a unique ID suffix. Evidence: `TestClient_PositionKey_UpdateAcrossStatuses` passes a 4-character suffix and checks the resolved `Target`.
- [x] A position-only `update` is valid. Evidence: `TestUpdatePositionOnly`, `TestUpdatePayloadEmptyCountsSortOrder`.
- [x] `before` and `after` together return an error. Evidence: `TestAddInputBeforeAndAfterRejected`, `TestUpdateInputBeforeAndAfterRejected`, `TestUpdatePositionErrors` ("both"), `TestPositionCLI_UpdateFlagsAndJSON`.
- [x] A missing target returns an error, and nothing is written. Evidence: `TestUpdatePositionErrors` ("missing"), `TestAddWithPosition`, `TestClient_PositionKey_Errors`.
- [x] A target equal to the issue itself returns an error. Evidence: `TestUpdatePositionErrors` ("self"), `TestClient_PositionKey_Errors`.
- [x] `update { status, before }` keeps the computed position. Evidence: `TestUpdateStatusWithPositionKeepsPosition`.
- [x] The position works across statuses. Evidence: `TestClient_PositionKey_UpdateAcrossStatuses` moves a BACKLOG issue next to a PLANNED one and checks the PLANNED order.
- [x] The neighbor search uses all non-deleted issues, including terminal ones. Evidence: `PositionKey` uses `ListIssues(FilterOptions{All: true})`. Projection already removes deleted issues.
- [x] An issue already in place is a no-op with "X is already after Y". Evidence: `TestKeyBeside_AlreadyInPositionKeepsKey`, `TestUpdatePositionAlreadyInPlace`, `TestPosition_Unchanged`.
- [x] The CLI flags and `--json` behave the same as MCP. Evidence: `TestPositionCLI_UpdateFlagsAndJSON`, `TestPositionCLI_Add`.
- [x] Tool descriptions and `mcp-tools.md` mention `before`/`after`. Evidence: `TestAddUpdateDescriptionsMentionPosition`, `TestMCPToolsRef_AddUpdateMentionPosition`, and the regenerated reference files.
- [x] `make test` passes. Evidence: exit code 0 (Go vet, eslint, 832 frontend tests, Go tests). Approved by the user after review.
