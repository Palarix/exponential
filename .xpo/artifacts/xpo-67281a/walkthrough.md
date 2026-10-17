# Walkthrough: Remove remote mode

## What and why

xpo had a remote mode. `xpo serve` ran a headless server with SSH-key auth and JWTs. Clients used `RemoteTransport` or a proxying `xpo board`. That server described branch state from its own checkout, which nobody works in, and start and merge ran there too (xpo-5d307a). Fixing that meant moving git state over the wire, so we decided xpo is local-only. Issue data travels with the code through git. Team mode (xpo-8757a9) will be designed from scratch.

## How the pieces fit now

- **Client.** `Client.Transport` is a `*LocalTransport` (`internal/exponential/client.go`). The old split between `Transport` (an interface) and `local` (the concrete type, nil in remote mode) is now one field. The `c.local == nil` guards (`merge`, `maintenance`, `review`, `drafts`, `drive`) are gone, and so is `ErrLocalOnly`. I kept the field name `Transport` so that the outside callers in `board.go`, `start.go`, `merge.go` and `review.go` didn't change.
- **Board server.** `SetupRoutes` (`internal/server/router.go`) is one flat list of `mux.HandleFunc` calls. Gone:
  - the `handle`/`handleCap` auth wrappers,
  - the proxy branch,
  - the `Headless` checks,
  - the `/mcp` mount.

  `startMirror` and `internal/live/mirror.go` went too. The event stream is always the board's own.
- **Board access.** The `--host` token guard (`tokenguard.go`, `hostguard.go`) and its `/auth` page stay. They belong to the local board, not remote mode. Only `/auth/challenge` and `/auth/verify` went.
- **MCP.** `newStdioServer` always registers `start` and `merge`; they are folded back into `register`. `RegisterTools`, `httpUserOverride` and the `onEvent`/`broadcast` hook only served `xpo serve`, so they went too.
- **Chat.**
  - `RunnerConfig.CanWork` and `ErrWorkUnavailable` existed only to refuse Work Mode in remote mode.
  - `Profile.CanWork` stays: it still means "the project is a git repo" (xpo-e627e1).
  - `MCPTarget` has only `Executable`. The Codex bearer-token plumbing is gone.
- **Config.** `Remote` and `Permissions` are gone from `Config`. `internal/config/remote.go` is renamed to `user.go`. It now holds only `UserConfig.Projects` and the inbox cursor; `GetInboxLastRead()` and `SetInboxLastRead(t)` lost their `remoteURL` parameter.
- **Drafts.** Owner scoping always got an empty owner without auth, so the `owner` parameters are gone.
- **`xpo whoami`.** It prints `Identity:` from `GetUser()`: the configured user, else git config. Outside a project, it uses an empty config.
- **Frontend.**
  - `LocalMode.remote`, `useGitActions` and the `gitActions` prop are gone.
  - Start Work, Review Changes, View changes and MergeView render under their other conditions.
  - They no longer wait for `/api/local/mode` to load.

## Decisions

| Question | Decision |
|---|---|
| Keep the `Transport` interface? | No. It had one implementation and no mocks. |
| `xpo whoami` | Keep it as a local identity command. |
| `permissions` config, draft owner scoping, Docker files | Removed. |
| Old `remote:`/`permissions:` keys and `user.yaml` `servers:` | Ignored, no migration (no-backward-compat rule). `servers:` is dropped the next time the cursor is saved. |
| MCP `broadcast` hook | Removed. Only serve set `onEvent`. |

## Acceptance criteria

- [x] `xpo serve`, `xpo login` and `xpo logout` don't exist, and `xpo board` has no proxy path. *Evidence: the built binary reports `unknown command "serve"` and `unknown command "login"`. `ProxyURL` is gone from `Server`.*
- [x] No `remote.*` or `permissions` config keys, and no `XPO_REMOTE_*` env vars. *Evidence: `TestEnvBindings_CoverEveryField` passes without them.*
- [x] `internal/auth`, `RemoteTransport`, `Transport`, `ErrLocalOnly`, `live.Mirror`, `Headless`, `ProxyURL` and `/mcp` HTTP are gone. *Evidence: a grep over the repo, excluding `.xpo/` and the CHANGELOG, finds none in code.*
- [x] `GET /api/local/mode` returns `{git, root}` only, and the MCP stdio server always offers `start` and `merge`. *Evidence: `TestLocalMode_LocalBoard`, `TestLocalMode_ReportsGitRepo`, `TestWorkTools_OnLocalStdioServer`.*
- [x] The board's `--host` token sign-in still works. *Evidence: the `tokenguard_test.go` suite passes.*
- [x] `xpo whoami` prints the local identity. *Evidence: `TestWhoamiIdentity_*`; the binary printed `Identity: Nicolas Bettenburg <…>` inside and outside a project.*
- [x] Docs and the README don't describe remote mode, and the CHANGELOG has an entry. *Evidence: the `design.md` and README diffs, and the "Removed" entry under Unreleased.*
- [x] `make test` passes. *Evidence: exit 0, with 1225 frontend tests and all Go packages green.*
