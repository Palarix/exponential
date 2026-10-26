# Walkthrough: resuming a DOING issue no longer deletes its worktree

## What happened
In Work Mode, Codex called `start` with `force: true` on its own issue. It did this because the xpo skill said "To resume an issue already in DOING, call `start` with `force: true`." `StartWork(force=true)` removed the existing worktree first and then re-added it. The re-add failed, so the branch was left without a worktree.

The bug had two parts. The instruction was wrong. And force was destructive, so following a wrong instruction could lose work.

## What changed

### `internal/exponential/start.go`
- **Resume without force.** The resume path used to run only when the current directory was inside the worktree. The MCP server runs in the hub, so for agents that check almost never passed, and they got "already in progress — use --force". The path now also runs when the issue is DOING and `startersOwn(issue)` is true. It returns the existing worktree, doesn't change the issue, and doesn't run the setup hook.
- **`startersOwn`** returns true when starting needs no takeover: the issue is unassigned, assigned to an agent, or assigned to the starter's principal. It uses the same rule that `startPayload` already used to decide whether to reassign. Both now share the new `principal()` helper.
- **Non-destructive force.** `WorktreeRemove` is gone from `StartWork`. If the issue's worktree exists, force reuses it with "Reusing existing worktree", and `kept` skips the setup hook. If it doesn't exist, force adds it for the existing branch or for a new one, as before. Force still takes over the assignee through `startPayload(issue, true)`.

This reverses xpo-7325f7's decision to keep force destructive. That decision assumed force was rare and done on purpose. Agents were told to use it routinely.

### Instructions
- `agents.go` skill template, and the `.claude/`, `.codex/` and `.opencode/` SKILL.md copies: "If you are already working in the issue's worktree (for example in Work Mode), don't call `start` again. Use `force: true` only to take over an issue someone else is working on, and only after the user approves."
- `internal/chat/prompt.md`, Work Mode: "The issue is already started, so don't call `start`."

The `.claude/` skill copy has local customizations (a "Writing style" line in SKILL.md and an edited `spec-guide.md`). They are kept on purpose. Only the force sentence was swapped there, and the marker hash was updated. The `.codex/` and `.opencode/` copies match the template exactly.

## Acceptance criteria
- [x] The skill no longer says to resume with force, and limits force to takeovers with the user's approval. All three copies are updated. Verified with `CheckSkillFile`: `.codex` and `.opencode` are up to date, and `.claude` differs only by its kept customizations.
- [x] The Work Mode prompt says not to call `start`. See `prompt.md`.
- [x] Without force, your own DOING issue resumes from outside its worktree: `TestStartWork_Worktree_ResumeOwnDoingFromHub`.
- [x] Without force, someone else's DOING issue still errors: `TestStartWork_Worktree_OthersDoingStillNeedsForce`.
- [x] Force keeps the worktree and an uncommitted file, returns the same path, reassigns the issue and skips the hook: `TestStartWork_Worktree_ForceKeepsExisting`. It replaces `ForceRemovesExisting` and `Idempotent_ForceStillDestroys`.
- [x] Force adds a missing worktree: `TestStartWork_Worktree_ForceAddsMissing`.
- [x] `make test` passes.
- [x] CHANGELOG entry under Fixed.
