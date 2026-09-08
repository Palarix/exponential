# Walkthrough: Test coverage for checks.go and storage/root.go

## What was built

Two new test files:
- `internal/exponential/checks_test.go` — 7 tests for `CheckGitRepo`, `CheckGithubWorkflows`, `CheckGitHooks`
- `internal/storage/root_test.go` — 6 tests for `HubRoot`, `ResetHubRoot`, `XpoDir`

## Test inventory

### checks_test.go

| Test | What's verified |
|---|---|
| `TestCheckGitRepo_True` | Returns true in a git repo |
| `TestCheckGitRepo_False` | Returns false in a non-git directory |
| `TestCheckGithubWorkflows_True` | Detects `.github/workflows/*.yml` |
| `TestCheckGithubWorkflows_False` | Returns false when no workflows exist |
| `TestCheckGitHooks_None` | Fresh repo has no active hooks |
| `TestCheckGitHooks_Active` | Detects a real hook file |
| `TestCheckGitHooks_IgnoresSamples` | `.sample` files are excluded |

### root_test.go

| Test | What's verified |
|---|---|
| `TestHubRoot_InGitRepo` | Returns the repo root in a normal checkout |
| `TestHubRoot_NonGitDir` | Falls back to "." outside a git repo |
| `TestHubRoot_Caching` | Value is cached after first call (changing dirs doesn't change result) |
| `TestResetHubRoot_ClearsCachedValue` | After reset, re-discovers from current directory |
| `TestXpoDir` | Returns `<hub>/.xpo` |
| `TestHubRoot_InWorktree` | From a linked worktree, returns the hub (primary checkout) path |

## Key decisions

- **`HubRoot_InWorktree`** — creates a real linked worktree, `cd`s into it, and verifies `HubRoot` resolves back to the primary checkout. This is the critical path that merge, start, and every storage operation depend on.
- **Caching test** — verifies the `sync.Once` pattern works as intended, and that `ResetHubRoot` properly clears it for test isolation.
