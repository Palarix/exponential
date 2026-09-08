# Walkthrough: Test coverage for git.go

## What was built

New file `internal/exponential/git_test.go` with 14 tests covering every exported function in `git.go`.

## Test inventory

| Function | Tests | What's verified |
|---|---|---|
| `DefaultBranch` | 3 | Fallback to "main" with no remote, fallback with non-standard branch name, correct detection via `symbolic-ref` with a bare remote |
| `BranchExists` | 2 | Returns true for existing branch, false for nonexistent |
| `CreateAndCheckoutBranch` | 2 | Creates + switches to new branch; errors when branch already exists |
| `CheckoutBranch` | 2 | Switches to existing branch; errors for nonexistent |
| `WorktreeAdd/Remove/List` | 2 | Full lifecycle (add → list → remove → verify gone); hub checkout always appears in list |
| `FindWorktreeForBranch` | 3 | Linked worktree found with correct path; not found returns false; hub checkout on branch returns hub path (raw behavior — caller decides if hub counts) |

## Key decisions

- **Symlink resolution** — macOS temp dirs are under `/var/folders` which symlinks to `/private/var/folders`. Git resolves symlinks, so `initTestRepo` calls `filepath.EvalSymlinks` to match.
- **`DefaultBranch_WithRemote`** — required `git remote set-head origin develop` after push because `git push` alone doesn't set `refs/remotes/origin/HEAD`.
- **`FindWorktreeForBranch_HubCheckout`** — tests the raw behavior (returns the hub path + true), not the caller's policy. The merge fix in xpo-1fccf2 compares against `HubRoot()` at the call site.
