# Spec: read-only git MCP tools for chat sessions

## What

Add five read-only MCP tools to the local stdio xpo MCP server (`xpo mcp`): `git_status`, `git_diff`, `git_log`, `git_show` and `code_history`. The server builds every git command itself. The model supplies only an issue ID, a path, a commit SHA, a line range and a limit. Chat sessions get the tools through the existing `--allowedTools mcp__xpo__*` rule, so the chat tool profile does not change.

## Why

Part of Pillar 1 of xpo-3798b7, on the chat engine from xpo-56ae3a. Chat sessions run with `Read,Grep,Glob` and no Bash. The agent cannot answer "what changed on this issue's worktree?" or "why is this code like this?".

Bash prefix rules such as `Bash(git diff:*)` are not safe. `--output=<file>` writes files. `--ext-diff`, textconv and `core.fsmonitor` run programs from repo config. `git -c` sets arbitrary config. A fixed command line with validated arguments removes all of these paths.

**Design rule (agreed during spec review):** anything that touches code runs where the code is, on the user's machine. Issue data lives wherever the transport points. The git tools follow this rule: they read the issue through the process's transport (local or remote) and always run git against the local hub.

## Acceptance Criteria

- [ ] `git_status {id}` returns the branch, the worktree path, ahead/behind counts against the default branch, and the staged, unstaged and untracked files.
- [ ] `git_diff {id, path?, stat?}` returns the diff from the merge base to the worktree, including uncommitted changes to tracked files. It lists untracked files by name.
- [ ] `git_diff` with `stat: true` returns per-file insertions and deletions and no patch text.
- [ ] `git_diff` on an issue with a branch and no worktree returns the diff from the merge base to the branch tip.
- [ ] `git_log {id?, path?, limit?}` returns commits with sha, subject, author, date and the issue IDs parsed from the subject.
- [ ] `git_show {sha, path?}` returns one commit's metadata, message and patch.
- [ ] `code_history {path, start_line?, end_line?, limit?}` returns the commits and the issues (ID, title, status) that touched the file or line range.
- [ ] Every tool rejects a path that is absolute, contains `..`, or starts with `-`.
- [ ] `git_show` rejects a `sha` that is not 4–40 hex characters.
- [ ] Every tool returns an error for an issue with no branch. The error names the issue.
- [ ] Every tool caps its text output at 64 KiB. A capped result sets `truncated: true` and ends with a marker line.
- [ ] A test repo with `diff.external`, a textconv driver and `core.fsmonitor` set to marker-writing scripts runs every tool. No marker file appears.
- [ ] No tool creates or modifies `.git/index` or any `.lock` file (`GIT_OPTIONAL_LOCKS=0`).
- [ ] With a remote transport (`cfg.Remote.URL` set), the tools resolve the issue through the server and run git against the local hub.
- [ ] `xpo serve` does not offer the git tools on its `/mcp` endpoint.
- [ ] The chat transcript shows readable summaries for the new tools (for example "Read the diff of xpo-a1b2c3").
- [ ] The chat system prompt tells the agent about the git tools.
- [ ] `make test` passes.

## Flow

### 1. Safe git runner — `internal/exponential/gitread.go`

1. Add `readGit(dir string, args ...string) (out string, truncated bool, err error)`.
2. Prefix every call with `git -C <dir> -c core.fsmonitor=false -c core.hooksPath=/dev/null -c core.quotePath=false --no-pager`.
3. Set `GIT_OPTIONAL_LOCKS=0`, `GIT_TERMINAL_PROMPT=0` and `GIT_PAGER=cat` on top of `os.Environ()`.
4. Remove `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_EXTERNAL_DIFF` and all `GIT_CONFIG_*` injection variables from the env.
5. Apply a 10-second context timeout.
6. Write stdout to a `capWriter`. It keeps 64 KiB, drains the rest so git never blocks, and sets `truncated`.
7. Add `validRepoPath(p string) (string, error)`. It cleans the path and rejects empty, `.`, absolute, `..`-escaping and `-`-prefixed paths.
8. Add `validSHA(s string) error` with the pattern `^[0-9a-fA-F]{4,40}$`.
9. Pass every caller path after a `--` separator. Pass every revision after `--end-of-options`, except in `blame` (see Decisions).

### 2. Issue resolution

1. Add `(c *Client) issueGitRef(id string) (issueGitRef, error)`. It returns the canonical issue ID, the branch, the worktree path (empty when none), the default branch and the hub.
2. Resolve the issue with `c.GetIssue`, so a short suffix works with a local and a remote transport.
3. Use the branch of `FindWorktreeForIssue(issue.ID)` when a worktree exists.
4. Otherwise, run `for-each-ref refs/heads/` in the hub and match with `branchNameMatchesIssue`. Skip the default branch.
5. Return `"issue <id> has no branch — start it first"` when no branch exists.
6. Run all git commands in `storage.HubRoot()` or the worktree. Never use the process cwd.

### 3. Operations — `internal/exponential/gitread.go`

1. `GitStatus`: run `rev-list --left-right --count <base>...<branch>` in the hub. When a worktree exists, run `status --porcelain=v2 --untracked-files=all` in it and parse it into staged, unstaged and untracked lists.
2. `GitDiff`: compute `merge-base <base> <branch>`. With a worktree, run `diff <mergebase>` in the worktree. Without one, run `diff <mergebase> <branch>` in the hub. Add `--no-ext-diff --no-textconv --no-color`.
3. `GitDiff` with `stat`: add `--numstat` and parse the result into `[]GitDiffFile`. Binary files get `binary: true`.
4. `GitDiff`: add the untracked file names from `status --porcelain=v2` when a worktree exists.
5. `GitLog`: build the range. With an `id`, use `<base>..<branch>`. Without one, use `<base>`. Add `-- <path>` when set.
6. `GitShow`: resolve the SHA with `rev-parse --verify <sha>^{commit}`. Run `show --no-ext-diff --no-textconv --no-color --format=fuller <full-sha> -- <path?>` in the hub.
7. `CodeHistory` without lines: run `log --follow` on `<base> -- <path>`.
8. `CodeHistory` with lines: run `blame --porcelain --no-textconv -L <start>,<end> <base> -- <path>`. Group lines by commit into line ranges. Sort commits newest first.
9. Parse issue IDs from every subject. An ID is `<prefix>-` plus 6 lowercase hex characters, not inside a longer word.
10. In `CodeHistory`, resolve each distinct issue ID to its title and status through the transport. Keep IDs that do not resolve, with an empty title.
11. Default `limit` to 20. Clamp it to 1–100.

### 4. MCP tools — `internal/mcpserver/git_tools.go`

1. Add the input types to `internal/jsonio/input.go`. Use `id` for the issue ID, as `show` and `start` do.
2. Use the `exponential` result types (with JSON tags) as the tool outputs.
3. Add `(t *toolset) registerGitTools(s *mcp.Server)`. Each description says the tool is read-only and names the 64 KiB cap.
4. Add `newStdioServer(cfg)` in `server.go`. It registers the issue tools and the git tools. `Run` uses it.
5. `RegisterTools` (used by `xpo serve`) does not call `registerGitTools`.
6. Each handler calls `t.clientFor(req)` and the matching `Client` method. The text result is a one-line summary.

### 5. Chat surface

1. Add `path` and `sha` to `mcpToolDetailKeys` in `internal/exponential/claudestream.go`.
2. Add verbs for the new tools to `TOOL_VERBS` in `web/src/components/Agent/chat-thread-utils.ts`. Shorten a `git_show` SHA to 12 characters.
3. Update `internal/chat/prompt.md`: the agent can inspect git with the read-only tools, and pairs `code_history` with `rationale` or `spec`.

### 6. Tests (written first)

1. `internal/exponential/gitread_test.go`: status, diff (worktree and branch-only), stat with a binary file, diff by path, truncation, log by issue and by path, limit clamping, show, blame grouping, issue-ID parsing.
2. Security tests: rejected paths, SHAs and line ranges. A repo with `diff.external`, textconv and `core.fsmonitor` scripts: plain git runs all three, the tools run none. Index mtimes do not change and no `index.lock` appears.
3. Remote transport test: an `httptest` server answers `GetIssue`. `GitDiff` returns the local worktree diff.
4. `internal/mcpserver/git_tools_test.go`: the stdio server lists the git tools and `RegisterTools` does not; `git_diff` end to end over an in-memory MCP session; handler tests and the no-branch error.
5. Tool-call detail keys and `toolSummary` cases for the new tools.

## Decisions

| Question | Decision | Rationale |
|---|---|---|
| Raw flags or fixed commands | Fixed commands, validated arguments | Removes `--output`, `--ext-diff` and `-c` attack paths. |
| Where git runs | Always the local hub, whatever the transport | Code lives on the user's machine. Remote mode only changes where issue data comes from. |
| `xpo serve` `/mcp` | Does not offer the git tools | The server's checkout does not hold the users' worktrees. |
| Remote-mode chat | No wiring change in this issue | The chat reaches `<remote>/mcp` in remote mode, so it gets the tools after xpo-6dd7a0 switches it to stdio. No throwaway stopgap. |
| Issue argument name | `id`, not `issue_id` | Matches `show`, `start` and `history`. The chat transcript already shows `id` as the call detail. |
| Output types | `exponential` result types, not `jsonio` copies | `jsonio` mirrors would only copy fields. The results have JSON tags and no other consumer. |
| Diff base | Merge base with the default branch | Shows only the issue's changes, not later changes on the default branch. |
| Untracked files in the diff | Names only | Contents can be large or binary. The agent can `Read` a named file. |
| `code_history` revision | Default branch tip, not the worktree | History answers "why is merged code like this". Blame on the working copy reads uncommitted files. |
| `blame` and `--end-of-options` | Not used for `blame` | `blame` reads `--` after `--end-of-options` as a revision. Its revision is the default branch from config or git, never caller input. |
| Commit dates | Author time, RFC 3339 in UTC | `log` and `blame` then report dates the same way. |
| Line range input | `start_line` and `end_line` integers | Easier to validate than a `"10-20"` string. |
| Locking | No `WithGitLock` | The tools do not write. `GIT_OPTIONAL_LOCKS=0` stops index refreshes. |
| Output cap | 64 KiB per result (confirmed) | Large diffs fill the chat context. `stat` and `path` let the agent narrow the request. |
| Chat surface | In this issue (confirmed) | Transcript verbs and the prompt line are small and make the tools usable. |

## Edge Cases

- **HIGH — remote-mode chat.** Until xpo-6dd7a0 lands, a remote-mode chat does not see the git tools. The CLI's `xpo mcp` (for Claude Code and similar agents) has them in both modes.
- **HIGH — path escape.** The validator rejects `..`, absolute and `-`-prefixed paths. Git reads tree paths, not files, except blame. Blame runs on `<base>`, so it also reads tree content.
- **MEDIUM — merged issue.** The branch is gone after `merge`. `git_status` and `git_diff` return the no-branch error. `git_log {path}` and `code_history` still find the squash commit by its subject prefix.
- **MEDIUM — binary files.** `git diff` prints "Binary files differ". `--numstat` prints `-` counts. The parser sets `binary: true`.
- **MEDIUM — renamed file in `code_history`.** `--follow` tracks renames for the no-lines case. Blame tracks them by default.
- **MEDIUM — no git repo.** The hub is not a git repo. Every tool returns git's error.
- **LOW — `sha` not found.** `git_show` returns "commit <sha> not found".
- **LOW — line range past end of file.** Git blame fails. The tool returns git's stderr.
- **LOW — commits in the same second.** Blame order then follows first appearance in the file.

## Assumptions

- Issue IDs are `<prefix>-` plus 6 lowercase hex characters.
- Squash-merge commit subjects start with `<issue-id>: `, as `merge` writes them.
- The stdio MCP process runs on the machine that holds the hub and the worktrees.

## Related

- xpo-5d307a — remote mode: board and chat `start`/`merge` run in the server's checkout (bug).
- xpo-6dd7a0 — local MCP server works fully in remote mode; chat always uses it.
