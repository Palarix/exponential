# Spec: MCP update/add — position an issue before or after another issue

## What

Add two optional, mutually exclusive parameters, `before` and `after`, to the MCP `add` and `update` tools. Each takes an issue ID. The server computes a `sort_order` key that places the issue directly above (`before`) or below (`after`) the target in the global order. The CLI `add`/`update` get the same options, as `--before`/`--after` flags and in `--json` mode.

## Why

The board's manual order lives in `sort_order` (fractional index keys, `internal/sortorder`). Agents cannot set it, so "sort xpo-199a69 below xpo-64be51" fails and the user must drag the issue. Agents must never handle raw keys, so the API takes issue IDs.

## Acceptance Criteria

- [ ] `update { id: X, after: Y }` puts X directly below Y in the global order. No other issue sits between them.
- [ ] `update { id: X, before: Y }` puts X directly above Y in the global order.
- [ ] `add { title, after: Y }` and `add { title, before: Y }` create the issue at that position, not at the end of its status group.
- [ ] `before`/`after` accept a unique ID suffix, the same as `parent` and `links`.
- [ ] `update` with only `before` or `after` is a valid update. It does not fail with "no fields set".
- [ ] `before` and `after` together return an error.
- [ ] A target that does not exist returns an error. Nothing is written.
- [ ] A target equal to the issue itself returns an error.
- [ ] `update { status, after }` uses the computed key. The status-change auto-append (`buildUpdate`) does not override it.
- [ ] The position also works when X and Y have different statuses. After X moves to Y's status, X sits next to Y.
- [ ] The neighbor search uses all non-deleted issues, including DONE/CANCELED/DUPLICATE.
- [ ] When X already sits in the requested position, nothing is written and the reply is "X is already after Y" (or "before").
- [ ] CLI: `xpo update X --after Y`, `xpo add "t" --before Y`, and `--json` payloads with `before`/`after` behave the same as MCP.
- [ ] The `add`/`update` tool descriptions and the generated `mcp-tools.md` skill reference say that agents can reorder with `before`/`after`.
- [ ] `make test` passes.

## Flow

1. In `internal/jsonio/input.go`, add `Before string` and `After string` to `AddInput` and `UpdateInput`. Give them `jsonschema` descriptions.
2. In `ToCreatePayload` and `ToUpdatePayload`, return an error when both `Before` and `After` are set.
3. Add `internal/exponential/position.go` with a pure function `keyBeside(issues []*model.Issue, selfID, targetID string, below bool) (string, error)`.
4. In `keyBeside`, drop the issue `selfID` from the list. Sort the remaining keys.
5. In `keyBeside`, for `below`, take the neighbor as the smallest key strictly greater than the target's key. Generate `GenerateKeyBetween(target, neighbor)`.
6. In `keyBeside`, for `above`, take the neighbor as the largest key strictly smaller than the target's key. Generate `GenerateKeyBetween(neighbor, target)`.
7. In `keyBeside`, return the issue's current key when it already sits between the target and the neighbor. The update then becomes a no-op.
8. Add `type Position struct { Key, Target string; Above bool }` with `Unchanged(id string) string` for the no-op message.
9. Add `func (c *Client) PositionKey(selfID, before, after string) (*Position, error)`. It returns nil when both are empty.
10. In `PositionKey`, resolve the target with `c.GetIssue`. Reject the target when it equals `selfID`.
11. In `PositionKey`, load all issues with `c.ListIssues(FilterOptions{All: true})`. Call `keyBeside`.
12. In `mcpserver/tools.go` `update`, accept a position-only update in the empty check. Call `PositionKey` after `GetIssue` and set `payload.SortOrder`.
13. In `mcpserver/tools.go` `update`, return `pos.Unchanged(id)` when `UpdateIssue` returns no messages.
14. Extend `jsonio.UpdatePayloadEmpty` to count `SortOrder`.
15. In `mcpserver/tools.go` `add`, call `PositionKey("", before, after)` and set `payload.SortOrder`.
16. In `cmd/exponential/update.go`, apply the same steps in `--json` mode. Add `--before`/`--after` flags to flag mode.
17. In `cmd/exponential/add.go`, apply the same steps in `--json` mode. Add `--before`/`--after` flags to flag mode.
18. Move the `add`/`update` tool descriptions in `tools.go` to constants that mention `before`/`after`. Update the `add`/`update` rows in `GenerateMCPToolsRef`.
19. Regenerate `references/mcp-tools.md` in `.claude`, `.codex`, `.opencode`.
20. Add a CHANGELOG entry that references xpo-61232d.

## Decisions

- **Compute the key on the client side (`Client.PositionKey`), not in `LocalTransport.buildUpdate`.** It goes through `GetIssue`/`ListIssues`, so it works for the local and the remote transport. The server and the event model do not change. The cost is a small race: a concurrent reorder between the read and the write can give a slightly stale position. The board drag has the same race.
- **The neighbor comes from the global order of all issues, not from the target's status group.** The global neighbor is the closest key. So X lands directly next to Y in every filtered view: the backlog group, the board column, and sub-issue lists. This matches xpo-b92fb0.
- **Include terminal issues (DONE/CANCELED/DUPLICATE) when we look for the neighbor.** They hold keys in the same key space. If we skip them, the new key can fall on the wrong side of a terminal issue. That only matters after a reopen, but it costs nothing.
- **Keys equal to the target's key count as "not a neighbor".** Legacy data can hold duplicate keys. `GenerateKeyBetween` fails on equal bounds, so we look for the next strictly different key.
- **Already in position → no-op with an explicit message.** Repeated agent calls do not churn keys or create events. The reply says "X is already after Y" so the agent knows the call worked. The remote transport always replies "Updated X", because the server does not report no-ops.
- **CLI gets the flags too, named `--before`/`--after`.** `--json` gets them for free from `jsonio`, and flag mode is two flags. The names match MCP.
- **No raw `sort_order` parameter.** Agents only pass issue IDs (per the issue).

## Edge Cases

- **HIGH — status change plus position.** `update { status: PLANNED, after: Y }`. `buildUpdate` only auto-appends when `payload.SortOrder == nil`, so the explicit key wins. A test covers this.
- **MEDIUM — sub-issues.** `SortIssues` keeps a parent before its children. If an agent puts a child above its parent, the stored key changes but the views still show the parent first. We allow it without an error.
- **MEDIUM — the target is DONE.** DONE sorts by `updated_at`, not by key, in views. The key still gets written. Visible order changes only after the issue leaves DONE. We allow it without a warning.
- **LOW — target is last/first in the global order.** The neighbor is "", and `GenerateKeyBetween` handles an open bound.
- **LOW — target with an empty key.** `backfillSortOrder` fills every key during projection, so this does not happen. `keyBeside` returns an error if it does.
- **LOW — target is a deleted issue.** `GetIssue` fails, and the tool returns that error.

## Assumptions

- The remote transport sends `sort_order` in update and create payloads. Verified: `RemoteTransport` posts the full `model.CreatePayload`/`model.UpdatePayload` to `/api/draft`, and both carry `sort_order`.
- `ListIssues(All: true)` returns every non-deleted issue with its projected `SortOrder` for both transports.
