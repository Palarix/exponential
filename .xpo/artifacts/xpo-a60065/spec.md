# Spec: remote mode guards for git actions

## What

In remote mode, hide or refuse every action that runs git in the server's checkout. The board hides Start Work and Review Changes. The MCP servers in remote mode do not offer `start` or `merge`. `xpo serve` refuses its own start and merge routes. Local mode does not change.

## Why

The AI features ship for local mode only. Team Mode (xpo-8757a9) fixes remote mode properly. Until then, remote mode must not run git in the server's checkout (xpo-5d307a). Review Changes is guarded too: its commits, diff and mergeability routes are forwarded to the server, which reads its own git.

## Acceptance Criteria

- [ ] `GET /api/local/mode` returns `{"remote": true}` on a board in proxy mode and `{"remote": false}` on a local board. `xpo serve` does not register it.
- [ ] In remote mode, the issue sidebar renders neither Start Work nor Review Changes, and `MergeView` cannot open.
- [ ] While the mode is unknown (loading or failed), both buttons stay hidden.
- [ ] In local mode, both buttons and `MergeView` behave as before.
- [ ] `xpo serve` answers `POST /api/issues/{id}/start` and `POST /api/issues/{id}/merge` with 501 and a message. The board in local mode still runs them.
- [ ] `xpo serve`'s `/mcp` does not list `start` or `merge`.
- [ ] Stdio `xpo mcp` with `remote.url` set does not list `start` or `merge`. In local mode it lists them.
- [ ] The CLI commands `xpo start` and `xpo merge` do not change.
- [ ] `make test` passes.

## Flow

### Backend

1. Add `handleLocalMode` in `internal/server/handlers.go`. It returns `{"remote": s.ProxyURL != ""}`.
2. Register `GET /api/local/mode` in the `!s.Headless` block of `internal/server/router.go`, next to the worktree route.
3. In `handleStartWork` and `handleMergeIssue`, return 501 when `s.Headless`. Message: "start and merge run on your machine and are not available through xpo serve yet".
4. In `internal/mcpserver/tools.go`, move `start` and `merge` from `register` to a new `registerWorkTools`.
5. `newStdioServer` calls `registerWorkTools` only when `cfg.Remote.URL` is empty. `RegisterTools` (used by `xpo serve`) never calls it.

### Frontend

1. Add `fetchLocalMode()` in `web/src/api/client.ts` and a `useGitActions()` hook in `web/src/api/queries.ts`. The hook returns `true` only when the route answered `remote: false`. Use `staleTime: Infinity`, because the mode does not change while the board runs.
2. Add a `gitActions` prop to `PropertySidebar`. It renders Start Work and Review Changes only when `gitActions` is true.
3. In `IssueDetail`, pass `gitActions` from `useGitActions()`. Render `MergeView` only when `gitActions` is true.

### Tests (written first)

1. Server: the mode route in local and proxy mode; the route is absent on a headless server; 501 for start and merge on a headless server.
2. MCP: the tool lists of the stdio server in local mode, the stdio server with `remote.url`, and `RegisterTools`.
3. Frontend: `PropertySidebar` with a PLANNED issue that has branch stats. Both buttons render with `gitActions`, and neither renders without it.

## Decisions

| Question | Decision | Rationale |
|---|---|---|
| Mode signal | New `GET /api/local/mode` | `GET /api/config` is forwarded to the server in proxy mode. `/api/local/` routes are answered by the board in both modes. |
| Unknown mode | Hide the buttons | A flash of a wrong button in remote mode is worse than a short delay on a local board. |
| `xpo serve` response | 501 Not Implemented | The server does not support the function. 409 would suggest a fixable state. |
| Stdio `start` in remote mode | Not registered | `start` would work there (local worktree), but the guard keeps start and merge together until Team Mode. The CLI keeps `xpo start`. |
| Git read routes on `xpo serve` | Unchanged | Nothing in the board calls them once Review Changes is hidden. Team Mode removes them. |
| Chat prompt | Unchanged | The agent only sees registered tools. |

## Edge Cases

- **MEDIUM — old board against a new server.** Its buttons still render, but start and merge get a 501 with a clear message instead of acting on the server's checkout.
- **LOW — new board against an old server.** The board answers `/api/local/mode` itself, so the guard works.
- **LOW — an issue opened before the mode loads.** The buttons appear once the route answers, in local mode only.
