# ff merge folds bookkeeping into the tip; agents stop picking ff

## What

1. `merge` with `strategy: ff` amends `.xpo/issues.db` and `.xpo/artifacts/<id>/` into the fast-forwarded tip commit, instead of adding a separate `xpo: merge <id>` commit.
2. The tool description, the mcp-tools reference and skill step 9 tell agents to leave `strategy` unset unless the user asks for one.

## Why

Every ff merge currently leaves an extra bookkeeping commit on main. Agents choose `ff` unprompted, because step 9 has them make one well-named commit first and nothing says to leave the strategy at its default.

## How

- **`merge.go`, the bookkeeping block:** every strategy now amends (`--amend --no-edit --no-verify`). For ff, the amended commit is the fast-forwarded branch tip; the branch and worktree are removed right after.
  - If the ff range has several commits, only the tip is amended.
  - A no-op ff can't reach this point: `MergeIssue` already rejects a branch with no commits ahead of base before anything runs. So amending never rewrites a commit that was already on base, and no guard is needed.
- **The `xpo: merge <id>` default message is removed.**
  - An explicit `commit_message` with ff now rewords the amended tip (`--amend --no-verify -m <msg>`). Before, it became the subject of the bookkeeping commit; without this it would be silently ignored.
- **`result.MergeSHA`** already re-reads HEAD after bookkeeping, so it reports the amended SHA.
- **The amend-failure fallback is unchanged:** reset `.xpo` and return the warning.
- **Text:**
  - `merge` tool description (`mergeToolDescription` in `tools.go`) and the `strategy` jsonschema text: omit `strategy` unless the user asks for one.
  - The `merge` row in `GenerateMCPToolsRef()`: same.
  - Step 9 in `GenerateSkillMD()`: "Call `merge` without a `strategy`".
  - Regenerate the installed skill copies in this repo (`.claude`, `.codex`, `.opencode`) with `WriteAgentSkill`.

## Decisions

- **With `keep_branch`**, the kept branch ends one commit behind main's amended tip (the same content minus bookkeeping). That's acceptable: the branch is history only.
- **Spec change during implementation:** the originally planned "no-op ff keeps a separate commit" guard was dropped, because that case is unreachable. The custom-message handling was added.

## Acceptance criteria

- [ ] `merge` with `strategy: ff` leaves exactly the branch's commits on main, with no extra `xpo: merge <id>` commit. The tip includes `.xpo/issues.db` and `.xpo/artifacts/<id>/`.
- [ ] An explicit `commit_message` with ff becomes the tip's subject.
- [ ] Squash and merge behaviour is unchanged (existing tests).
- [ ] An amend failure leaves the events recorded but uncommitted and returns the warning.
- [ ] The tool description, the mcp-tools reference and the step 9 skill text tell agents to omit `strategy` by default. A test asserts the skill text.
- [ ] `make test` passes.
