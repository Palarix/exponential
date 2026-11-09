## MCP Tools Reference

The xpo MCP server exposes the tools listed below. Exact tool identifiers depend on your
agent harness (e.g. Claude Code surfaces them as `mcp__xpo__<tool>`; other harnesses
may use different naming conventions). The table uses the base tool names.

| Tool | Description |
|---|---|
| `list` | List or search issues; use `match` for free-text search and `milestone` to filter by milestone |
| `show` | Read one issue with details, dependencies, and comments |
| `add` | Create a new issue. Set `before` or `after` to an issue ID to place it directly above or below that issue |
| `update` | Update fields including status transitions (BACKLOG/PLANNED/DOING/BLOCKED/DONE/CANCELED/DUPLICATE) and milestone. Reorder with `before` or `after` set to an issue ID |
| `start` | Start working on an issue: transitions to DOING and creates a git worktree. Returns the worktree path |
| `merge` | Squash-merge an issue branch into the default branch, record a MERGE event, close the issue, and clean up the worktree. All changes must be committed in the worktree first. Call it without a `strategy` unless the user asks for one |
| `comment` | Add a markdown comment to an issue |
| `link` | Add a relationship between two issues |
| `unlink` | Remove one relationship between two issues, whichever side stores it |
| `history` | View the audit trail for an issue |
| `milestone` | List, create, update, reorder, complete or delete milestones |
| `spec` | Read, write, or delete the design spec for an issue |
| `walkthrough` | Read, write, or delete the implementation walkthrough for an issue |
| `rationale` | Search across specs and walkthroughs for prior design decisions related to a topic |
| `artifact` | Manage generic artifacts attached to an issue |
| `git_status` | Show an issue branch's ahead/behind state and its worktree's uncommitted files (read-only) |
| `git_diff` | Show what an issue's branch changed, including uncommitted changes (read-only) |
| `git_log` | List the commits of an issue's branch or of the default branch (read-only) |
| `git_show` | Show one commit's metadata, message and patch (read-only) |
| `code_history` | Find the commits and issues that touched a file or a line range (read-only) |

<!-- xpo:skill 1.5.0 sha256:c57d4b533baa -->
