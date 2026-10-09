# Walkthrough: Per-run access token for `xpo board` on a non-loopback `--host`

## What was built and why
xpo-809ad0 made the board listen on loopback by default and added a `Host`-header guard against DNS rebinding. Even after that, `xpo board --host 192.168.x.y` or `--host 0.0.0.0` let anyone who could reach the port read and modify issues, and in remote/proxy mode they acted with the user's bearer token. On a wildcard bind the `Host` guard is turned off, so DNS rebinding was possible again.

This change adds a Jupyter-style access token. It is generated per run and is only active when the bind address is not loopback. Local use (the default) has no extra step.

## How the pieces fit together

### 1. `internal/server/tokenguard.go` (new)
- **`GenerateAccessToken()`** returns 32 bytes from `crypto/rand`, hex-encoded (64 chars).
- **`Server.AccessToken`** is a new field on `Server`. When it is empty the guard returns `next` unchanged, so the loopback path is identical to before.
- **`AccessURL()`** returns `URL()` with `/?token=<token>` appended when a token is set. The CLI prints this URL and opens the browser at it.
- **`tokenGuard(next)`** is wired in `ServeOn` as `hostGuard(tokenGuard(mux))`. The `Host` check runs first, then the token check. It decides each request in this order:
  1. `/healthz`: pass through, so health probes keep working.
  2. `GET /auth`: render the token form. `POST /auth`: check the token. A valid one gets the cookie and a `303` to `/`; an invalid one re-renders the form with "Invalid token" and returns `401`.
  3. Valid session cookie: pass through.
  4. `GET` with a valid `?token=`: set the cookie and `302` to the same path and query with `token` removed, so the token doesn't stay in the URL bar or history.
  5. Anything else: `/api/*`, `/mcp` and every non-GET request get `401 {"error":"unauthorized"}`. A page load gets `302 /auth`.
- All token comparisons use `subtle.ConstantTimeCompare`.
- The `/auth` routes live in the middleware, not the mux. That means they never clash with serve mode's `POST /auth/challenge|verify` or the proxy's `POST /auth/{path...}`. The page is a self-contained `html/template` with inline CSS that uses CSS system colors, so it follows the browser's light or dark mode without depending on the React bundle.

### 2. The cookie
- **Name: `exponential-<first 8 hex of sha256(token)>`.** Browsers scope cookies by host, not port, so two network-exposed boards on one machine need different names. The first version used the port. In review we dropped that, because ports go to whichever project starts its board first: 8080 can be project A today and project B tomorrow. A hash of the token is unique to each run and doesn't reveal the token. The `xpo_board_` prefix was also dropped because it read like a board identifier.
- **Attributes:** `HttpOnly`, `Path=/`, `SameSite=Lax`. It is a session cookie with no `Expires`, and it has no `Secure` flag because the board serves plain HTTP. The value is the token itself. Because the token is per run, a cookie from a previous run fails the comparison, and the user simply authenticates again.
- **Why Lax, not Strict** (the issue said Strict; changed with the user's agreement): a Strict cookie is not sent when the user follows a link to the board from Slack, email or a terminal, so an authenticated user would land on `/auth`. Lax still withholds the cookie on cross-site POST/PUT/DELETE, which covers every write. Cross-origin reads are already blocked by the browser's same-origin policy. Jupyter makes the same choice.
- **DNS rebinding on a wildcard bind:** the attacker's page is served from the attacker's hostname, so the browser never attaches this cookie to those requests. The token guard therefore covers the case where the `Host` guard is turned off.

### 3. `cmd/exponential/board.go`
On a non-loopback `--host`, the CLI generates the token, sets `srv.AccessToken`, prints `Access token required. Open: <AccessURL>` and opens the browser at that URL. The warning now says the traffic is plain HTTP and that the token is required. It still recommends `ssh -L`. The `--host` help and the long description are updated to match.

### 4. `web/src/api/client.ts`
A new `fetchOk()` helper does the fetch and the `!res.ok → ApiError` check. On a `401` it also calls `window.location.assign('/auth')`. `request()` and the two raw-text diff fetchers (`fetchIssueDiff`, `fetchCommitDiff`) now go through it, so no `fetch` in the client skips it. SSE needs no change: `EventSource` sends same-origin cookies, and if the cookie is missing, a page load is redirected before SSE ever starts.

## Non-obvious details
- With a wildcard bind, `URL()` prints `localhost`, so `AccessURL()` does too. Someone connecting from another machine replaces the host with the machine's address; the token part is what matters.
- `cookieName()` hashes the token on every request. The cost is negligible, and it means there is no extra field that could drift out of sync with `AccessToken`.
- There is no rate limiting on `POST /auth`. A 256-bit token cannot be brute-forced.

## Acceptance Criteria
- [x] Loopback `--host` (default): no token and no cookie check. Evidence: `TestTokenGuardDisabledWithoutToken`. The existing `hostguard_test.go` and handler tests pass unchanged.
- [x] Non-loopback `--host`: the console prints `http://<host>:<port>/?token=<64 hex>`, and the browser opens it. Evidence: `TestGenerateAccessToken` and `TestAccessURL`. A live run with `--host 0.0.0.0` printed `Access token required. Open: http://localhost:18931/?token=0f27…a134`.
- [x] `/?token=<valid>` sets an `HttpOnly` cookie and redirects without `token`, keeping other query params. Evidence: `TestTokenGuardQueryExchange` (`/board?view=list&token=…` → `/board?view=list`). Live: `302 -> /board?x=1`, and the cookie jar shows `#HttpOnly_localhost exponential-…`.
- [x] Without the cookie: `/api/issues` → 401 JSON, `/` and `/board/...` → 302 `/auth`, `/healthz` → 200. Evidence: `TestTokenGuardWithoutCookie`. Live curl gave `401 {"error":"unauthorized"}`, `302 -> /auth` and `200`.
- [x] `/auth` form: the right token sets the cookie and redirects to `/`; a wrong one returns 401 with an error. Evidence: `TestAuthPage` and `TestAuthSubmit`. Live: `303 -> /` and `401`.
- [x] A wrong `?token=` or cookie is rejected, and comparisons are constant-time. Evidence: `TestTokenGuardCookie` (a wrong value and another run's cookie name both get 401) and the wrong-token case in `TestTokenGuardQueryExchange`. `validToken` uses `subtle.ConstantTimeCompare`.
- [x] The frontend redirects to `/auth` on any API 401. Evidence: `web/src/api/client.test.ts` covers `request()`, `fetchIssueDiff` and `fetchCommitDiff`, and checks that a 500 does not redirect.
- [x] A wildcard bind (`0.0.0.0`) also requires the token. Evidence: the live smoke test above ran with `--host 0.0.0.0`.
- [x] `make test` passes (exit 0): `go vet`, eslint, vitest (36 files, 540 tests) and `go test ./...`.
