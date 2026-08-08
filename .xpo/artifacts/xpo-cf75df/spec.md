## What

1. Update the generated `xpo-workflow` skill text to teach agents about worktree isolation and add an explicit review gate before walkthrough/commit/merge.
2. Make `xpo init` idempotent so existing projects can re-run it to refresh skill files without losing config customizations.

## Why

The `start` and `merge` MCP tools were added after the workflow skill text was written. The skill still tells agents to manually `update` status, which bypasses worktree creation entirely. Agents also tend to blast through completion comment → walkthrough → commit → merge without waiting for user review.

Existing projects need a safe way to pick up the updated skill text. Currently `xpo init` errors when `.xpo/` exists (without `--force`), and `--force` overwrites `config.yaml` with defaults.

## Acceptance Criteria

- Step 4 (Start) instructs agents to call `start` instead of `update`, explains worktree path and parallel isolation.
- Step 5 (Implement & Test) reminds agents to work inside the worktree path.
- Step 6 (Completion Comment) has an explicit "stop and wait" gate before walkthrough/commit/merge.
- Step 8 (Walkthrough) has a prerequisite check for user approval.
- Step 9 (Complete) covers `merge` for worktree-aware completion with an explicit gate checklist.
- MCP tools reference table includes `start` and `merge` entries.
- `xpo init` on an existing project skips config.yaml overwrite but refreshes skill files — no error, no data loss.
- `xpo init --force` behavior unchanged (full re-init including config overwrite).
- `make test` and `make build` pass.

## Flow

### Part 1: Skill text (agents.go) — DONE

Changes in `generateWorkflowBody()` and `generateMCPToolsBody()` in `internal/exponential/agents.go`.

### Part 2: Idempotent init (setup.go + init.go)

**`internal/exponential/setup.go` — `InitProject`:**
- When `.xpo/` exists and `!force`: don't return an error. Set `Created: false` and continue.
- Guard `config.yaml` write: only write when `result.Created || force`. Everything else in `InitProject` is already idempotent (issues.db uses `O_CREATE|O_APPEND`, gitignore/gitattributes use `Contains` checks).

**`cmd/exponential/init.go`:**
- No structural changes needed — removing the error return from `InitProject` means init naturally falls through to Phase 2 (skill writes).

## Decisions

- **Skip config.yaml, not everything**: `MkdirAll`, `issues.db` open, `.gitignore`/`.gitattributes` checks are all idempotent — no need to guard them. Only `config.yaml` overwrites user state.
- **No new flag**: Option 2 (make init graceful) over option 1 (`--refresh-skills` flag). Re-running `init` is the natural UX — a separate flag is unnecessary complexity.
