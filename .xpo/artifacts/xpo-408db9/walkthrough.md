## What changed

The MCP and REST API merge endpoints now delete the issue branch by default after merge, matching the behavior that `drive.go` (headless mode) already had.

## How it works

The core `MergeOptions` struct has two branch-related fields: `DeleteBranch` and `KeepBranch`. The CLI merge command uses both with an interactive prompt. The fix is at the API layer only:

- **MCP tool** (`internal/mcpserver/tools.go`): Replaced `delete_branch` (opt-in bool, default false) with `keep_branch` (opt-out bool, default false). The handler passes `DeleteBranch: !in.KeepBranch` to `MergeOptions`, so branches are deleted unless the caller explicitly opts out.
- **REST API** (`internal/server/handlers.go`): Same change — `delete_branch` field replaced with `keep_branch`, handler passes `DeleteBranch: !body.KeepBranch`.
- **`drive.go`**: Unchanged — it calls `MergeOptions` directly with `DeleteBranch: true`, which was already correct.
- **CLI `merge` command**: Unchanged — it has its own `--delete-branch` / `--keep-branch` flags with an interactive prompt.

The core `MergeOptions` struct and `MergeIssue` logic are untouched. The default flip happens only at the API boundary where agents interact.
