# Walkthrough: read-only git MCP tools for chat sessions

## What was built

The local stdio MCP server (`xpo mcp`) has five new read-only tools: `git_status`, `git_diff`, `git_log`, `git_show` and `code_history`. The chat agent can now answer "what changed on this issue?" and "why is this code like this?". It still has no Bash.

The caller supplies only an issue ID, a repo path, a commit SHA, a line range and a limit. xpo builds every git command line itself.

## Why not allow Bash for git

Git commands that look read-only can still write files or run programs:

- `--output=<file>` makes `diff` and `log` write a file.
- `diff.external` and textconv drivers run programs from the repo config.
- `core.fsmonitor` makes `git status` run a program.
- `git -c …` sets any config, including the three settings above.

A prefix rule such as `Bash(git diff:*)` blocks none of these. A fixed command line with validated arguments blocks all of them.

## The design rule: code runs where the code is

During spec review we traced how remote mode works. The chat agent always runs on the user's machine, but in remote mode its MCP tools point at `<remote>/mcp`, so the tool calls run on the server. For git that is wrong, because the worktrees are on the user's machine.

The rule we agreed: **anything that touches code runs on the user's machine, and issue data lives wherever the transport points.** The git tools follow it:

- They resolve the issue through `c.GetIssue`, which works with a local or a remote transport.
- They always run git in the local hub (`storage.HubRoot()`) or the issue's worktree. They never use the process cwd.
- Only the stdio server registers them. `newStdioServer` in `internal/mcpserver/server.go` calls `registerGitTools`. `RegisterTools`, which `xpo serve` uses for its `/mcp`, does not, because the server's checkout does not hold the users' worktrees.

The same trace found older remote-mode bugs. They are filed separately:

- **xpo-5d307a:** the board's Start Work / Merge buttons and the chat's `start` / `merge` act on the server's checkout. CLI `xpo merge` fails in remote mode.
- **xpo-6dd7a0:** make the local `xpo mcp` work fully in remote mode (specs, walkthroughs, artifacts, rationale search, agent attribution), then point the chat at it. A chat in remote mode gets the git tools after that issue lands. We did not add a stopgap here.

## How the pieces fit

### `internal/exponential/gitread.go`

**`readGit(dir, args...)`** is the single place that runs git. Each call:

- Prefixes `git -C <dir> -c core.fsmonitor=false -c core.hooksPath=/dev/null -c core.quotePath=false --no-pager`.
- Sets `GIT_OPTIONAL_LOCKS=0`, so `status` and `diff` do not refresh or lock the index. It also sets `GIT_TERMINAL_PROMPT=0` and `GIT_PAGER=cat`.
- Removes `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_EXTERNAL_DIFF` and the `GIT_CONFIG_*` injection variables from the environment. These would point git at another repo or inject config.
- Times out after 10 seconds.
- Writes stdout to a `capWriter`. It keeps 64 KiB, drains the rest so git never blocks on a full pipe, and reports `truncated`. Patch text then ends with a marker line that tells the agent to narrow the request with `path` or `stat`.

**Argument validation:**

- `validRepoPath` cleans the path and rejects empty, `.`, absolute, `..`-escaping and `-`-prefixed paths.
- `validSHA` accepts 4–40 hex characters only, so `HEAD`, branch names and ranges are rejected.
- Caller paths always come after `--`. Revisions come after `--end-of-options`. The one exception is `blame`: it reads `--` after `--end-of-options` as a revision. Blame's revision is the default branch, from config or git, never caller input, so the guard is not needed there.
- Every diff-producing command also passes `--no-ext-diff --no-textconv --no-color`.

**`issueGitRef`** finds the issue's branch:

1. It prefers the branch of the issue's linked worktree (`FindWorktreeForIssue`).
2. Otherwise it scans `refs/heads/` in the hub with `branchNameMatchesIssue`, skipping the default branch.
3. With no branch, it returns `issue <id> has no branch — start it first`.

It does not use `findLocalBranch` or `CurrentBranch`, because both run in the process cwd.

**Operations:**

| Method | Git commands | Notes |
|---|---|---|
| `GitStatus` | `rev-list --left-right --count base...branch` in the hub; `status --porcelain=v2` in the worktree | Porcelain v2 gives a two-letter code per file. Changes in the first column count as staged, in the second as unstaged. |
| `GitDiff` | `merge-base`, then `diff <mergebase>` in the worktree (includes uncommitted changes), or `diff <mergebase> <branch>` in the hub when there is no worktree | `stat` adds `--numstat`; binary files get `binary: true`. Untracked files are listed by name only. |
| `GitLog` | `log base..branch` for an issue, otherwise `log base`, optionally `-- path` | The fields are split with `\x1f` and `\x1e`, so subjects can hold any character. |
| `GitShow` | `rev-parse --verify <sha>^{commit}`, then `show --format=fuller <full-sha>` | The result carries the full SHA. |
| `CodeHistory` | `log --follow base -- path`, or `blame --porcelain -L s,e base -- path` | Blame output is grouped by commit into line ranges and sorted newest first. |

`parseIssueIDs` finds `<prefix>-` plus 6 lowercase hex characters that are not inside a longer word. It trims a trailing `-` from the configured prefix (tests use `test-`, real configs store it bare) and defaults to `issue`. `CodeHistory` resolves each distinct ID once through the transport, which pairs it with `rationale` and `spec`. IDs that do not resolve (deleted, or from another project) stay in the list with no title.

Commit dates use author time as RFC 3339 in UTC, so `log` and `blame` match. `limit` defaults to 20 and is clamped to 1–100.

### `internal/mcpserver/git_tools.go`

These are thin handlers. Each one calls `t.clientFor(req)` and the matching `Client` method, and returns a one-line text summary. The `exponential` result types (with JSON tags) are the tool outputs: copies in `jsonio` would have added only field copying. The input types are in `internal/jsonio/input.go`, and the issue argument is `id`, the same as `show` and `start`.

### Chat surface

- `internal/exponential/claudestream.go`: `path` and `sha` are now MCP tool-call detail keys, after `id`. So `git_diff {id, path}` shows the issue, and `code_history {path}` shows the file.
- `web/src/components/Agent/chat-thread-utils.ts`: transcript verbs ("Checked the git status of", "Read the diff of", "Listed commits of", "Read commit", "Traced the history of"). A `git_show` SHA is cut to 12 characters.
- `internal/chat/prompt.md`: one line tells the agent about the tools and to pair `code_history` with `rationale` or `spec`.

## Things a future reader should know

- The tools take no lock (`WithGitLock`) because they never write. `GIT_OPTIONAL_LOCKS=0` is what keeps `status` from touching the index while an agent works in the same worktree.
- `code_history` reads the default branch tip, not the worktree. It answers "why is merged code like this". Blame on the working copy would read uncommitted files.
- After `merge`, the issue's branch is gone. `git_status` and `git_diff` then return the no-branch error, but `git_log {path}` and `code_history` still find the squash commit by its `<issue-id>: ` subject.
- Work mode (an agent working through an issue from the board) will need xpo-5d307a and xpo-6dd7a0 first.

## Acceptance criteria

- [x] `git_status {id}` returns the branch, worktree path, ahead/behind and staged/unstaged/untracked files. *Evidence: `TestGitStatus_Worktree` (1 ahead, 1 behind, one file in each list); `TestGitStatusTool`; smoke test on this issue's own worktree.*
- [x] `git_diff` returns the merge-base diff including uncommitted changes, and lists untracked files. *Evidence: `TestGitDiff_WorktreeIncludesUncommitted` (also asserts a later commit on main is excluded); `TestGitDiffTool_EndToEnd`.*
- [x] `git_diff` with `stat` returns per-file counts and no patch. *Evidence: `TestGitDiff_Stat`, including a binary file.*
- [x] `git_diff` on a branch without a worktree diffs to the branch tip. *Evidence: `TestGitDiff_BranchWithoutWorktree`.*
- [x] `git_log` returns sha, subject, author, date and issue IDs. *Evidence: `TestGitLog_Issue`, `TestGitLog_PathOnDefaultBranch`, `TestGitLog_LimitClamp`, `TestParseIssueIDs`.*
- [x] `git_show` returns metadata, message and patch. *Evidence: `TestGitShow`.*
- [x] `code_history` returns commits and issues for a file or line range. *Evidence: `TestCodeHistory_File` (resolved and unresolved IDs), `TestCodeHistory_Lines` (line ranges per commit); smoke test traced `internal/chat/args.go` lines 36–50 to xpo-56ae3a.*
- [x] Bad paths are rejected. *Evidence: `TestGitRead_RejectsBadPaths` covers every tool; `TestValidRepoPath_CleansGoodPaths`.*
- [x] Bad SHAs are rejected. *Evidence: `TestGitShow_RejectsBadSHA` (`HEAD`, `main`, `--output=…`, ranges, too short).*
- [x] An issue with no branch returns a named error. *Evidence: `TestGitStatus_NoBranch`, `TestGitTools_NoBranch`.*
- [x] Output is capped at 64 KiB with `truncated` and a marker line. *Evidence: `TestGitDiff_Truncates`.*
- [x] Repo-configured programs never run. *Evidence: `TestGitRead_RunsNoRepoPrograms` first proves plain git runs the `diff.external`, textconv and `core.fsmonitor` scripts, then runs every tool and finds no marker file.*
- [x] No index writes or lock files. *Evidence: `TestGitRead_DoesNotWriteIndex` makes the index stale, runs status and diff, and checks the hub and worktree index mtimes and `index.lock`.*
- [x] A remote transport resolves the issue remotely and diffs locally. *Evidence: `TestGitDiff_RemoteTransport` with an `httptest` server.*
- [x] `xpo serve` does not offer the git tools. *Evidence: `TestGitTools_OnlyOnStdioServer` lists tools over an in-memory MCP session for both servers.*
- [x] The transcript shows readable summaries. *Evidence: `toolSummary` "describes the git tools in words"; `TestToolDetail_GitTools`.*
- [x] The chat prompt mentions the git tools. *Evidence: `internal/chat/prompt.md`.*
- [x] `make test` passes. *Evidence: lint clean, 707 frontend tests, all Go packages ok.*
