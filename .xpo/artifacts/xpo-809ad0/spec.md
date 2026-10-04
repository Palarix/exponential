# Spec: xpo board binds loopback only

## What
`xpo board` listens on `127.0.0.1` by default instead of all interfaces, gains a `--host` flag to override, rejects requests with a non-loopback `Host` header, and the docs describe `ssh -L` as the supported way to view a board from another machine.

## Why
The board has no authentication. Binding `":<port>"` exposes full read/write access (and, in remote mode, the user's bearer token via the proxy) to every machine that can reach the port. A missing `Host` check also allows DNS-rebinding attacks from any web page the user visits.

## How
1. **`registry.FindFreePort(host, startPort, maxAttempts)`** — takes a host and listens on `net.JoinHostPort(host, port)`. The only caller is `Server.Bind()`.
2. **`Server.Host string`** — new field, default `127.0.0.1`. `Bind()` passes it through.
3. **`xpo board --host <addr>`** — default `127.0.0.1`. If the address is not loopback, log a warning: the board is unauthenticated and reachable by anyone on the network; prefer `ssh -L`.
4. **Host-header guard** — middleware in `ServeOn` wrapping the mux. Allowed hostnames (port ignored): `localhost`, `127.0.0.1`, `::1`, plus the `--host` value when it is a concrete host. When `--host` is a wildcard (`0.0.0.0`, `::`, empty), the guard is disabled, because the user has explicitly asked for network exposure and we cannot know the names clients will use. Rejected requests get `403` with a JSON error. Applies to the board only (`ServeOn`); `xpo serve` is untouched.
5. **Vite dev proxy** — change `'/api': 'http://localhost:8080'` to `http://127.0.0.1:8080`. Node 17+ resolves `localhost` to `::1` first and would otherwise get connection refused. The Host header that Vite forwards (`localhost:5173`) passes the guard.
6. **Log/browser URL** — keep `http://localhost:<port>` when the host is loopback; otherwise print the actual host.
7. **Docs** — README section "Viewing a board on another machine": `ssh -L 8080:127.0.0.1:8080 user@host`, then open `http://localhost:8080`. Mention `--host` as an unsafe escape hatch.

## Acceptance Criteria
- `xpo board` binds only `127.0.0.1` (verified by a test on the listener address).
- `--host` overrides the bind address; a non-loopback value logs a warning.
- Requests with `Host: evil.example:8080` get 403; `localhost:<any port>`, `127.0.0.1`, and `[::1]` pass.
- `xpo board --dev` with the Vite dev server still works.
- README documents `ssh -L`.
- `make test` passes.

## Out of scope
- Authentication for the board (SSH-key/cookie session): a possible later feature.
- `xpo serve` bind defaults.
