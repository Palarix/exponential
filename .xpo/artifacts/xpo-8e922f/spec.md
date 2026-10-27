# Spec: Agent skill: recover when merge fails because the default branch moved on

## What

Change step 9 (Complete) of the xpo skill so that an agent can recover when `merge` fails with conflicts because the default branch moved on.

## Why

Step 9 forbids `git merge` and `git rebase` outright. When `merge` reports conflicts, the agent has no allowed way to fix them, so it stops and gives the conflict to the user. The ban must cover only the default branch. To bring the default branch into the issue's branch from its worktree is safe, and it is the normal fix.

## How

Edit the template in `internal/exponential/agents.go` (`generateWorkflowBody`, step 9).

1. After the list item that calls `merge`, add this paragraph:

   > If it fails because the default branch moved on (conflicts), bring the default branch into the issue's branch from the worktree (`git rebase <default-branch>`, or `git merge <default-branch>`), resolve any conflicts, re-run the tests, commit, and call `merge` again.

2. Replace the closing rule with:

   > Committing on the issue's worktree is your job. Merging is `merge`'s job: never run `git push`, or merge, rebase or commit on the default branch yourself. Rebasing the issue's branch onto the default branch (or merging the default branch into it) from the worktree is allowed.

3. Regenerate the installed copies in `.claude/`, `.codex/` and `.opencode/skills/xpo/SKILL.md`, with fresh `xpo:skill` markers.

4. Add a test in `agents_test.go` (test first). It checks that step 9:
   - contains the recovery paragraph,
   - allows a rebase or merge of the default branch into the issue's branch from the worktree,
   - does not contain the old blanket ban ("never run `git merge`, `git rebase`").

## Out of scope

- No change to `merge` itself or to its error messages. The pre-check (`merge conflicts between committed refs`) and `mergeFailure` already report the conflicting files.

## Decisions

- **Branch name.** `GenerateSkillMD()` takes no arguments, so the template does not know the configured default branch. The text uses the placeholder `<default-branch>` (for example `git rebase <default-branch>`). The agent finds the real name itself. A hard-coded `main` would be wrong in repos that use another name.

## Acceptance criteria

- [ ] Step 9 tells the agent to bring the default branch into the issue's branch, resolve the conflicts, re-run the tests, commit, and call `merge` again.
- [ ] The closing rule forbids `git push` and merge, rebase or commit on the default branch, and explicitly allows a rebase or merge of the default branch into the issue's branch from the worktree.
- [ ] The three installed SKILL.md copies match the generator.
- [ ] A new test covers the new text. `make test` passes.
- [ ] CHANGELOG.md has an entry that references xpo-8e922f.
