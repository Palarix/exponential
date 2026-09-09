# Walkthrough: Isolate tests from global git config

## Problem

`DefaultBranch()` in `internal/exponential/git.go` probes `git config init.defaultBranch`,
which reads the host's global git config. On machines where this is set to `master` (or
anything other than `main`), the function returns the wrong branch name even though test
repos are explicitly initialized with `-b main`. This caused ~25 test failures across
`internal/exponential` and `internal/mcpserver`.

## Fix

Added a `TestMain` function to each affected package (`internal/exponential`,
`internal/mcpserver`) that sets `GIT_CONFIG_GLOBAL=/dev/null` and
`GIT_CONFIG_SYSTEM=/dev/null` before running tests. This prevents `git` subprocesses from
reading the host's global or system config, so `git config init.defaultBranch` only finds
values explicitly set in the test repo's local config.

Also fixed a bare `git init` call in `internal/storage/collapse_test.go` that was missing
the `-b main` flag (all other test helpers already had it).

## Key decisions

- **`TestMain` over per-helper `t.Setenv`**: A single `TestMain` per package covers all
  tests — current and future — without requiring every helper to remember the isolation.
  `os.Setenv` (not `t.Setenv`) is correct here because `TestMain` runs before any `*testing.T`
  exists.
- **`/dev/null` over an empty temp file**: `/dev/null` is a valid empty config file on all
  Unix systems and needs no cleanup.