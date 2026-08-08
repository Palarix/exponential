## What changed

The generated `xpo-workflow` skill text (used by all agent harnesses) now teaches agents about worktree isolation and enforces an explicit review gate before walkthrough/commit/merge. Additionally, `xpo init` is now idempotent — safe to re-run on existing projects to refresh skill files.

## Files changed

### `internal/exponential/agents.go`

Two functions were updated: `generateWorkflowBody()` and `generateMCPToolsBody()`.

**`generateWorkflowBody()` — Step 4 (Start):**
The old text told agents to "transition to DOING and set assignee to yourself" via the `update` tool. This bypassed `start` entirely, meaning agents never created worktrees. The new text instructs agents to call `start` with the issue ID, explains that `start` handles both the status transition and worktree creation, and describes the `worktree_path` return value. It also covers parallel isolation (each issue gets its own worktree) and `force: true` for resuming work.

**`generateWorkflowBody()` — Step 5 (Implement & Test):**
Added a leading bullet reminding agents that all file reads, edits, builds, and test runs must target the worktree directory, not the main checkout.

**`generateWorkflowBody()` — Step 6 (Completion Comment):**
Added a bold "Stop here and wait" callout. This is the key behavioral fix — agents were blasting through completion comment → walkthrough → commit → merge as a single atomic operation, not giving the user a chance to tophat. The callout explains that proceeding without user response risks wasted walkthrough work and unwanted commits/merges.

**`generateWorkflowBody()` — Step 8 (Walkthrough):**
Added a prerequisite check: "If the user has not responded since your completion comment, you are NOT on this step." This catches agents that skip the gate in Step 6.

**`generateWorkflowBody()` — Step 9 (Complete):**
Added `merge` tool guidance and an explicit gate checklist. The three conditions (explicit user approval, passing tests, walkthrough attached) must all be true before committing or merging. "Silence after a completion comment" is explicitly called out as not counting as approval.

**`generateMCPToolsBody()`:**
Added `start` and `merge` to the MCP tools reference table. These tools existed in the MCP server but were missing from the generated documentation.

### `internal/exponential/setup.go`

**`InitProject()`:**
The function previously returned an error when `.xpo/` already existed without `--force`. Now it continues gracefully — `Created` is set to `false` and execution falls through to the idempotent steps (issues.db touch, gitignore/gitattributes checks). The `config.yaml` write is guarded by `result.Created || force`, so re-running `xpo init` never overwrites user customizations.

The rest of Phase 1 was already idempotent:
- `issues.db`: `O_CREATE|O_WRONLY|O_APPEND` then immediate close — a no-op touch
- `.gitignore` / `.gitattributes`: `Contains` checks before appending

### `cmd/exponential/init.go`

Added a message for the re-run case: "`.xpo` directory already exists, refreshing agent configuration". This replaces the old error exit and tells the user what's happening as init proceeds to refresh skill files.

## Migration path

Existing projects pick up the updated skill text by: updating the `xpo` binary, then running `xpo init`. Init sees `.xpo/` exists, skips config.yaml, and overwrites the skill files under `.claude/skills/xpo-workflow/` (or equivalent for other harnesses) with the new content.
