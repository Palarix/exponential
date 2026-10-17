# Remove remote mode: xpo is local-only

## What

Delete remote mode: `xpo serve`, `login`/`logout`, server auth, `RemoteTransport`, board proxy mode with its live mirror, the `remote.*` config, and the xpo-a60065 guards. Local mode works as it does today.

## Why

Remote mode projects branch state from the server's checkout, and start and merge run there too (xpo-5d307a). Fixing that means moving git state over the wire. We decided xpo is local-only: issue data travels with the code through git. Team mode (xpo-8757a9) will be designed from scratch later.

## Keep (looks remote, but local mode uses it)

- The board's `--host` access token and the `/auth` sign-in page (`tokenguard.go`, `hostguard.go`), plus the frontend's 401 redirect to `/auth`. Only `/auth/challenge` and `/auth/verify` go.
- `GET /api/local/mode`. It still returns `git` and `root` (xpo-e627e1); only the `remote` field goes.
- `chat.Profile.CanWork`, which means "is a git repo".
- `user.yaml` and its inbox read cursor (`ProjectState`). Only the `servers:` branch goes.
- `xpo demo`. It doesn't depend on serve.

## Remove

### Commands and server
1. `cmd/exponential/serve.go`, `login.go`, `logout.go`, and their entries in the `main.go` exception lists.
2. `Dockerfile`, `docker-compose.yml` and the `make docker` target. They only run `xpo serve`.
3. In `internal/server`:
   - The `Headless` field and its guards.
   - The `errGitActionsOnServe` constant and its 501 guards.
   - The `/mcp` mount and the `MCPHandler` field.
   - The auth fields (`NonceStore`, `AuthorizedKeys`, `SigningKey`, `VerifyKey`) and the `handle`/`handleCap` auth wrappers. Routes use `mux.HandleFunc` directly.
   - `requireCapability` and its call sites.
   - `handleAuthChallenge` and `handleAuthVerify`.
   - `logging.go`, which only serve uses.
   - `/mcp` in `isBackendPath` and in the token guard.
4. `internal/auth/` (the whole package).
5. In `internal/mcpserver`: `RegisterTools(…, httpReq, …)`, `httpUserOverride`, and the `registerWorkTools` gate. `start` and `merge` are always registered; the split folds back into `register`.

### Board proxy and mirror
6. `Server.ProxyURL` and `ProxyToken`, the reverse-proxy branch in `router.go`, `startMirror`, `remoteTopics`, the proxy branch in `live.go`, and `internal/live/mirror.go`.
7. In `cmd/exponential/board.go`: the proxy setup and `chat.MCPTarget.RemoteURL`/`RemoteToken`. The chat's HTTP MCP branch in `args.go` and `codex.go` (`codexTokenEnv`, the bearer arguments and the token parameter) go too.
8. `chat.RunnerConfig.CanWork`, `ErrWorkUnavailable`, and their 501 handling in `server/chat.go`.

### Transport
9. Delete `remote_transport.go`, the `Transport` interface (`transport.go`) and `ErrLocalOnly`. `Client` holds a `*LocalTransport`, and its `Transport` and `local` fields become one field. Callers that use `c.Transport` (`board.go`, `start.go`, `merge.go`, `review.go`) switch to it. Every `c.local == nil` guard goes (`merge`, `maintenance`, `review`, `drafts`, `drive`).
10. Draft owner scoping (the `owner` parameters, `ownedDraft`, and the `auth.UserFromContext` call sites). Without auth, the owner is always empty.

### Config
11. `Config.Remote`/`RemoteConfig` (and with them the env vars `XPO_REMOTE_URL` and `XPO_REMOTE_TOKEN`), `ResolveRemote`, `ServerCredential`, `UserConfig.Servers`, `ReadRemoteURL`, `IsLocalProjectConfig`, `SetRemoteURL` and `hostFromURL`. `GetInboxLastRead` and `SetInboxLastRead` drop their `remoteURL` parameter.
12. `Config.Permissions`/`PermissionsConfig` and the role and capability helpers. They only mattered with auth.

### whoami
13. Keep `xpo whoami` as a local identity command. It prints the identity from `Client.GetUser()` (override, then `cfg.User`, then git config) and drops `Mode:`, the JWT decoding and the permissions output.

### Frontend
14. Remove `LocalMode.remote` and `useGitActions`, and the `gitActions` prop on `IssueDetail`/`PropertySidebar`. Start Work, Review Changes and MergeView render under their other existing conditions.

### Docs
15. In `docs/design.md`: the distributed mode text, the diagram's remote half, transport abstraction (rewrite it short, with correct paths), the `authorized_keys` and `server.key` rows, proxy mode, authentication, the permissions and remote config sections, and MCP "(stdio / HTTP)" becomes stdio.
16. In the README: the Docker section, the distributed mode section, and the `XPO_REMOTE_URL` example (use another key). Add a CHANGELOG entry.
17. Run `go mod tidy` (this drops `golang.org/x/crypto` as a direct dependency).

### Tests
Delete the remote, auth, serve, proxy and mirror tests that the mapping found. Rework the ones that stay:
- `remote_guards_test.go`: keep the local `/api/local/mode` and start-work tests, minus `remote`.
- `env_test.go`: use another sample key.
- The inbox cursor tests.
- `PropertySidebar.test.tsx`: drop the remote cases.
- Tests that build `&Client{Transport:…, local:…}`.

## Edge cases

- An old `config.yaml` with `remote:` or `permissions:`, or an old `user.yaml` with `servers:`: unknown keys are ignored. No migration (no-backward-compat rule).
- `useGitActions` also hid git buttons until the mode loaded. With it gone, the buttons render at once, under their other existing conditions.

## Acceptance criteria

- [ ] `xpo serve`, `xpo login` and `xpo logout` don't exist, and `xpo board` has no proxy path.
- [ ] No `remote.*` or `permissions` config keys, and no `XPO_REMOTE_*` env vars.
- [ ] `internal/auth`, `RemoteTransport`, `Transport`, `ErrLocalOnly`, `live.Mirror`, `Headless`, `ProxyURL` and `/mcp` HTTP are gone (grep is clean).
- [ ] `GET /api/local/mode` returns `{git, root}` only. The MCP stdio server always offers `start` and `merge`.
- [ ] The board's `--host` token sign-in still works.
- [ ] `xpo whoami` prints the local identity.
- [ ] Docs and the README don't describe remote mode. The CHANGELOG has an entry.
- [ ] `make test` passes.

## Decisions

| Question | Decision |
|---|---|
| `xpo whoami` | Keep it as a local identity command. |
| `permissions` config | Remove it. |
| Draft owner scoping | Remove it. |
| Docker files | Remove them. |
