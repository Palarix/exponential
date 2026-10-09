# Walkthrough: worktrees are the only mode

## Why branch mode had to go
`.xpo/` holds **hub state**: the append-only event log (`issues.db`) and the artifacts that every concurrent piece of work reads and writes. In worktree mode the primary checkout (the hub) stays on the default branch and `.xpo/` stays with it, while each issue gets its own checkout under `.xpo/worktrees/<branch>/`.

Branch mode checked the feature branch out *in the hub*, so `.xpo/` travelled with the branch. That caused three problems:

- **Merge broke.** MCP writes kept `issues.db` dirty on the feature branch, so `git checkout main` refused to run.
- **Events could be lost.** When the checkout did succeed, a squash merge took the branch's `issues.db` wholesale. `unionLines` tried to stitch main's version back in, but any events another worktree had written to main since the branch forked could still be overwritten.
- **Every fix was another patch.** The rest of the system already treats `.xpo/` as hub state, so each fix for branch mode added more checkout-and-merge workarounds.

Rather than keep patching, this issue removes the mode.

## What changed

### `StartWork` (`internal/exponential/start.go`)
In a git repo, `StartWork` now does one thing: create or reuse the worktree, either with a new `<id>-<slug>` branch off the default branch or for an existing branch. Outside a git repo it still only transitions the issue to DOING.

Removed: the "resume on current branch" path, the "branch already exists, use --force" pre-check (in worktree mode an existing branch just gets a worktree), and the checkout flow.

Two details are easy to miss:
- **The hub guard comes first.** If the hub has the issue branch checked out, which happens when someone upgrades halfway through old branch-mode work, start fails with "branch X is checked out in the hub — check out main there first". Without this guard, git refuses to add the worktree, and the "resume worktree" lookup below would match the hub *itself* as the issue's worktree (`git worktree list` reports the main checkout too) and wrongly report success.
- **The resume path is not gated on `CheckGitRepo()`.** That helper only stats `.git` in the current directory, so it returns false in a subdirectory of the worktree, which would break `TestStartWork_Worktree_Idempotent_Subdirectory`. The path used to be gated on `Config.Worktrees`, which defaulted to true, so running it unconditionally keeps the old behavior. Outside git, `FindWorktreeForBranch` simply finds nothing.

### `MergeIssue` (`internal/exponential/merge.go`)
- It always requires `HubBranch() == base` and **never checks anything out**. If the hub is elsewhere, the merge fails before touching git state, with the existing "park the hub on the default branch" message.
- The `mainIssuesDB`/`unionLines` save-and-restore is gone. With the hub never leaving main, `issues.db` isn't brought in from the branch, so there's nothing to reconcile. The `.gitattributes` `merge=union` driver stays, for the regular `merge` strategy.
- Worktree cleanup runs whenever the branch has a worktree. **A branch without a worktree still merges**, for example one made by hand or whose worktree was removed. Merging from the hub never needed the worktree in the first place.

### Removed surface
- `git.go`: `CheckoutBranch`, `CreateAndCheckoutBranch` and `RemoteBranchExists`. All three were used only by branch mode.
- `config.Config.Worktrees` and its default. `xpo init` no longer writes `worktrees: true`.
- CLI: `xpo start --mode` and `xpo merge --no-wt`. cobra now reports "unknown flag".
- `jsonio.StartToolInput.Mode`:
  - `xpo start --json` decodes strictly, so `mode` is an unknown-field error.
  - On MCP, the go-sdk's inferred schema sets `additionalProperties: false`, so an old client's `mode` argument fails validation instead of being silently dropped. Rejecting it was the user's choice: it's a breaking release, and callers should notice.

### Old configs
`LoadConfig` checks `v.IsSet("worktrees") && !v.GetBool("worktrees")` and writes one warning line to `warnOut` (stderr, swappable in tests). `worktrees: true`, which every `xpo init`-generated config contains, is a silent no-op. The user chose a warning over a hard error so that no command becomes unusable just because of a stale key.

### Docs and agent instructions
The README worktree section, `merge --help` and the MCP `start`/`merge` descriptions were updated. In the skill template in `agents.go`, "worktree or branch (depending on configuration)" and similar phrases now just say worktree. The installed copies under `.claude`, `.codex` and `.opencode` were regenerated with fresh `xpo:skill` hashes and checked against the generator with `CheckSkillFile`, so `xpo doctor` reports them as up to date and not hand-edited.

## Tests
- **Removed:** the MCP `mode` tests, the branch-mode start tests (create, reject existing branch, resume on current branch), the `unionLines` tests and the two `*PreservesMainIssuesDB` merge tests, which only covered the union-merge, and the checkout helper tests.
- **Fixtures changed:** `setupMergeRepo` and `TestMergeIssue_Squash` used to leave the hub on the feature branch and rely on merge's auto-checkout. They now commit the issue on main, branch off, and return to main. Every strategy test therefore also covers "a branch without a worktree merges from the hub". The MCP merge tests now commit inside the worktree that `start` returns.
- **Added:**
  - `TestStartWork_GitRepo_AlwaysCreatesWorktree`
  - `TestStartWork_BranchCheckedOutInHub`
  - `TestMergeIssue_HubNotOnBase_NoCheckout`
  - `TestMergeIssue_BranchWithoutWorktree`
  - `TestMCPStart_CreatesWorktree`
  - `TestStartToolInputRejectsMode`
  - `TestLoadConfigWarnsOnWorktreesFalse` and `TestLoadConfigAcceptsLegacyWorktreesTrueSilently`

## Acceptance criteria
- [x] `xpo start` always creates a worktree, and `--mode` and the MCP/JSON `mode` field are removed. Evidence: `TestStartWork_GitRepo_AlwaysCreatesWorktree`, `TestMCPStart_CreatesWorktree` and `TestStartToolInputRejectsMode`; the flag registration is deleted from `update.go`.
- [x] `xpo merge` only uses the hub path, and `--no-wt` is removed. Evidence: `TestMergeIssue_HubNotOnBase_NoCheckout` (the hub stays on the feature branch and gets an error), `TestMergeIssue_BranchWithoutWorktree`, and the flag deleted from `cmd/exponential/merge.go`.
- [x] No branch-mode code is left in `start.go`, `merge.go` or `git.go`. Evidence: a grep for `Worktrees`, `CheckoutBranch`, `unionLines` and `useWorktrees` finds nothing.
- [x] `Config.Worktrees` is removed, and `worktrees: false` warns. Evidence: the two `LoadConfig` warning tests.
- [x] Branch-mode tests are removed, and implicit branch-mode tests now assert worktree behavior. Evidence: see Tests above.
- [x] Docs, help and the skill template no longer mention branch mode. Evidence: README, `merge --help`, the MCP descriptions and the regenerated skill copies, verified up to date.
- [x] CHANGELOG has a **BREAKING:** entry under Removed with migration notes.
- [x] `make test` passes (exit 0). The user tested the change and approved it ("lgtm").
