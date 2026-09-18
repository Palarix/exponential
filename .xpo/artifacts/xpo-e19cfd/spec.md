## Spec: Add --json input/output to merge command

## What

Add a `--json` flag to the `merge` CLI command that reads `jsonio.MergeToolInput` from
stdin and emits `jsonio.MergeOutput` on success / `jsonio.ErrorOutput` on failure.

## Why

The `merge` command is the last mutation command without `--json` support in the epic.

## How

Single file change: `cmd/exponential/merge.go`.

### Flow (--json path)

```
stdin → jsonio.DecodeStrict → MergeToolInput → validate id + strategy
→ client.ResolveReviewIssue → check branch exists → HubCleanForMerge
→ client.MergeIssue → MergeOutput JSON
```

No interactive prompts, no terminal summary — all params come from the payload.

### Steps

1. Add `mergeJSONFlag bool` and register `--json` flag in `init()`.
2. Add a `--json` branch at the top of `runMerge`, before the existing logic:
   - Read stdin via `readStdinExplicit()`
   - Decode into `jsonio.MergeToolInput` via `jsonio.DecodeStrict`
   - Validate: `id` required; `strategy` must be empty/squash/merge/ff
   - Mirror the MCP handler's logic: resolve issue, check branch, hub clean, merge
   - `DeleteBranch: !input.KeepBranch` (matches MCP default: delete unless told otherwise)
   - `CommitMessage` passed through from input
   - Emit `jsonio.MergeOutput{ID, MergeSHA, Messages}` as indented JSON
   - All errors use `exitJSONError`
3. Add imports: `jsonio`, `encoding/json`.

### Input schema (MergeToolInput)

```json
{
  "id": "xpo-123",
  "strategy": "squash",
  "commit_message": "optional custom message",
  "keep_branch": false
}
```

Only `id` is required. `strategy` defaults to squash. `keep_branch` defaults to false
(branch is deleted).

### Non-goals

- No changes to interactive prompts or terminal output in non-JSON mode.
- The `--no-wt` flag is not in the MCP schema — in JSON mode the worktree config
  is used as-is (same as MCP).

## Acceptance Criteria

- `echo '{"id":"xpo-123"}' | xpo merge --json` works with the MCP input schema
- Output matches `jsonio.MergeOutput` schema
- Errors match `jsonio.ErrorOutput` schema
- `make test` passes
