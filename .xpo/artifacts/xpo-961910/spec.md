## What

Add an optional `default_branch` field to `.xpo/config.yaml` that explicitly sets the base branch for all branching, merging, and diff operations. Also introduce a global config accessor (`config.Get()` / `config.Set()`) so package-level functions like `DefaultBranch()` can read config without needing `Client` or parameter threading.

## Why

**`default_branch` config:** The current `DefaultBranch()` function relies entirely on git state (`origin/HEAD`, `init.defaultBranch`, probing `main`/`master`). This breaks in fresh clones, CI, non-standard branch names, and worktree setups. An explicit config field is authoritative, committed to the repo, and always wins.

**Global config accessor:** `DefaultBranch()` is a package-level function in `exponential` with no access to `Config`. The config is loaded once in `main.go` and stored in a package-level var, then passed to `NewClient(cfg)` per command. But package-level helpers like `DefaultBranch()` can't reach it. The `storage` package already proves the cached singleton pattern with `HubRoot()`. Giving `config` the same pattern eliminates the friction between package-level functions and the config they need.

## Acceptance Criteria

1. `config.Set(cfg)` / `config.Get()` exist as a global accessor pair; `config.Reset()` exists for tests
2. `config.Set(cfg)` is called in `main.go` after `LoadConfig()`
3. A `DefaultBranch` string field exists in the Config struct
4. `DefaultBranch()` stays a zero-arg package-level function — checks `config.Get().DefaultBranch` first, then the existing git fallback chain
5. `xpo init` auto-detects the default branch and writes `default_branch` to the config template
6. Existing tests pass; new tests cover the config override path and the global accessor
7. `XPO_DEFAULT_BRANCH` env var works via Viper's automatic env binding

## Flow

### Step 1: Add global accessor to config package (`internal/config/config.go`)

Add a thread-safe global config accessor using `sync.RWMutex`:

```go
var (
    global   *Config
    globalMu sync.RWMutex
)

func Set(cfg *Config) {
    globalMu.Lock()
    global = cfg
    globalMu.Unlock()
}

func Get() *Config {
    globalMu.RLock()
    defer globalMu.RUnlock()
    return global
}

func Reset() {
    globalMu.Lock()
    global = nil
    globalMu.Unlock()
}
```

`RWMutex` over `sync.Once` because the config is loaded externally (not computed on first access) and may be set again (e.g., `init.go` reloads after creating `.xpo/`). In practice, `Set` is called once at startup before any concurrent reads, so the lock is a correctness safeguard, not a hot path.

### Step 2: Add `DefaultBranch` field to Config struct (`internal/config/config.go`)

```go
DefaultBranch string `mapstructure:"default_branch" yaml:"default_branch,omitempty"`
```

After the `WorktreeSetup` field. No Viper default — empty string means "use git detection".

### Step 3: Wire `config.Set()` in CLI entrypoint (`cmd/exponential/main.go`)

In `PersistentPreRunE`, after `config.LoadConfig()` returns, add:

```go
config.Set(cfg)
```

Also in `cmd/exponential/init.go` after its `LoadConfig()` call.

### Step 4: Update `DefaultBranch()` (`internal/exponential/git.go`)

Add a config check as the **first** step, before any git commands:

```go
func DefaultBranch() string {
    if cfg := config.Get(); cfg != nil && cfg.DefaultBranch != "" {
        return cfg.DefaultBranch
    }
    // ... existing git fallback chain unchanged ...
}
```

No signature change. No caller changes. Zero blast radius.

### Step 5: Update `xpo init` (`internal/exponential/setup.go`)

After deriving `prefix`, auto-detect the default branch and include it in the config template:

```go
detectedBranch := DefaultBranch()
```

Add `default_branch: %s` to the `configContent` format string with `detectedBranch` as the value.

### Step 6: Tests

**`internal/config/config_test.go`:**
- `TestGlobalAccessor` — `Set` then `Get` returns the same config; `Reset` clears it; `Get` returns nil before `Set`

**`internal/exponential/git_test.go`:**
- `TestDefaultBranch_ConfigOverride` — set config with `DefaultBranch: "custom"`, assert `DefaultBranch()` returns `"custom"` without touching git
- `TestDefaultBranch_EmptyConfigFallsThrough` — set config with empty `DefaultBranch`, assert git detection still works
- Clean up with `config.Reset()` in each test

## Decisions

- **`sync.RWMutex` over `atomic.Pointer`** — clearer intent, familiar Go pattern, and `RWMutex` is negligible overhead for a startup-set, read-many value. `atomic.Pointer` would also work but reads less obviously in code review.
- **`Get()` returns `*Config` (may be nil)** — callers must nil-check. This is intentional: `DefaultBranch()` is called during `xpo init` before any config exists, and it must fall through to git detection in that case.
- **No validation of the config value** — if a user sets `default_branch: "nonexistent"`, git operations will fail with clear git errors. Validating would require running git commands, defeating the purpose of a config override that works without git state.
- **`omitempty` on YAML tag** — existing configs won't grow a `default_branch: ""` when re-serialized.
- **`Set` called in both `main.go` and `init.go`** — `init.go` reloads config after creating `.xpo/`, so the global needs to be updated. The second `Set` replaces the first, which is correct.

## Edge Cases

- **LOW:** `config.Get()` returns nil during `xpo init` before config exists — the nil check in `DefaultBranch()` handles this cleanly; falls through to git detection.
- **LOW:** `XPO_DEFAULT_BRANCH` env var — works automatically via Viper's `AutomaticEnv()` with the `XPO` prefix.
- **LOW:** Config value with whitespace — YAML parser handles this.

## Assumptions

- The `init.go` command's `LoadConfig()` call (line 40) is the only place outside `main.go` that loads config in production. The MCP server and HTTP server receive config via constructor, not by calling `LoadConfig()`.
- `DefaultBranch()` is never called from multiple goroutines simultaneously in a way that would race with `config.Set()`. In practice, `Set` completes before any command runs.
