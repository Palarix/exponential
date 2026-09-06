# Walkthrough: Tolerate untracked files on hub during merge

## What changed

The pre-merge working tree cleanliness check was replaced across all merge paths (CLI, MCP server, web UI) with a smarter function that only blocks on actual conflicts.

## The old behavior

`IsWorkingTreeCleanIgnoringXpo()` ran `git status --porcelain` and rejected ANY non-`.xpo/` dirty file — including untracked scratch files like `idea.md`. The generic error message ("commit or stash them first") caused AI agents to stash indiscriminately, which corrupted `.xpo/issues.db` (designed to flow hub-ward) and polluted git history.

## The new behavior

`HubCleanForMerge(branch string) error` categorizes dirty files:

1. **`.xpo/` files** — ignored (uncommitted events are expected on the hub)
2. **Untracked files (`??`)** — ignored. Git merge itself will error if the branch introduces a file that collides, with a specific message.
3. **Modified tracked files** — checked against the branch's changed files via `git diff --name-only base...branch`. Only files that appear in BOTH the local modifications and the branch's changes are flagged as conflicts.

When a conflict IS found, the error lists the specific files and explicitly says "Do NOT stash" to short-circuit agent thrashing.

## Files changed

- **`internal/exponential/merge.go`** — Added `HubCleanForMerge()`. Removed `IsWorkingTreeCleanIgnoringXpo()`. Kept `IsWorkingTreeClean()` for non-merge callers (`drive.go`). Updated merge failure message to be agent-safe.
- **`cmd/exponential/merge.go`** — Replaced the branching worktree/non-worktree check (lines 60-72) with a single `HubCleanForMerge()` call.
- **`internal/mcpserver/tools.go`** — Same replacement. Restructured to resolve the issue before the check so the branch name is available.
- **`internal/server/handlers.go`** — Updated both `handleMergeability` (controls the merge button in the web UI) and `handleMergeIssue` (performs the merge) to use `HubCleanForMerge()`.
- **`internal/exponential/merge_test.go`** — Added 5 tests: clean tree, untracked files allowed, .xpo changes allowed, non-conflicting modifications allowed, conflicting modifications blocked with specific error.

## Key decisions

- **Kept `IsWorkingTreeClean()`** for `drive.go` — drive's preflight check is a different concern (general health before spawning agents) and doesn't need the branch-aware logic.
- **Three-dot diff** (`base...branch`) to compare only the branch's divergent changes, not the full diff against HEAD.
- **Error messages include "Do NOT stash"** — agents read error text as instructions, so the phrasing is intentional to prevent the stash-thrash loop.
