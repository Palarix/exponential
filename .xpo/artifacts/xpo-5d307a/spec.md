# Spec: remote mode — start and merge run on the client

## What

In remote mode, starting and merging an issue run git on the user's machine and record issue data on the server. This covers the CLI, the stdio MCP server and the board's buttons. The board also answers its git read routes (diff, files, commits, mergeability, merged changes) locally. The chat's remote `/mcp` path is out of scope: xpo-6dd7a0 moves the chat to the stdio server.

This is a **stop-gap** with a **soft merge contract**: the client merges locally and reports the merge. The server does not verify it, like Jira or Linear. The long-term team server (xpo-8757a9) replaces this with merges that the server observes on origin.

## Why

Part of xpo-3798b7. Found while specifying xpo-a7f115. The rule: **code runs where the code is; issue data lives wherever the transport points.** `docs/design.md` says the same: `StartWork`, `MergeIssue` and `Review` "only make sense locally". Today in remote mode:

| Entry point | Today | After |
|---|---|---|
| CLI / stdio MCP `start` | Correct: remote status update, local worktree | Unchanged |
| CLI / stdio MCP `merge` | Fails: `MergeIssue` returns `ErrLocalOnly` | Local git merge; MERGE + DONE recorded on the server |
| Board Start Work / Merge | Proxied: the server runs git in its own checkout | Answered by the board: local git, remote issue data |
| Board diff / files / commits / mergeability / merged | Proxied: the server reads its own git | Answered by the board from local git |
| Chat `start` / `merge` (remote `/mcp`) | The server runs git in its own checkout | Unchanged here; fixed by xpo-6dd7a0 |

Work mode chat needs this: an agent works in a local worktree and merges it.

## Acceptance Criteria

- [ ] `MergeIssue` with a remote transport merges the branch into the local default branch with the same strategies and checks as local mode.
- [ ] After a remote-mode merge, the server has a MERGE event and a DONE transition for the issue. Their author is the authenticated user.
- [ ] In remote mode, the MERGE payload contains the final `merge_sha`.
- [ ] A remote-mode merge does not write `.xpo/issues.db` or `.xpo/artifacts/` in the local checkout, and does not amend them into the merge commit.
- [ ] A remote-mode merge uses the local branch. It ignores the `branch_stats` the server sends.
- [ ] If recording on the server fails after the local merge, the error gives the merge SHA and says not to re-run merge.
- [ ] The result message in remote mode says the merge is local and must be pushed.
- [ ] A new server route `POST /api/issues/{id}/record-merge` records MERGE + DONE without running git. It needs the `issue.merge` capability.
- [ ] In proxy mode, the board answers these routes itself: `POST /api/issues/{id}/start`, `POST /api/issues/{id}/merge`, `GET /api/issues/{id}/mergeability`, `GET /api/issues/{id}/merged`, `GET /api/issues/{id}/diff`, `GET /api/issues/{id}/files`, `GET /api/issues/{id}/commits`, `GET /api/issues/{id}/commits/{sha}/diff`, `GET /api/commits/{sha}`.
- [ ] In proxy mode, those board handlers read the issue through the remote transport and the branch from local git.
- [ ] In local mode (no proxy), board routes and `MergeIssue` behave as before. Existing tests pass unchanged.
- [ ] `make test` passes.

## Flow

### 1. Server: record a merge without git

1. Refactor `recordMerge` and `buildMergeEvents` in `internal/exponential/merge.go` around a merge value: `{Branch, BaseSHA, MergeSHA, Strategy}`.
2. Add `(c *Client) RecordMerge(id string, rec ...) ([]string, error)`. With a local transport, it appends MERGE + DONE, as `recordMerge` does today.
3. Add `RecordMerge` to the `Transport` interface.
4. Implement it in `LocalTransport` with step 2.
5. Implement it in `RemoteTransport` as a post to `/api/issues/{id}/record-merge`.
6. Add `handleRecordMerge` in `internal/server/handlers.go`. It validates the body and calls `client.RecordMerge` with the authenticated user as `UserOverride`.
7. Register it with `handleCap(..., "issue.merge", ...)` in the non-proxy route block of `router.go`.
8. Make the handler reject an issue that is DONE, CANCELED or DUPLICATE with 409.

### 2. `MergeIssue` with a remote transport

1. Remove the `c.local == nil` guard.
2. Add `(c *Client) localBranchStats(issue)`. With a remote transport, it drops the server's `BranchStats` and computes them from local git.
3. With a local transport, make `localBranchStats` keep the `FillLocalBranchStats` behavior.
4. Use `localBranchStats` in `MergeIssue` and in `ResolveReviewIssue`.
5. Keep the hub checks, the git merge and the commit unchanged.
6. After the commit, call `c.RecordMerge` with the transport in use.
7. In remote mode, pass the final `HEAD` SHA as `MergeSHA`.
8. Skip the `.xpo` bookkeeping amend when the transport is remote.
9. Keep worktree removal and branch deletion unchanged.
10. In remote mode, add the message `Merged locally — push <base> to share it`.

### 3. Board: answer git routes locally in proxy mode

1. In `router.go`, inside the `!s.Headless` block, register the nine routes from the acceptance criteria when `s.ProxyURL != ""`. They are more specific than the `/api/{path...}` catch-all, so they win.
2. In proxy mode, make `resolveIssueBranch` load the issue with `exponential.NewClient(s.Config).GetIssue(id)`.
3. Make `resolveIssueBranch` then call `localBranchStats`. Local mode keeps `s.GetProjectedIssues()`.
4. In proxy mode, make `handleMergedChanges` load the issue through the remote transport. `FindMergeCommit` then uses `merge_sha` from the payload.
5. Leave `handleStartWork` and `handleMergeIssue` unchanged. `NewClient(s.Config)` already returns a remote-transport client in proxy mode.
6. Add a comment next to the `/api/local/` block: git routes describe this machine.

### 4. Tests (written first)

1. `internal/exponential`: `MergeIssue` with an `httptest` remote. Assert the local squash commit, the `record-merge` request body with `merge_sha`, no local `issues.db` change, and the push message.
2. Remote merge ignores a server `branch_stats` that names `origin/<branch>`.
3. Remote record failure: the error names the merge SHA.
4. `internal/server`: `record-merge` appends MERGE + DONE, rejects a DONE issue and enforces `issue.merge`.
5. `internal/server`: in proxy mode, `POST /api/issues/{id}/start`, `GET .../diff` and `GET .../merged` are answered locally. A fake remote records which paths reached it.
6. Proxy-mode merge through the board: local commit plus one `record-merge` call to the fake remote.

## Decisions

| Question | Decision | Rationale |
|---|---|---|
| Merge contract | Soft: the client merges and reports, and the server trusts it | Simplest stop-gap, no worse than Jira or Linear. Team mode later observes merges on origin (xpo-8757a9). |
| How the client records a merge | New `POST /api/issues/{id}/record-merge` | `/api/draft` only takes CREATE, UPDATE, COMMENT and DELETE, and is gated by weaker capabilities. A dedicated route keeps the `issue.merge` capability. |
| Which branch to merge in remote mode | Local branch from local git | The server's `branch_stats` come from the server's remote-tracking branches and can be stale or `origin/…`. |
| `.xpo` bookkeeping in remote mode | Skipped | The issue log lives on the server. Writing the local copy would make it diverge. |
| `merge_sha` in remote mode | Recorded | No amend follows the merge commit, so its SHA is final. Local mode keeps an empty `merge_sha`. |
| Push after merge | No auto-push; a message tells the user | Simplest. Pushing is outward-facing and team workflows differ. |
| Board git routes in proxy mode | Answered locally under the proxy catch-all | Carve-outs are acceptable for this stop-gap. The Merge view must show what will be merged, which is local. |
| Server's own start/merge/git routes | Unchanged | Local-mode boards use the same handlers. The chat's remote `/mcp` still reaches them until xpo-6dd7a0. |

## Edge Cases

- **HIGH — record fails after the local merge.** The code is merged locally, but the issue is not DONE. The error gives the SHA and says to set DONE by hand. It says not to re-run merge, because the branch has no commits ahead any more.
- **HIGH — the server cannot verify the merge.** `record-merge` trusts an authenticated `issue.merge` holder. This is the accepted soft contract.
- **MEDIUM — issue already DONE on the server.** `record-merge` returns 409 after the local merge. The error says the code is merged and the issue was already closed.
- **MEDIUM — no local branch.** Remote mode returns "no branch found" even when the server knows a pushed branch. The user runs `git fetch` and checks the branch out, or merges where the branch is.
- **MEDIUM — merged changes for an issue merged on another machine.** `GET .../merged` returns "merge commit not found" until the user pulls the default branch. The error names the SHA.
- **LOW — live updates in proxy mode.** The board mirrors the remote's event stream (`internal/server/live.go`, `startMirror`). The server's MERGE and DONE events reach every board through it. No change needed.

## Assumptions

- In proxy mode, `s.Config.Remote.URL` is set, so `NewClient(s.Config)` uses `RemoteTransport`. `cmd/exponential/board.go` sets `ProxyURL` from the same value.
- The user's clone has the issue branch locally (created by `start` on this machine, or checked out).

## Open Questions

None. Both earlier questions are resolved in Decisions (no auto-push; server routes unchanged).
