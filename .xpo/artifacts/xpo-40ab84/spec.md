# Spec: MCP merge attribution

## What
MCP `merge` records MERGE and DONE as the configured user with no `on_behalf_of`. It should record them as the agent acting on behalf of the principal, like every other MCP write.

## Why
Agent merges currently look like direct human actions. That breaks the principal/via display (xpo-a9560c's dashed underline) and the agent/human attribution introduced in xpo-35fc16.

## How
- `Client.MergeIssue` and `Client.ResolveReviewIssue` call `c.syncLocal()` first, matching `StartIssue`. That copies `UserOverride` and `OnBehalfOf` onto the `LocalTransport` before any event is built or appended.
- No other behaviour changes. The CLI path has no override, so it keeps recording the human.
- Existing mis-attributed events stay as they are (the log is append-only).

## Acceptance criteria
- [ ] MCP merge records MERGE and DONE with `created_by` = agent and `on_behalf_of` = principal
- [ ] Merge without an override still records the configured user and no `on_behalf_of`
- [ ] A Go test covers the agent case (written first, failing before the fix)
- [ ] `make test` passes
