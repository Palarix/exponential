# Walkthrough: "Open in VS Code" action button for local worktrees

## What was built

When an issue has a git worktree on your machine, the issue detail view's top bar shows an outlined **Open** button with a VS Code icon, next to the `n / total` counter. Clicking it opens the worktree folder in VS Code through a `vscode://file/<path>` link. The tooltip reads "Open worktree in VSCode".

## The one design decision that matters: who answers "where is the worktree?"

The obvious design was to add a `worktree_path` field to `BranchStats`, next to the branch name that's already in the issue JSON. We rejected it because of how remote mode works:

- `xpo board` always runs on the user's machine, from their project folder, and serves the web UI.
- When `cfg.Remote.URL` is set, `SetupRoutes` (`internal/server/router.go`) reverse-proxies **every** `/api/*` request to the remote server. The issue JSON, including `branch_stats`, is then built by the remote server's `git`, in the remote server's checkout.

A worktree path in the issue payload would therefore be looked up on the wrong machine. It would be missing (the worktree was created locally by `xpo start`) or point at the server's disk. The folder is local, but the lookup would not be.

The fix is a **local-only** route that the `xpo board` process answers itself:

```go
if !s.Headless {
    mux.HandleFunc("GET /api/local/issues/{id}/worktree", s.handleGetLocalWorktree)
}
```

It is registered *before* and *outside* the local/proxy `if/else`. Go 1.22's `ServeMux` picks the most specific matching pattern, so `/api/local/issues/{id}/worktree` beats the proxy's `GET /api/{path...}` catch-all. The `/api/local/` prefix marks "this describes the viewer's machine" and is the natural place for any future local-only endpoints.

It is **not** registered when `Headless` (`xpo serve`): a headless server is by definition not the viewer's machine, so exposing its paths would reintroduce the bug.

## How the pieces fit

### Backend

- **`exponential.FindWorktreeForIssue(issueID)`** (`internal/exponential/git.go`) scans `git worktree list --porcelain` via the existing `WorktreeList()` and returns the first entry whose branch matches the issue, using the existing `branchNameMatchesIssue` (a branch path segment starts with the issue ID).
  - It matches by **issue ID**, not by `branch_stats.branch`, because in remote mode `branch_stats` comes from the remote and is absent for unpushed branches.
  - It **skips the first entry**. Git always lists the main working tree first, so the hub checkout is never returned, even in branch mode where the hub has the issue branch checked out. We skip by position rather than comparing paths against `storage.HubRoot()`, because symlinked temp dirs (macOS `/var` → `/private/var`) make path comparison fragile.
- **`handleGetLocalWorktree`** (`internal/server/handlers.go`) returns `200 {"path","branch"}` or `404`. It does not check that the issue exists: in proxy mode the local process has no issue data, and the lookup is purely git-based.

### Frontend

- **`fetchLocalWorktree(issueId)`** (`web/src/api/client.ts`) maps 404 to `null`.
- **`vscodeUrl(path)`** (`web/src/utils/editor.ts`) is a pure function. It normalizes backslashes, ensures a leading `/`, and `encodeURIComponent`s each path segment, leaving a Windows drive segment (`C:`) as is. We encode per segment rather than using `encodeURI`, because `encodeURI` leaves `#` and `?` alone, and they would truncate the URL.
- **`VSCodeIcon`** (`web/src/components/ui/VSCodeIcon.tsx`) is the monochrome simple-icons mark (CC0). lucide-react ships no brand icons.
- **`IssueDetail`** fetches the worktree whenever `issue.id` **or `issue.status`** changes: `start` creates the worktree and `merge` removes it, so the button appears and disappears without a reload. A `cancelled` flag guards against a stale response when you navigate quickly between issues.

### Styling (changed during review)

The first version was a borderless ghost link. On review it became an outlined toolbar button matching `IconButton` (the Filter / menu-trigger style). Rather than duplicating classes, it calls `iconButtonClass(false, "w-auto px-2 gap-1.5 mr-2 text-xs")`. `cn` uses tailwind-merge, so `w-auto` replaces `w-7` while the border, surface and hover tokens come from the shared base. If the toolbar button style changes, this button follows. It uses the shared radix `Tooltip`, not a native `title`.

## Acceptance criteria

- [x] `GET /api/local/issues/{id}/worktree` returns the local worktree path/branch for an issue with a worktree, and 404 otherwise. Evidence: `TestLocalWorktree_Found`, `TestLocalWorktree_NotFound` (`internal/server/handlers_test.go`).
- [x] The route is served locally when `ProxyURL` is set (not forwarded to the remote). Evidence: `TestLocalWorktree_ServedLocallyInProxyMode` asserts that the backend stub is never hit and that the local path is returned.
- [x] The route is not registered in headless (`xpo serve`) mode. Evidence: `TestLocalWorktree_NotRegisteredWhenHeadless`.
- [x] The main worktree / default branch is never returned. Evidence: `TestFindWorktreeForIssue_SkipsHubCheckout` (the hub is on the issue branch and is still not returned).
- [x] Issue detail shows an outlined "Open" toolbar button (same style as `IconButton`) with a VS Code icon and the tooltip "Open worktree in VSCode" when a worktree exists; clicking opens `vscode://file/<path>`. Evidence: `IssueDetail.tsx` top bar; the class composition is covered by the new `IconButton.test.ts` case; user-tested and approved.
- [x] No button when no worktree exists. Evidence: `fetchLocalWorktree` returns `null` on 404, and the button renders only when `worktree` is non-null.
- [x] Go unit tests for `FindWorktreeForIssue` and the route (local + proxy mode); a vitest test for `vscodeUrl`. Evidence: `git_test.go` (3 tests), `handlers_test.go` (4 tests), `web/src/utils/editor.test.ts` (POSIX, special-character encoding, Windows).
- [x] `make test` passes. Evidence: exit 0; 369 frontend tests plus the full Go suite.

## Future extensions

- **Configurable editor** (Cursor, Zed, JetBrains): `vscodeUrl` is the only place that knows the scheme. Add an `editor` config key and a scheme map.
- **More top-bar actions**: follow the same pattern, an `<a>` or `<button>` styled with `iconButtonClass` and a label override, plus any machine-local data through `/api/local/*`.
