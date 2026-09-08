# Spec: Add test coverage for start.go worktree edge cases

## What

Additional tests for `StartWork` worktree edge cases not covered by the existing 11 tests.

## Tests

1. **TestStartWork_Worktree_ExistingBranch** — branch already exists (e.g. from a previous start), worktree mode uses `WorktreeAddExisting` instead of `WorktreeAdd`. Verify worktree is created and uses the existing branch.
2. **TestStartWork_Worktree_ForceRemovesExisting** — worktree already exists for the branch; `force: true` removes it and creates a fresh one.
3. **TestStartWork_Worktree_SetupHook** — `Config.WorktreeSetup` is set to a command. Verify the command runs in the worktree directory (write a marker file).
4. **TestStartWork_Worktree_SetupHookFails** — setup hook exits non-zero. Verify warning message but no error (start still succeeds).

## Acceptance Criteria

- [ ] All 4 tests pass
- [ ] `make test` passes
- [ ] No production code changes
