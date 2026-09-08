# Walkthrough: start.go worktree edge case tests

## What was built

4 new tests in `start_test.go` plus a shared `setupWorktreeTestRepo` helper.

## Test inventory

| Test | Code path |
|---|---|
| `TestStartWork_Worktree_ExistingBranch` | Branch already exists, `WorktreeAddExisting` path (line 78) |
| `TestStartWork_Worktree_ForceRemovesExisting` | Worktree exists, `force: true` removes and recreates (lines 68-74) |
| `TestStartWork_Worktree_SetupHook` | `WorktreeSetup` config runs in worktree dir (lines 100-109) |
| `TestStartWork_Worktree_SetupHookFails` | Hook exits non-zero, warning message but no error (line 105) |

## Key decisions

- **`setupWorktreeTestRepo`** extracts the common repo+config setup that was inline in `TestStartWork_Worktree_CreatesWorktree`, using `filepath.EvalSymlinks` for macOS compatibility.
- **Setup hook tests** use `touch .setup-ran` and `exit 1` as minimal shell commands to verify execution and failure handling.
