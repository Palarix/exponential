# Walkthrough: Fix merge abort after squash conflict

## What changed

### Production: `merge.go`

**`runGitMerge`** — now captures `CombinedOutput()` and returns a `mergeError` type that carries the git output. Previously stdout/stderr were discarded, so conflicting filenames were lost from the error.

**Abort path** — tries `git merge --abort` first (works for `--no-ff` which sets `MERGE_HEAD`), then falls back to `git reset --merge` (works for squash which has no `MERGE_HEAD`). The git merge output is appended to the error message so users see which files conflict.

### Test: `merge_test.go`

`TestMergeIssue_Conflict` now verifies the full contract:
- `"merge failed"` in error
- `"Do NOT stash"` warning in error
- Conflicting filename `"shared.txt"` appears in error (via captured git output)
- No `UU`/`AA`/`DD` conflict markers in `git status` after abort
- `shared.txt` is not modified after abort
