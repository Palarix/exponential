# Spec: Per-run access token for `xpo board` on a non-loopback `--host`

## What
When `xpo board --host <addr>` binds a non-loopback address, which includes wildcards like `0.0.0.0` and `::`, every request except `/healthz` and `/auth` needs a session cookie. The cookie comes from a per-run access token in the style of Jupyter. On loopback (the default) nothing changes.

## Why
xpo-809ad0 made loopback the default and added a `Host`-header guard. Even so, `--host <lan-ip>` still lets anyone who can reach the port read and modify issues. In remote/proxy mode they also act with the user's bearer token. On a wildcard bind the `Host` guard is turned off, so the board has no DNS-rebinding protection either.

## How

### Server (`internal/server/tokenguard.go`, new)
1. **`Server.AccessToken string`**: when it is empty, the guard is a no-op (the loopback default).
2. **`GenerateAccessToken() (string, error)`**: 32 bytes from `crypto/rand`, hex-encoded (64 chars). Called once in `runBoard` when `!IsLoopbackHost(boardHost)`.
3. **`s.tokenGuard(next)`**: middleware wrapped inside `hostGuard` in `ServeOn`, so the host guard runs first and the token guard second.
   - **Public paths:** `GET /healthz`, `GET /auth` and `POST /auth`. These pass through.
   - **Token exchange:** a `GET` whose `?token=` matches (`subtle.ConstantTimeCompare`) sets the cookie and sends a `302` to the same path and query with `token` removed. A wrong `?token=` gets the same treatment as having no cookie.
   - **Valid cookie** (constant-time compare against `AccessToken`): pass through.
   - **No valid cookie:**
     - Requests for `/api/*` or `/mcp`, or any non-GET request: `401` with JSON `{"error":"unauthorized"}`.
     - Any other `GET` (a page or asset load): `302` to `/auth`.
4. **Cookie:** `Name = "exponential-<id>"`, where `<id>` is the first 8 hex characters of `sha256(token)`. Cookies are scoped by host, not port, so each run gets its own cookie name and boards on one host never overwrite each other's cookie, whichever port each one gets. `Value = token`, `Path=/`, `HttpOnly`, `SameSite=Lax` (see Decisions), no `Expires` (session cookie), no `Secure` (plain HTTP).
5. **`GET /auth`:** a small HTML page written in Go with `html/template`. It is self-contained: inline CSS and no JS or React dependency. It has one password field, `token`, and a submit button, and the form POSTs to `/auth`.
6. **`POST /auth`:** with a valid token it sets the cookie and sends a `303` to `/`. With an invalid token it re-renders the form with "Invalid token" and returns `401`. There is no rate limiting because a 256-bit token cannot be brute-forced.
7. The `/auth` routes are registered by the guard, not the mux. That avoids any clash with serve mode's `POST /auth/challenge|verify` and the proxy's `POST /auth/{path...}` (exact `/auth` ≠ `/auth/...`).

### CLI (`cmd/exponential/board.go`)
- If the host is non-loopback, generate the token, set `srv.AccessToken`, and after `Bind()` log `Access token required. Open: <srv.URL()>/?token=<token>`. The auto-opened browser uses that same URL.
- The warning text changes from "the board is unauthenticated" to say that a token is required, the traffic is plain HTTP, and `ssh -L` is still preferred. The `--host` flag help and the long description get the same update.

### Frontend (`web/src/api/client.ts`)
- One helper, `handleUnauthorized(res)`: on `401` it calls `window.location.assign("/auth")`. `request()` calls it, and so do the raw `fetch` sites in `client.ts` (`fetchIssueDiff`, `fetchCommitDiff`, and any others).
- No SSE change: `EventSource` sends cookies on the same origin.

## Acceptance Criteria
- [ ] Loopback `--host` (default): no token, no cookie check, and behaviour is identical to today (existing tests pass unchanged).
- [ ] Non-loopback `--host`: the console prints `http://<host>:<port>/?token=<64 hex>`, and the browser opens that URL.
- [ ] Visiting `/?token=<valid>` sets an `HttpOnly` cookie and redirects to `/` without `token` in the URL. Other query params are kept.
- [ ] Without a cookie: `/api/issues` → 401 JSON; `/` and `/board/...` → 302 `/auth`; `/healthz` → 200.
- [ ] `/auth` form: the correct token sets the cookie and redirects to `/`, and a wrong token returns 401 and shows an error.
- [ ] A wrong `?token=` or cookie is rejected. Comparisons are constant-time.
- [ ] The frontend redirects to `/auth` when any API call returns 401.
- [ ] A wildcard bind (`0.0.0.0`) also requires the token.
- [ ] `make test` passes. The Go tests cover the guard table, the exchange redirect, and the auth form, and a vitest test covers the 401 redirect.

## Out of scope
TLS, token persistence across restarts, logout, and `xpo serve` (it already has SSH-key auth).

## Decisions
1. **`SameSite=Lax`, not `Strict`** (confirmed by the user). With Strict, the cookie is not sent when the user follows a board link from Slack, email or the terminal, so an authenticated user would land on `/auth`. Lax still blocks the cookie on cross-site POST/PUT/DELETE, which covers every write. Cross-origin reads are already blocked by the same-origin policy. Jupyter makes the same choice.
2. **Cookie name is a per-run ID derived from the token** (revised in review; it was the port). Ports are assigned in whatever order projects start the board, so 8080 might be project A today and project B tomorrow. That makes the port a poor key. `sha256(token)[:8]` is unique per run and reveals nothing about the token. Session cookies left over from earlier runs stay until the browser closes, and the server rejects them.
