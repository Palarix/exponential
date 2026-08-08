## What changed

`MergeIssue` in `internal/exponential/merge.go` now stages the issue's artifact directory alongside `issues.db` before committing the merge.

## The bug

Specs, walkthroughs, and generic artifacts are written to `.xpo/artifacts/<issue-id>/` on the hub's working tree via the MCP server. The merge flow only staged `.xpo/issues.db` (line 132), so artifacts were left as untracked files on `main` after every merge.

## The fix

One line added after the existing `issues.db` staging:

```go
exec.Command("git", "-C", storage.HubRoot(), "add", filepath.Join(".xpo", "artifacts", issue.ID)).Run()
```

This follows the same fire-and-forget pattern as the `issues.db` staging — no error check, because `git add` on a nonexistent path is a no-op. The `path/filepath` import was added to support `filepath.Join`.

## Why this approach

- Stages the whole `<issue-id>/` directory rather than individual files, catching spec + walkthrough + any generic artifacts in one call.
- No existence check needed — `git add` silently does nothing if the path doesn't exist.
- Matches the established error-handling pattern (the `issues.db` `git add` on line 132 also ignores its error).
