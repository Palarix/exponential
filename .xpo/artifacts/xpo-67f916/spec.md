# Spec: Fix merge abort after squash conflict

## What

Two fixes:
1. After a failed squash merge, use `git reset --merge` instead of `git merge --abort` (which is a no-op for squash because no `MERGE_HEAD` exists)
2. Capture merge stderr so conflicting filenames appear in the error message

## How

### merge.go changes

**`runGitMerge`** — capture `CombinedOutput()` instead of discarding it. Return a custom error type that includes the output so the caller can surface it.

**Abort path (line 163-165)** — try `git merge --abort` first (works for `--no-ff`), then fall back to `git reset --merge` (works for squash). Include the captured merge output in the error message.

### Test changes

**`TestMergeIssue_Conflict`** — restore the full contract:
- Verify the conflicting filename (`shared.txt`) appears in the error
- Verify the working tree is clean after abort

## AC

- [ ] Squash conflict leaves a clean working tree after abort
- [ ] Conflicting filename appears in the error message
- [ ] `--no-ff` conflicts also abort cleanly
- [ ] `make test` passes
