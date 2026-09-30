# Spec: "Open in VS Code" action button for local worktrees

## What

In the issue detail view, when the issue has a git worktree on the machine running `xpo board`, show an **Open** button with a VS Code icon in the `TopBar`'s right slot, before the `n / total` counter. Clicking it follows `vscode://file/<abs-worktree-path>`, which opens the folder in VS Code.

## Why

Jumping from the board to the code for an in-flight issue currently means finding `.xpo/worktrees/<branch>` by hand. One click is faster.

## Key constraint: resolve the path locally

`xpo board` always runs on the user's machine, from their project folder. In remote mode (`cfg.Remote.URL` set) it reverse-proxies **all** `/api/*` to the remote server (`router.go`). If the worktree path were part of the issue payload (e.g. `BranchStats`), the remote server would compute it from *its* checkout: the wrong machine, so the path would be missing or wrong.

So the path must be answered by the local `xpo board` process in both modes.

## How

### Backend

- New route `GET /api/local/issues/{id}/worktree`, registered **outside** the local/proxy `if/else` in `SetupRoutes`, so it is served locally in both modes. Go's `ServeMux` picks the more specific pattern over the proxy's `GET /api/{path...}` catch-all.
- Not registered when `s.Headless` (`xpo serve`): a headless server is by definition not on the viewer's machine.
- The handler uses `exponential.WorktreeList()` and returns the first entry whose branch matches the issue via `branchNameMatchesIssue(branch, id)`. It skips the main worktree, which `git worktree list` always reports first, so the hub is never returned, even in branch mode. Skipping by position avoids symlink path mismatches (macOS `/var` vs `/private/var`).
  - Matching by **issue ID**, not by `branch_stats.branch`, because in remote mode `branch_stats` comes from the remote and may be absent for unpushed branches.
- Add an exported helper `exponential.FindWorktreeForIssue(issueID string) (WorktreeEntry, bool)` so the logic is testable without HTTP.
- Response `200 {"path": "...", "branch": "..."}`, or `404` when no worktree exists. Does not validate that the issue exists (in remote mode the local process may have no issue data; the lookup is purely git-based).

### Frontend

- `fetchLocalWorktree(issueId): Promise<{path, branch} | null>` in `api/client.ts` (404 → `null`).
- `IssueDetail` fetches it when the issue ID or status changes (`start` creates the worktree, `merge` removes it) and renders an `<a href={vscodeUrl(path)}>` with an inline VS Code SVG icon (lucide has no brand icons) and the label "Open".
- Styling: an outlined toolbar button that matches `IconButton` (the Filter / menu-trigger style). It reuses `iconButtonClass()` with overrides for auto width, horizontal padding and a text label, so the border, surface and hover colors stay in sync with the other toolbar buttons. It uses the shared `Tooltip` ("Open worktree in VSCode").
- `vscodeUrl(path)` = `"vscode://file"` + the path with each segment `encodeURIComponent`-encoded (so `#`/`?` survive), a leading `/` ensured, and a Windows drive segment (`C:`) left unencoded (Windows paths like `C:\...` → `/C:/...`). Pure function, unit-tested.
- Hidden when the fetch returns null or errors.

## Out of scope

- A configurable editor (Cursor, Zed, JetBrains). The URL builder is isolated so this is easy to add later.
- Showing the button on board/backlog cards.

## Acceptance criteria

- [ ] `GET /api/local/issues/{id}/worktree` returns the local worktree path/branch for an issue with a worktree, and 404 otherwise.
- [ ] The route is served locally when `ProxyURL` is set (not forwarded to the remote).
- [ ] The route is not registered in headless (`xpo serve`) mode.
- [ ] The main worktree / default branch is never returned.
- [ ] Issue detail shows an outlined "Open" toolbar button (same style as `IconButton`) with a VS Code icon and the tooltip "Open worktree in VSCode" when a worktree exists; clicking opens `vscode://file/<path>`.
- [ ] No button when no worktree exists.
- [ ] Go unit tests for `FindWorktreeForIssue` and the route (local + proxy mode); a vitest test for `vscodeUrl`.
- [ ] `make test` passes.
