## What

Default `DeleteBranch` to `true` in the MCP merge handler so branches are cleaned up after merge, matching the `drive.go` behavior.

## Why

After a squash merge the issue branch is dead weight. The worktree is always cleaned up, but the branch lingers unless the caller explicitly passes `delete_branch: true`. Agents using the MCP tool don't pass it, leaving stale branches.

## How

In `internal/mcpserver/tools.go`, the merge handler constructs `MergeOptions` at line 527-531. Change `DeleteBranch: in.DeleteBranch` to default to `true` when the caller hasn't explicitly set it to `false`. Since the MCP schema uses `omitempty` on the bool field, an unset value is `false` — so we need to flip the default: use `!in.KeepBranch` or simply hardcode `true` unless a new opt-out field is added.

**Chosen approach:** Add a `KeepBranch` bool field to the MCP input struct (opt-out instead of opt-in). Set `DeleteBranch: !in.KeepBranch`. This way the default behavior is to delete, and callers who want to keep the branch explicitly pass `keep_branch: true`.

## Acceptance Criteria

- Calling `merge` via MCP without any branch flag deletes the branch after merge
- Passing `keep_branch: true` preserves the branch
- The old `delete_branch` field is removed to avoid conflicting booleans
- Existing tests pass
