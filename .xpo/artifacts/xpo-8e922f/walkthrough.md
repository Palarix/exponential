# Walkthrough: Agent skill: recover when merge fails because the default branch moved on

## What changed

Step 9 (Complete) of the xpo skill now tells the agent what to do when `merge` fails because the default branch moved on. Before, the closing rule banned `git merge` and `git rebase` outright. The agent had no allowed way to fix a conflict, so it stopped and gave the conflict to the user.

The ban now covers only the default branch. To bring the default branch into the issue's branch from the worktree is the normal fix, and the skill now allows it explicitly.

## The new text

After the list item that calls `merge`:

> If it fails because the default branch moved on (conflicts), bring the default branch into the issue's branch from the worktree (`git rebase <default-branch>`, or `git merge <default-branch>`), resolve any conflicts, re-run the tests, commit, and call `merge` again.

The closing rule:

> Committing on the issue's worktree is your job. Merging is `merge`'s job: never run `git push`, or merge, rebase or commit on the default branch yourself. Rebasing the issue's branch onto the default branch (or merging the default branch into it) from the worktree is allowed.

## Files

- `internal/exponential/agents.go` — the step 9 template in `generateWorkflowBody`.
- `internal/exponential/agents_test.go` — `TestSkillStep9RecoversFromMergeConflicts`. It cuts step 9 out of `GenerateSkillMD()`, folds line wraps to single spaces, checks the recovery paragraph and the new closing rule, and checks that the old blanket ban is gone. The test was written first and failed before the template change.
- `.claude/`, `.codex/`, `.opencode/skills/xpo/SKILL.md` — regenerated with `WrapSkillFile(GenerateSkillMD(), version.CLIVersion)`.
- `CHANGELOG.md` — entry under Unreleased / Changed.

## Decisions

- **`<default-branch>` placeholder, not `main`.** `GenerateSkillMD()` takes no arguments, so the template does not know the configured default branch. The user chose a placeholder so the agent finds the real name itself. A hard-coded `main` is wrong in repos that use another name.
- **No change to `merge`.** The pre-check (`merge conflicts between committed refs`) and `mergeFailure` already list the conflicting files. Only the skill text was missing.
- **The `.claude` copy keeps a hand-added line.** It has "Follow the **Writing style** section in `references/spec-guide.md`…" from the ASD-STE100 experiment (64ea6a9). The generator does not produce this line, so regeneration removed it, and it was put back by hand. That copy differs from the generator output, as it did before this change.

## Acceptance criteria

- [x] Step 9 tells the agent to bring the default branch into the issue's branch, resolve the conflicts, re-run the tests, commit, and call `merge` again. — template change; checked by `TestSkillStep9RecoversFromMergeConflicts`.
- [x] The closing rule forbids `git push` and merge, rebase or commit on the default branch, and explicitly allows a rebase or merge of the default branch into the issue's branch from the worktree. — same test, which also checks that the old ban is gone.
- [x] The three installed SKILL.md copies match the generator. — regenerated; `.claude` also keeps its deliberate Writing style line.
- [x] A new test covers the new text. `make test` passes. — exit 0 (lint, 1573 frontend tests, Go suite).
- [x] CHANGELOG.md has an entry that references xpo-8e922f.
