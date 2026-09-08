# Spec: Add test coverage for git.go operations

## What

Unit tests for every exported function in `internal/exponential/git.go`.

## Why

Zero test coverage on the file that every git operation flows through. The merge auto-checkout regression (xpo-1fccf2) came from `FindWorktreeForBranch` — would have been caught with basic tests.

## Tests

### DefaultBranch
1. **TestDefaultBranch_Main** — repo with `main` as default. Verify returns "main".
2. **TestDefaultBranch_CustomName** — repo with a non-standard default branch name (e.g. "trunk"). Verify returns "trunk".

### BranchExists
3. **TestBranchExists_True** — existing branch. Verify returns true.
4. **TestBranchExists_False** — nonexistent branch. Verify returns false.

### CreateAndCheckoutBranch
5. **TestCreateAndCheckoutBranch** — create a new branch. Verify current branch changes and branch exists.
6. **TestCreateAndCheckoutBranch_AlreadyExists** — branch already exists. Verify error.

### CheckoutBranch
7. **TestCheckoutBranch** — switch to existing branch. Verify current branch changes.
8. **TestCheckoutBranch_Nonexistent** — switch to nonexistent branch. Verify error.

### WorktreeAdd / WorktreeRemove / WorktreeList
9. **TestWorktreeLifecycle** — add a worktree, verify it appears in list with correct branch, remove it, verify it's gone from list and filesystem.

### FindWorktreeForBranch
10. **TestFindWorktreeForBranch_Found** — linked worktree exists for branch. Verify returns path + true.
11. **TestFindWorktreeForBranch_NotFound** — no worktree for branch. Verify returns "" + false.
12. **TestFindWorktreeForBranch_HubCheckout** — hub itself is on the branch (no linked worktree). Verify returns the hub path + true (this is the raw behavior; the *caller* decides whether to treat hub-as-worktree differently, as merge.go does).

## Approach

- New file `internal/exponential/git_test.go`
- Each test creates its own temp dir with `t.TempDir()`, inits a git repo
- Reuse `runGit` helper from `branches_test.go` (same package)
- `DefaultBranch` uses `git symbolic-ref` which requires at least one commit

## Acceptance Criteria

- [ ] All 12 tests pass
- [ ] `make test` passes
- [ ] No production code changes
