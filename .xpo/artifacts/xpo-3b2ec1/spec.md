# Spec: Isolate tests from global git config

## What
Tests fail on machines where `git config --global init.defaultBranch` is not `main`.

## Why
`DefaultBranch()` in `git.go` queries `git config init.defaultBranch`, which reads the
**global** git config. On machines with `init.defaultBranch=master`, this returns `master`
even though test repos are initialized with `-b main`.

## How
Add a `TestMain` function to each affected package that sets `GIT_CONFIG_GLOBAL=/dev/null`
and `GIT_CONFIG_SYSTEM=/dev/null` before running tests. This isolates all tests from the
host machine's git configuration.

### Files to change
1. **`internal/exponential/main_test.go`** — new file with `TestMain` setting env vars
2. **`internal/mcpserver/main_test.go`** — new file with `TestMain` setting env vars
3. **`internal/storage/collapse_test.go`** — fix bare `git init` to use `-b main`

## Acceptance criteria
- `make test` passes on a machine with `init.defaultBranch=master`
- `make test` passes on a machine with `init.defaultBranch=main`
- No test relies on the host's global git configuration