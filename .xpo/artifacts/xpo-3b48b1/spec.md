# Spec: resuming a DOING issue must never delete its worktree

## What
Stop agents from destroying their own worktree when they resume an issue. Fix the instructions that lead them to call `start` with force, and make `start` safe even when they do.

## Why
Codex followed the skill's line "To resume an issue already in DOING, call `start` with `force: true`". `StartWork(force=true)` removes the existing worktree and then re-adds it. The re-add failed, so the branch was left without a worktree. The worktree was clean, so nothing was lost this time.

xpo-7325f7 kept force destructive on purpose: "`force=true` still does destructive…". This issue reverses that decision with the user's approval.

## How
1. **Skill** (`internal/exponential/agents.go`, then regenerate `.claude/`, `.codex/` and `.opencode/` skills): replace the force sentence with:
   - "If you are already working in the issue's worktree (for example in Work Mode), don't call `start` again."
   - "Use `force: true` only to take over an issue someone else is working on, and only after the user approves."
2. **Work Mode prompt** (`internal/chat/prompt.md`): add "The issue is already started and its worktree is your working directory, so don't call `start`."
3. **Resume without force** (`start.go`): when the issue is DOING, the assignee is the person starting it, and `FindWorktreeForBranch` finds its worktree, return that worktree with "Resuming existing worktree". The current directory doesn't matter. Don't change status and don't run the setup hook. If the issue is DOING under someone else, still return the "use --force" error.
4. **Non-destructive force** (`start.go`): never call `WorktreeRemove`. If the worktree exists, reuse it. If it doesn't, add it for the existing branch, or for a new branch. Force still reassigns the issue.

## Acceptance criteria
- [ ] The generated skill no longer says to resume with force, and says force is only for taking over with user approval. The skill files are regenerated.
- [ ] The Work Mode prompt says not to call `start`.
- [ ] `start` without force on a DOING issue assigned to the same person, called from outside its worktree, returns the existing worktree path and doesn't error.
- [ ] `start` without force on a DOING issue assigned to someone else still errors.
- [ ] `start` with force on an issue that has a worktree keeps the worktree and its files, including uncommitted ones. It returns the same path and reassigns the issue. This replaces `TestStartWork_Worktree_ForceRemovesExisting` and `TestStartWork_Worktree_Idempotent_ForceStillDestroys`.
- [ ] `start` with force on a DOING issue whose worktree is missing adds the worktree.
- [ ] `make test` passes.
- [ ] CHANGELOG entry.

## Open questions
- "Assigned to the same person": compare the issue's assignee with the starter identity that `startPayload` uses. If the issue has no assignee or an agent assignee, treat it as the same person, which matches what `startPayload` already allows.
