# Walkthrough: registry isAlive treats every process as dead on Windows

## What was wrong
`internal/registry` keeps a shared `instances.json` of running `xpo board` processes and prunes entries whose PID is no longer alive — on every `List()` and `Register()`. Liveness was checked with `os.FindProcess(pid).Signal(syscall.Signal(0))`. That's the classic Unix idiom, but on Windows Go's `Process.Signal` only implements `os.Kill`; anything else returns `EWINDOWS`. So on Windows every entry looked dead: `List()` returned nothing and wiped the file, and `Register()` evicted every peer. The old comment warned about the opposite failure (false positives), which hid the real problem.

## What changed
`isAlive` is now platform-specific via build constraints:

- **`internal/registry/isalive_unix.go`** (`//go:build !windows`) — still `Signal(0)`, but:
  - `EPERM` counts as alive. `kill(pid, 0)` returns EPERM when the process exists but belongs to another user; previously that would have pruned a live entry.
  - `pid <= 0` is rejected up front. `kill(0, …)` targets the caller's process group and `kill(-1, …)` every process we can signal, so a corrupt/zero PID in the registry would otherwise look alive forever.
- **`internal/registry/isalive_windows.go`** (`//go:build windows`) — `windows.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` then `GetExitCodeProcess`; alive iff the code is `STILL_ACTIVE` (259). The exit-code check matters because a process handle can still be opened for a process that has exited but whose handle is held elsewhere. `ERROR_ACCESS_DENIED` from `OpenProcess` counts as alive (the process exists but is protected) — the Windows analogue of EPERM. Same `pid <= 0` guard.
- **`registry.go`** — the old `isAlive` and its `syscall` import are removed; callers are unchanged.
- **`go.mod`** — `golang.org/x/sys` moved from indirect to the direct `require` block. This was a targeted edit: `go mod tidy` also reshuffled unrelated deps (`huh`, `x/exp`, `go-udiff`) and that churn was deliberately left out of this change.
- **`Makefile`** — `lint` now runs `GOOS=windows go vet ./...` after the native vet. It won't catch runtime behaviour like this bug, but it makes Windows-only files type-check on every `make test`, which they otherwise never would on a Mac/Linux dev box.

## Tests
`internal/registry/isalive_test.go` has no build tag, so the same assertions apply to both implementations:
- current PID → alive
- an exited child → dead. The child is the test binary itself re-exec'd with `-test.run=^$`, so it exits immediately on any OS without depending on `true`/`cmd.exe`.
- PIDs `0` and `-1` → dead

## Acceptance Criteria
- [x] `isAlive` split into Unix and Windows build-tagged files with the specified semantics — `isalive_unix.go`, `isalive_windows.go`
- [x] Misleading comment removed/replaced — each implementation documents its own semantics
- [x] Tests pass on macOS; `GOOS=windows go test -c ./internal/registry` compiles; `GOOS=windows go vet ./...` passes — all run clean
- [x] `make lint` includes the Windows vet step — Makefile `lint` target
- [x] `golang.org/x/sys` is a direct dependency — go.mod direct `require` block
- [x] `make test` passes — exit 0 (369 frontend tests + Go suite)

## Caveat
Windows runtime behaviour was verified by compile + vet only; no Windows host was available. Running `go test ./internal/registry/` on Windows would confirm it end-to-end.
