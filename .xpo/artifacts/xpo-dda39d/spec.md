## What

Stage `.xpo/artifacts/<issue-id>/` alongside `.xpo/issues.db` during the merge commit in `MergeIssue`.

## Why

Specs, walkthroughs, and generic artifacts are written to the hub's working tree throughout the workflow via the MCP server. The merge flow only stages `issues.db`, leaving artifacts as untracked files on `main`. They should be part of the merge commit so the issue's full record is captured in git history.

## Acceptance Criteria

- After `MergeIssue`, the merge commit includes any files under `.xpo/artifacts/<issue-id>/`.
- If no artifact directory exists for the issue, no error — the `git add` is a no-op.
- Existing behavior for `issues.db` staging unchanged.
- `make test` passes.

## Flow

In `internal/exponential/merge.go`, `MergeIssue`, after line 132:

```go
exec.Command("git", "-C", storage.HubRoot(), "add", ".xpo/issues.db").Run()
```

Add:

```go
artifactDir := filepath.Join(".xpo", "artifacts", issue.ID)
exec.Command("git", "-C", storage.HubRoot(), "add", artifactDir).Run()
```

`git add` on a nonexistent path is a no-op (exits non-error with the path ignored), so no guard needed.

## Decisions

- **Stage the whole `<issue-id>/` directory, not individual files**: simpler, catches spec + walkthrough + any generic artifacts in one call.
- **No error check on the artifact `git add`**: matches the existing pattern for `issues.db` staging (line 132 ignores the error too). If the directory doesn't exist, `git add` silently does nothing.
