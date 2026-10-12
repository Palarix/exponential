# Walkthrough: remote mode guards for git actions

## What was built and why

The AI features ship for local mode only. Remote mode will be redesigned by the Team Mode epic (xpo-8757a9). Until then, remote mode had actions that ran git in the **server's** checkout, which nobody works in (xpo-5d307a). This issue hides or refuses those actions. Local mode does not change.

| Surface | Before, in remote mode | Now |
|---|---|---|
| Start Work button | Server created the worktree in its own checkout | Not rendered |
| Review Changes button and `MergeView` | Commits, diffs and mergeability came from the server's git; merge ran there | Not rendered; `MergeView` cannot open |
| `xpo serve` `POST /api/issues/{id}/start` and `/merge` | Ran git in the server's checkout | 501 with an explanation |
| `start` / `merge` MCP tools on `xpo serve` `/mcp` | Ran git in the server's checkout | Not registered |
| `start` / `merge` on a stdio `xpo mcp` with `remote.url` | `start` worked, `merge` failed with `ErrLocalOnly` | Not registered |

Review Changes looked safe because the board shows a local worktree path. It was not: only `GET /api/local/issues/{id}/worktree` is answered locally. `MergeView` loads `/api/issues/{id}/commits`, `/diff` and `/mergeability`, and the board forwarded those to the server.

## How the pieces fit

### Telling the UI which mode it is in

`GET /api/local/mode` (`handleLocalMode` in `internal/server/handlers.go`) returns `{"remote": s.ProxyURL != ""}`. It is registered in the `!s.Headless` block of `internal/server/router.go`, next to the worktree route. Like every `/api/local/` route, it is more specific than the proxy's `/api/{path...}` catch-all, so the board answers it in both modes. `xpo serve` (headless) does not register it.

`GET /api/config` would not work for this: in proxy mode the board forwards it, so it describes the server, not the board.

### Frontend

- `fetchLocalMode()` in `web/src/api/client.ts`, cached under `queryKeys.localMode`.
- `useGitActions()` in `web/src/api/queries.ts` returns `true` only when the route answered `remote: false`. It uses `staleTime: Infinity`, because a board's mode does not change while it runs.
- `PropertySidebar` has a `gitActions` prop (default `false`). It renders Start Work and Review Changes only when that prop is true.
- `IssueDetail` passes `gitActions` and renders `MergeView` only when it is true. The Review Changes button was the only way to open `MergeView`; the guard covers it anyway.

**Decision: hidden while unknown.** A local board shows the buttons one fast local request late. A remote board never shows them, not even for a moment.

### `xpo serve` refuses start and merge

`handleStartWork` and `handleMergeIssue` return 501 with `errGitActionsOnServe` when `s.Headless`. 501 says "this server does not support the function". 409 would suggest a state the caller could fix. This also protects against older boards, whose buttons still render.

The server's git read routes (diff, files, commits, mergeability) stay as they are. Nothing in the new board calls them in remote mode, and Team Mode removes them.

### MCP

`start` and `merge` moved from `register` to a new `registerWorkTools` in `internal/mcpserver/tools.go`:

- `newStdioServer` calls it only when `cfg.Remote.URL` is empty.
- `RegisterTools`, which `xpo serve` uses for `/mcp`, never calls it.

A stdio `start` in remote mode did work (remote status update, local worktree). We still removed it, to keep start and merge together until Team Mode. The CLI keeps `xpo start`. The read-only git tools (xpo-a7f115) stay on stdio in both modes, because they read the local checkout.

The chat prompt needs no change: the agent only sees the tools that are registered.

## Tests

- `internal/server/remote_guards_test.go`: the mode route on a local board, on a proxy board (answered locally; a fake backend records that nothing reached it) and on a headless server (absent). 501 for start and merge on a headless server. A local board still handles start.
- `internal/mcpserver/work_tools_test.go`: tool lists for the local stdio server, the stdio server with `remote.url`, and `RegisterTools`.
- `internal/mcpserver/integration_test.go`: two tests that registered tools by hand now build the real `newStdioServer`, so they check what agents actually see.
- `PropertySidebar.test.tsx`: both buttons render with `gitActions`. Neither renders in remote mode or while the mode is unknown.

## Acceptance criteria

- [x] `GET /api/local/mode` returns `remote: true` on a proxy board and `remote: false` on a local board; `xpo serve` does not register it. *Evidence: `TestLocalMode_*`; a live local board answered `{"remote":false}`.*
- [x] In remote mode the sidebar renders neither button, and `MergeView` cannot open. *Evidence: `PropertySidebar` "hides them in remote mode…"; the `mergeViewOpen && gitActions` guard in `IssueDetail`.*
- [x] While the mode is unknown, both buttons stay hidden. *Evidence: the same test with `gitActions` undefined; `useGitActions` returns `data?.remote === false`.*
- [x] In local mode both buttons and `MergeView` behave as before. *Evidence: `PropertySidebar` "shows Start Work and Review Changes in local mode"; the existing IssueDetail tests pass.*
- [x] `xpo serve` answers start and merge with 501; a local board still runs them. *Evidence: `TestHeadless_RefusesStartAndMerge`, `TestLocalBoard_StillStartsWork`.*
- [x] `xpo serve`'s `/mcp` does not list `start` or `merge`. *Evidence: `TestWorkTools_NotOnServe`.*
- [x] Stdio `xpo mcp` with `remote.url` does not list them; in local mode it does. *Evidence: `TestWorkTools_NotOnRemoteStdioServer`, `TestWorkTools_OnLocalStdioServer`.*
- [x] The CLI `xpo start` / `xpo merge` do not change. *Evidence: no changes under `cmd/`.*
- [x] `make test` passes. *Evidence: lint, frontend tests and all Go packages pass; `make frontend` builds.*

## Note for later

The `XPO_REMOTE_URL` environment variable does not set `remote.url`, so remote mode cannot be switched on from the environment for a quick manual test. This did not affect this issue.
