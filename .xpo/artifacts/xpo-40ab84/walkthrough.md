# Walkthrough: MCP merge attribution

## The bug
Every MCP write is supposed to be recorded as the agent acting for a person: `created_by` is the agent (`claude-code/… <agent@mcp>`) and `on_behalf_of` is the principal. MCP `merge` broke that rule. Its MERGE event and DONE transition were recorded as the configured user, with no `on_behalf_of`. As a result, agent merges looked like direct human actions in the UI, with no dashed underline (xpo-a9560c) and the wrong agent/human attribution (xpo-35fc16).

## Why it happened
`Client` holds `UserOverride` (the agent identity) and `OnBehalfOf`. `LocalTransport` is what actually stamps events, and it only receives those two fields when `c.syncLocal()` runs. Every `Client` delegation method (`GetIssue`, `UpdateIssue`, …) calls `syncLocal()` first, and so does `StartWork`.

`MergeIssue` and `ResolveReviewIssue` are git-level methods. They call `c.Transport.GetIssue` and `c.local.appendEvent` directly, and they never synced. On the CLI that made no difference, because there is no override. The MCP server, however, creates a **fresh `Client` per request** (`mcpserver/server.go` `clientFor`). Nothing had synced that client before the merge ran, so:
- `buildMergeEvents` stamped `CreatedBy: c.Transport.GetUser()`, which fell back to the config/git user
- `appendEvent` saw an empty `OnBehalfOf`

## The fix
`MergeIssue` and `ResolveReviewIssue` now call `c.syncLocal()` first, the same way `StartWork` does. A comment in `MergeIssue` records why: the client is fresh on every MCP request.

## Tests
`internal/exponential/merge_test.go`:
- `TestMergeIssue_AgentOnBehalfOfPrincipal` sets `UserOverride` and `OnBehalfOf` on the client, merges, and asserts that both MERGE and DONE carry the agent as `created_by` and the principal as `on_behalf_of`. It was written first and failed with `created_by = "Test <test@test.com>"` and an empty `on_behalf_of`.
- `TestMergeIssue_DirectUserHasNoOnBehalfOf` guards the CLI path: the configured user is recorded and `on_behalf_of` is empty.
- The shared `mergeEvents` helper handles both payload shapes (struct, or a map after the JSON roundtrip).

## Not done
Events recorded before this fix (for example xpo-334821's MERGE and DONE) keep their old attribution, because the event log is append-only.

## Acceptance criteria
- [x] MCP merge records MERGE and DONE with `created_by` = agent and `on_behalf_of` = principal (`TestMergeIssue_AgentOnBehalfOfPrincipal`)
- [x] Merge without an override still records the configured user and no `on_behalf_of` (`TestMergeIssue_DirectUserHasNoOnBehalfOf`)
- [x] A Go test covers the agent case (written first, failed before the fix)
- [x] `make test` passes (exit 0)
