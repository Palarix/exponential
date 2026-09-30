# Spec: registry isAlive treats every process as dead on Windows

## What
Make `registry.isAlive(pid)` correct on both Unix and Windows so `List()` / `Register()` stop pruning live instances on Windows.

## Why
On Windows `(*os.Process).Signal` only supports `os.Kill`; `Signal(0)` returns `EWINDOWS`, so every entry is considered dead and deleted. The existing comment claims the opposite failure mode (false positives).

## How
- Remove `isAlive` from `registry.go` (and the `syscall` import).
- `internal/registry/isalive_unix.go` (`//go:build !windows`):
  - `os.FindProcess` + `Signal(syscall.Signal(0))`.
  - `nil` → alive; `errors.Is(err, syscall.EPERM)` → alive (exists, owned by another user); anything else → dead.
  - `pid <= 0` → dead (guards against `kill(0)`/`kill(-1)` semantics that signal process groups).
- `internal/registry/isalive_windows.go` (`//go:build windows`):
  - `windows.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))`; error → dead, except `ERROR_ACCESS_DENIED` → alive (process exists but is protected).
  - `defer windows.CloseHandle(h)`; `windows.GetExitCodeProcess`; alive iff code == `STILL_ACTIVE` (259).
  - `pid <= 0` → dead.
- Promote `golang.org/x/sys` to a direct dependency in `go.mod` (`go mod tidy`).
- Accurate doc comments on each implementation.
- `make lint`: add `GOOS=windows go vet ./...` after the native `go vet`.

## Tests (`internal/registry/isalive_test.go`, no build tag — runs on both platforms)
- `isAlive(os.Getpid())` → true.
- Start a child (`go` toolchain-free: re-exec the test binary with `-test.run=^$`), `Wait()` for it, then `isAlive(child.Pid)` → false.
- `isAlive(0)` and `isAlive(-1)` → false.

## Acceptance Criteria
- [ ] `isAlive` is split into Unix and Windows build-tagged files with correct semantics above.
- [ ] Misleading comment removed/replaced.
- [ ] Tests above pass on macOS; `GOOS=windows go test -c ./internal/registry` compiles; `GOOS=windows go vet ./...` passes.
- [ ] `make lint` includes the Windows vet step.
- [ ] `golang.org/x/sys` is a direct dependency.
- [ ] `make test` passes.

## Notes
- Reaped-child PID reuse is theoretically possible between `Wait()` and the check but negligible in a test.
- Windows runtime behaviour can't be exercised locally (no Windows CI); verified by compile + vet only.
