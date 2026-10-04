# Walkthrough: xpo board binds loopback only

## The problem
`registry.FindFreePort` listened on `":<port>"`. In Go, an address with no host part binds every interface (IPv4 and IPv6). The board has no authentication, so anyone who could reach the port got the full read/write API. In remote mode it was worse: the board reverse-proxies `/api/*` and `/auth/*` to the remote server and injects the user's bearer token, so a network peer could act as the user remotely. The log line and browser URL said `localhost`, and the README even claimed "binds to 127.0.0.1 only", which is why nobody noticed.

## What changed

### Bind address (`internal/registry/registry.go`, `internal/server/server.go`)
`FindFreePort(host, startPort, maxAttempts)` now takes a host and uses `net.JoinHostPort`, which brackets IPv6 literals correctly. `Server` has a `Host` field that `NewServer` sets to `server.DefaultHost` (`127.0.0.1`). `Bind()` passes it through. `127.0.0.1` rather than `localhost`: binding a name would resolve to one address family unpredictably.

### `--host` flag (`cmd/exponential/board.go`)
`xpo board --host <addr>` overrides the default. For any non-loopback value it logs a warning that the board is unauthenticated and suggests the matching `ssh -L` command. The `--help` text explains the same thing.

### Host-header guard (`internal/server/hostguard.go`)
Binding loopback doesn't stop **DNS rebinding**: a malicious page can point its own hostname at 127.0.0.1, and the browser will then send requests to the board as same-origin. These requests carry the attacker's hostname in the `Host` header, so `hostGuard` rejects with 403 any request whose host (port ignored, IPv6 brackets stripped, case-insensitive) is not a loopback name (`localhost` or a loopback IP) or the exact `--host` value.

The guard is installed in `ServeOn`, which only the board uses. `xpo serve` has its own JWT auth and is untouched.

**Wildcard binds disable the guard.** With `--host 0.0.0.0` or `::`, clients reach the board via LAN IPs or hostnames we can't predict, and the user has explicitly asked for network exposure. That decision was confirmed with the user. xpo-897f35 (per-run access token, BACKLOG) would close this gap with a cookie that a rebinding origin never has.

### `Server.URL()`
Builds the log and browser URL: `localhost` for loopback or wildcard binds, otherwise the actual host (bracketed for IPv6).

### Vite dev proxy (`web/vite.config.ts`)
Now targets `http://127.0.0.1:8080`. Node 17+ resolves `localhost` to `::1` first, which would hit a closed port now that the board listens on IPv4 loopback only. Vite forwards `Host: localhost:5173`, which passes the guard.

### Docs
README: the loopback claim is now true, plus an `ssh -L 8080:127.0.0.1:8080 user@host` recipe as the supported remote-viewing path and a warning on `--host`.

## Tests
`internal/server/hostguard_test.go`:
- `Bind()` with defaults yields a loopback listener.
- A table of `Host` cases: loopback names and ports pass; foreign names, LAN IPs and `localhost.evil.example` are rejected; a concrete `--host` is allowed; wildcard binds pass everything.
- `URL()` formatting, including IPv6.

Manual smoke test: `ss` showed `127.0.0.1:<port>` only; a bad `Host` returned 403; the LAN IP was refused; `--host 0.0.0.0` was reachable and printed the warning. `--dev` with Vite was not exercised live.
