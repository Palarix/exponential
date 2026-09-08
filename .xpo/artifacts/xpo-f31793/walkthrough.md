# Walkthrough: MCP-layer tests for start and merge tools

## What was built

New file `internal/mcpserver/start_merge_test.go` with 7 tests covering the `start` and `merge` MCP tool handlers.

## Test inventory

| Test | What's verified |
|---|---|
| `TestMCPStart_BranchMode` | `mode: "branch"` overrides config, creates branch (no worktree path) |
| `TestMCPStart_WorktreeMode` | `mode: "worktree"` creates worktree even with default config |
| `TestMCPStart_InvalidMode` | Invalid mode returns descriptive error |
| `TestMCPStart_MissingID` | Empty ID returns error |
| `TestMCPMerge_Squash` | Full start→commit→merge flow via MCP, returns merge SHA |
| `TestMCPMerge_InvalidStrategy` | Invalid strategy returns error |
| `TestMCPMerge_MissingID` | Empty ID returns error |

## Key decisions

- **`setupGitToolset`** — unlike the existing `setup()` which creates a non-git dir, this creates a full git repo with an initial commit. Uses `filepath.EvalSymlinks` for macOS.
- **Direct handler calls** — tests call `ts.start()` / `ts.merge()` with `nil` request (supported by `clientFor`'s nil-request path) to test MCP-layer logic without an MCP server.
- **`TestMCPMerge_Squash`** — exercises the full lifecycle (start → commit → merge) to verify the MCP merge handler works end-to-end, not just input validation.
