# Remove branch mode from xpo start/merge

## What
Worktree mode becomes the only way to work on an issue. `xpo start` always creates (or reuses) a worktree under `.xpo/worktrees/`. `xpo merge` always merges from the hub checkout, which must be on the default branch, and never runs `git checkout`. Every switch that selects branch mode is removed. **This is a breaking change** and is recorded in the CHANGELOG as one.

## Why
See the issue description. `.xpo/` is hub state, and branch mode makes it travel with the feature branch. That breaks `merge` (dirty `issues.db` blocks the checkout) and risks losing events (a squash merge overwrites main's `issues.db`, which `unionLines` only partly papers over).

## How

### Core (`internal/exponential`)
- **`start.go`**
  - `StartWork` drops the `useWorktrees` switch and the branch-mode code paths: the "resume on current branch" idempotent path, the `BranchExists`/`RemoteBranchExists` pre-check, and the `CheckoutBranch`/`CreateAndCheckoutBranch` flow.
  - In a git repo it always goes down the worktree path. Outside a git repo it still only transitions the issue, as today.
- **`merge.go`**
  - `MergeIssue` loses `useWorktrees`. It always requires `HubBranch() == base`, which surfaces the existing "park the hub on the default branch" error, and never checks out `base`.
  - The `mainIssuesDB`/`unionLines` save-and-restore is removed, along with `unionLines` and its tests.
  - Worktree cleanup runs whenever a worktree for the branch exists. **A branch with no worktree can still be merged**, for example a hand-made branch or one whose worktree was removed. It's merged from the hub just like a worktree branch.
- **`git.go`**: `CheckoutBranch` and `CreateAndCheckoutBranch` are deleted, with their tests, since nothing uses them any more.

### Config
- `Config.Worktrees` and the `worktrees` default are removed, and `xpo init` stops writing `worktrees: true`.
- A config that still sets `worktrees: false` gets a one-line stderr warning when it's loaded: `worktrees: false is no longer supported — xpo always uses worktrees; remove the key from .xpo/config.yaml`. `worktrees: true` is accepted silently, because it's what `xpo init` wrote and is now a no-op.

### CLI / MCP / JSON
- `xpo start --mode` and `xpo merge --no-wt` are removed. Passing them fails with cobra's "unknown flag" error.
- MCP `start` and `xpo start --json`: the `mode` field is removed from `jsonio.StartToolInput`. Strict decoding means a stale `mode` field is rejected with "unknown field". It isn't silently ignored, so callers notice.
- Text that described the choice is updated: tool descriptions, `merge --help`, the README's worktree section, and the skill template in `agents.go` ("worktree or branch (depending on configuration)" → worktree). The installed copies under `.claude/`, `.codex/` and `.opencode/` are regenerated from the template.

### Tests
- Remove `TestMCPStart_BranchMode`, `TestMCPStart_WorktreeMode` and `TestMCPStart_InvalidMode`, the branch-mode cases in `start_test.go`/`merge_test.go`, the `unionLines` tests, and the `CheckoutBranch` tests.
- Test clients that relied on the zero-value config (`Worktrees: false`, so implicitly branch mode) now exercise the worktree path. Their assertions are updated to match, not deleted.
- New tests:
  - `StartWork` in a git repo always returns a worktree path and leaves the hub on main.
  - Merging a branch that has no worktree succeeds from the hub.
  - `MergeIssue` errors without running a checkout when the hub isn't on main.
  - Loading a config with `worktrees: false` warns.

### CHANGELOG
Under **Removed**, flagged `**BREAKING:**`. It lists the removed flags, the removed MCP/JSON field and the `worktrees` key, plus a migration note: finish or merge any in-flight branch-mode work, and park the hub on main, before upgrading.

## Acceptance criteria
- [ ] `xpo start` always creates a worktree; `--mode` and the MCP/JSON `mode` field are removed.
- [ ] `xpo merge` only uses the hub path (no checkout); `--no-wt` is removed.
- [ ] No branch-mode code remains in `start.go`, `merge.go` or `git.go`. `unionLines`, `CheckoutBranch` and `CreateAndCheckoutBranch` are gone.
- [ ] `Config.Worktrees` is removed; `worktrees: false` in a config warns on load.
- [ ] Branch-mode tests are removed, and tests that implicitly ran branch mode now assert worktree behavior.
- [ ] Docs, help text and the skill template no longer mention branch mode.
- [ ] The CHANGELOG has a **BREAKING** entry with migration notes.
- [ ] `make test` passes.

## Decisions (confirmed with user)
1. **`worktrees: false` in an existing config:** print a stderr warning on load and use worktrees anyway. No hard error, and the key isn't silently ignored.
2. **Removed flags and `mode` field:** removed outright, so they fail with unknown flag or unknown field errors. There's no deprecated no-op release.

## Implementation notes (found while building)
- **Hub has the issue branch checked out** (upgrading mid-way through branch-mode work): `StartWork` now fails early with "branch X is checked out in the hub — check out main there first". Without this check, git can't add the worktree, and worse, the idempotent "resume worktree" lookup would match the hub itself as the issue's worktree. Covered by `TestStartWork_BranchCheckedOutInHub`.
- **The idempotent "resume worktree" path isn't gated on `CheckGitRepo()`.** That helper only checks for `.git` in the cwd, so gating on it would break resuming from a subdirectory of the worktree. The path used to be gated on `Config.Worktrees`, which defaulted to true, so it now just runs unconditionally. `FindWorktreeForBranch` returns false outside git anyway.
- **`RemoteBranchExists` was removed too.** The branch-mode pre-check was its only caller.
- **MCP rejects a stale `mode`:** the go-sdk's inferred input schema sets `additionalProperties: false`, so an old client's `mode` argument fails validation instead of being dropped silently.
- **Merge test fixtures:** `setupMergeRepo` and `TestMergeIssue_Squash` used to leave the hub on the feature branch and rely on merge's auto-checkout. They now commit the issue on main, branch off, and return the hub to main, which also covers "branch without a worktree merges from the hub". The two `*PreservesMainIssuesDB` tests covered only the removed union-merge and are deleted.
