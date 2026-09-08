# Walkthrough: Fix DefaultBranch local fallback

## What changed

`DefaultBranch()` in `git.go` now has a three-step fallback when `origin/HEAD` is unavailable:

1. `git config init.defaultBranch` — reads the repo-level (or global) init default
2. Probe `main` then `master` via `git rev-parse --verify` — checks if the well-known branch exists
3. Hardcoded `"main"` — final fallback

All commands use `-C hub` to target the hub root, avoiding false results when the working directory is in a worktree or on a feature branch.

## Why this approach

`git symbolic-ref HEAD` (the initial attempt) returns whichever branch the hub is currently on — in branch mode that's the feature branch, not the default. `init.defaultBranch` is the most reliable signal for what the repo was created with. The `main`/`master` probe handles repos where `init.defaultBranch` isn't explicitly set but the branch exists.

## Test changes

- `TestDefaultBranch_LocalBranchWithoutRemote` — sets `init.defaultBranch=trunk`, expects `"trunk"`
- `TestDefaultBranch_NoGitRepo` — non-git dir, expects `"main"` fallback
- `initTestRepo` — now calls `storage.ResetHubRoot()` for test isolation
