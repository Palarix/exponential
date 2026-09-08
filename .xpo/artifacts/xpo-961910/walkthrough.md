## What was built

Added an optional `default_branch` field to `.xpo/config.yaml` and a global config accessor (`config.Get()` / `config.Set()`) so that `DefaultBranch()` can read config without parameter threading or signature changes.

When `default_branch` is set in config, it short-circuits git-based detection entirely — no `origin/HEAD` probe, no `init.defaultBranch` check, no branch scanning. This makes branch resolution reliable in CI, shallow clones, fresh checkouts, and repos using non-standard default branches.

## How the pieces fit together

### Global config accessor (`internal/config/config.go`)

Three new functions: `Set(cfg)`, `Get()`, `Reset()`. Guarded by `sync.RWMutex`. `Set` is called once at startup in `cmd/exponential/main.go` after `LoadConfig()` returns. `Get` returns the stored pointer (or nil before startup). `Reset` clears it (used by tests).

This follows the same singleton pattern as `storage.HubRoot()` — computed/set once, read from anywhere. The config is immutable after startup, so the lock is a correctness safeguard, not a contention point.

### `DefaultBranch()` config check (`internal/exponential/git.go`)

Added a three-line check at the top of `DefaultBranch()`:

```go
if cfg := config.Get(); cfg != nil && cfg.DefaultBranch != "" {
    return cfg.DefaultBranch
}
```

The nil check handles `xpo init` (no config loaded yet) and test scenarios where `Reset()` was called. When the config value is empty or absent, the existing 4-step git fallback chain runs unchanged. No signature change — all 14 call sites work without modification.

### `xpo init` auto-detection (`internal/exponential/setup.go`)

`InitProject` now calls `DefaultBranch()` and writes the result into the config template as `default_branch: <detected>`. New projects get the field pre-populated; existing projects without it continue to use git detection.

### Client/server semantics

In remote mode, `StartWork` is a hybrid: data operations (get/update issue) go to the remote server via HTTP, but git operations (branching, worktrees) run locally. `DefaultBranch()` always reads the **local** config, which is correct — the branching happens in the local repo.

The server's one `DefaultBranch()` call site (`resolveIssueBranch` in handlers.go) reads the server's own config, also correct — it computes diffs against the server's repo.

## Key decisions

- **Global accessor over variadic parameter** — the original spec proposed `DefaultBranch(configOverride ...string)` with all 14 callers updated. The user identified this as a design smell: the config is already effectively global (loaded once, passed identically everywhere), and `DefaultBranch()` already reads global state via `storage.HubRoot()`. A global accessor eliminates all caller changes.

- **`sync.RWMutex` over `atomic.Pointer`** — clearer intent for a set-once, read-many value. Either would work; the mutex reads more obviously in review.

- **No validation of the config value** — if `default_branch` names a nonexistent branch, git commands fail with clear errors. Validating would require git probing, defeating the purpose of an explicit override.

## Tests added

**`internal/config/config_test.go`:** `TestGlobalAccessor` (Set/Get/Reset lifecycle), `TestLoadConfigDefaultBranch` (YAML field parsed correctly), `TestLoadConfigDefaultBranchEmpty` (absent field stays empty).

**`internal/exponential/git_test.go`:** `TestDefaultBranch_ConfigOverride` (config wins over git), `TestDefaultBranch_EmptyConfigFallsThrough` (empty string falls through to git), `TestDefaultBranch_NilConfigFallsThrough` (nil config falls through).

**`internal/exponential/setup_test.go`:** 8 tests covering `InitProject` auto-detection across `main`, `master`, `develop`, `trunk`, remote with `origin/HEAD`, no-git-repo fallback, force re-init, and config override preservation.
