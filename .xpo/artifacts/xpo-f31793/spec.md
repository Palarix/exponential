# Spec: MCP-layer tests for start and merge tools

## What

Tests for the `start` and `merge` MCP tool handlers, focusing on MCP-layer logic not covered by the underlying `StartWork`/`MergeIssue` tests.

## Tests

### start
1. **TestMCPStart_BranchMode** — `mode: "branch"` overrides config to create a branch, not a worktree
2. **TestMCPStart_WorktreeMode** — `mode: "worktree"` creates a worktree even when config has worktrees disabled
3. **TestMCPStart_InvalidMode** — invalid mode returns error
4. **TestMCPStart_MissingID** — empty ID returns error

### merge
5. **TestMCPMerge_Squash** — default strategy squash-merges and returns merge SHA
6. **TestMCPMerge_InvalidStrategy** — invalid strategy returns error
7. **TestMCPMerge_MissingID** — empty ID returns error

## Approach

- New `setupGitToolset` helper that creates a git repo (unlike `setup()` which creates a non-git dir)
- Call tool handlers directly via the `toolset` methods, same pattern as existing tests

## Acceptance Criteria

- [ ] All 7 tests pass
- [ ] `make test` passes
- [ ] No production code changes
