# Start Work must not change state when the project has no git repo

## What

When the board's project is not a git repository, Start Work (Work Mode) moves the issue to DOING and then fails with "work mode needs a git repository". The issue is left DOING, with no worktree and no chat. The Chat Mode agent's `start` tool has the same problem: the issue goes to DOING, then "Could not enter Work Mode".

## Why

`POST /api/chat/work/{id}` calls `startWorktree` (`internal/server/chat.go:403`). It runs `StartWork`, which moves the issue to DOING first. Outside git, `StartWork` succeeds without a worktree, which is correct for the CLI. Only afterwards does `startWorktree` see the empty worktree and return the error. By then the status has already changed. The agent path does the same: MCP `start` moves the issue, and then `ResolveWork` fails.

The UI also offers Start Work with no hint that Work Mode needs git.

## How

### Backend
1. Add `exponential.IsGitRepo(dir string) bool`, which checks for `dir/.git` (a file or a directory). `CheckGitRepo()` becomes `IsGitRepo(".")`.
2. Add an `errNoGitRepo()` helper in `internal/server`. Its text is `Work Mode needs a git repository. Run "git init" in <abs hub path>.`
3. `handleStartWorkChat`: after the `WorkChat` short-circuit and the `CanWork` check (remote mode still answers 501), a board without a git repo answers 409 `errNoGitRepo()`, before anything changes state.
4. `startWorktree` also checks git before `startIssue`, as a second guard.
5. `GET /api/local/mode` adds `"git": bool` and `"root": "<abs hub path>"`.
6. Chat runner: `profileFor` builds Chat Mode with `CanWork: r.cfg.CanWork && exponential.IsGitRepo(r.cfg.Dir)`. Outside git, Chat Mode then denies `start`, the same way remote mode does. The check runs every turn, so `git init` takes effect without restarting the board.
7. `internal/chat/prompt.md`: "If `start` is not available, tell the user that Work Mode needs a local board **and a git repository**."
8. `StartWork` itself is unchanged. The CLI and `POST /api/issues/{id}/start` still start an issue outside git (status only), as `TestStartWork_NonGitRepo` covers.

### Frontend
1. `fetchLocalMode()` returns `{ remote, git, root }`. Add a `useLocalMode()` hook. The query refetches on window focus (`refetchOnWindowFocus: "always"`, since the app default is `staleTime: Infinity`), so a `git init` shows up when the user comes back to the board. `useGitActions()` keeps its meaning (local board).
2. `PropertySidebar`: with an agent available (`canChat`) and `git === false`, Start Work is **disabled**. It reuses the disabled-button tooltip: `StartWorkBlockedReasons` gets a `noGitRoot` reason, shown first, saying "Work Mode needs a git repository. Run git init in <root>". The text comes from `noGitHint(root)` in `web/src/components/Agent/issue-chats.ts`, and `useStartWorkChat()` returns `noGitRoot` when an agent is available and git is missing.
3. Without an agent, Start Work only starts the issue (status change, no worktree), which works outside git, so it stays enabled.
4. `IssueChats` "Enter work mode" is disabled the same way: `enterWorkBlocker(chat, noGitRoot)` returns the hint.

## Acceptance criteria
- [ ] Outside a git repo, `POST /api/chat/work/{id}` on a PLANNED issue returns 409 with the "Run git init in <path>" message, and the issue stays PLANNED.
- [ ] Inside a git repo, Start Work behaves as before.
- [ ] `GET /api/local/mode` returns `git` and `root`.
- [ ] Outside git, a Chat Mode turn's profile has `CanWork` false: `start` is denied and never asks. Inside git it is true.
- [ ] With an agent available and no git repo, Start Work and "Enter work mode" are disabled, with a tooltip saying Work Mode needs git and where to run `git init`.
- [ ] Without an agent, Start Work stays enabled outside git and only moves the issue to DOING.
- [ ] `make test` passes.

## Decisions
- **Agent path fixed here** (user's choice), by gating `CanWork` on git rather than by rolling back the status after the fact.
- **Check before acting, no rollback.** A pre-check is simpler than undoing a status change, and it leaves no events behind.
- **Live git detection.** Both the runner and the board UI re-check git, so the hint "run git init" works without restarting the board.
