# Walkthrough: Start Work without a git repository

## The bug

`StartWork` (`internal/exponential/start.go`) moves the issue to DOING first, and creates the worktree after that, only when `CheckGitRepo()` is true. Outside git it returns success with no worktree. That is on purpose: the CLI can track issues in a folder without git.

The board's Work Mode went through `startWorktree` (`internal/server/chat.go`). It called `StartWork`, and only then noticed the empty worktree and returned "work mode needs a git repository". By then the DOING event had been written. The agent's `start` tool in Chat Mode had the same order: MCP `start` moved the issue, then `ResolveWork` failed with "Could not enter Work Mode".

## The fix: check before acting

We check for git before anything changes, rather than rolling the status back afterwards. A rollback would write a second event and could race with other writers.

### Shared check

`exponential.IsGitRepo(dir)` in `internal/exponential/checks.go` checks with `os.Stat` that `dir/.git` exists, as a directory or as a worktree's `.git` file. `CheckGitRepo()` now calls `IsGitRepo(".")`.

### Server

- `handleStartWorkChat` checks git right after the `CanWork` check. That order keeps remote mode's 501 ("needs a local board") for remote boards. Without git it answers 409 with `errNoGitRepo()`.
- `startWorktree` checks again before `startIssue`, so any other caller is safe too.
- `errNoGitRepo()` says what to do: `Work Mode needs a git repository. Run "git init" in <abs hub path>.` `hubPath()` makes `storage.HubRoot()` absolute; outside git, `HubRoot()` returns `"."`.
- `GET /api/local/mode` now returns `{remote, git, root}`.

### Agent (Chat Mode)

`Runner.profileFor` in `internal/chat/mode.go` sets Chat Mode's `CanWork` to `cfg.CanWork && IsGitRepo(cfg.Dir)`. With `CanWork` false, the profile denies `mcp__xpo__start`, the same way remote mode does. `prompt.md` now tells the agent that Work Mode needs a local board and a git repository.

This runs on every turn. That is one `stat` a few times per turn, next to a turn that starts an agent CLI. In review we kept it over caching: if we checked only at startup, the "run git init" hint would not work until the board restarts.

### Board UI

- `useLocalMode()` (`web/src/api/queries.ts`) reads `/api/local/mode` with `refetchOnWindowFocus: "always"`. The app's default is `staleTime: Infinity`, so a plain `true` would never refetch. After a `git init` in a terminal, the board picks it up when the user switches back. `useGitActions()` now builds on it.
- `useStartWorkChat()` returns `noGitRoot` when an agent is available and git is missing. With an agent, Start Work opens Work Mode, which needs git. Without one, Start Work only changes the status, which still works.
- `PropertySidebar` adds `noGitRoot` to `startDisabled`. `StartWorkBlockedReasons` lists the git hint first, before BACKLOG and the blockers.
- `IssueChatsList` passes `noGitRoot` to `enterWorkBlocker`, which returns the same hint. The text lives in one place: `noGitHint()` in `web/src/components/Agent/issue-chats.ts`.

## Unchanged

- `StartWork`, the CLI `xpo start`, and `POST /api/issues/{id}/start` still start an issue outside git (status only). `TestStartWork_NonGitRepo` still passes.

## Tests

Test fixtures that need Work Mode (`setupWorkServer`, `newModeRunner`) now create a `.git` directory, which is all `IsGitRepo` looks for.

## Acceptance criteria

- [x] Outside a git repo, `POST /api/local/issues/{id}/work-chat` on a PLANNED issue returns 409 with the "Run git init in <path>" message, and the issue stays PLANNED. **Evidence:** `TestChat_WorkChatWithoutGitRepoChangesNothing` (409, hint, no start, no chat) and `TestStartWorktree_WithoutGitRepoKeepsStatus` (real start path, status stays PLANNED).
- [x] Inside a git repo, Start Work behaves as before. **Evidence:** every existing `TestChat_WorkChat*` test passes with the `.git` fixture.
- [x] `GET /api/local/mode` returns `git` and `root`. **Evidence:** `TestLocalMode_ReportsGitRepo`, which flips `git` false→true by creating `.git`.
- [x] Outside git, a Chat Mode turn's profile has `CanWork` false: `start` is denied and never asks. Inside git it is true. **Evidence:** `TestRunner_ChatModeCanWorkNeedsGitRepo`.
- [x] With an agent available and no git repo, Start Work and "Enter work mode" are disabled, with a tooltip saying Work Mode needs git and where to run `git init`. **Evidence:** `PropertySidebar Start Work without a git repository > disables Start Work…`, `StartWorkBlockedReasons > says Work Mode needs git…`, and `enterWorkBlocker > says Work Mode needs a git repository…`.
- [x] Without an agent, Start Work stays enabled outside git and only moves the issue to DOING. **Evidence:** `keeps Start Work enabled without an agent`.
- [x] `make test` passes. **Evidence:** exit 0 (lint, 1226 frontend tests, all Go packages).
