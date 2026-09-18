## Walkthrough: Add --json input/output to merge command

## What changed

One file: `cmd/exponential/merge.go`.

The `merge` command now supports `--json` mode, accepting the same payload shape as the
MCP `merge` tool and emitting the same output.

## How it works

### JSON path (`runMergeJSON`)

```
stdin → jsonio.DecodeStrict → MergeToolInput → validate id + strategy
→ client.ResolveReviewIssue → check branch exists → HubCleanForMerge
→ client.MergeIssue → MergeOutput JSON
```

The function mirrors the MCP handler in `tools.go` line-for-line:

1. Decode `MergeToolInput` from stdin
2. Validate `id` is present, `strategy` is valid (empty/squash/merge/ff)
3. Resolve the issue and verify it has a branch
4. Run `HubCleanForMerge` to ensure the hub checkout is clean
5. Call `client.MergeIssue` with options derived from the input
6. Emit `MergeOutput{ID, MergeSHA, Messages}`

Branch deletion follows the MCP convention: `DeleteBranch: !input.KeepBranch` — branches
are deleted by default unless the caller explicitly sets `keep_branch: true`.

The `commit_message` field passes through to `MergeOptions.CommitMessage` — when empty,
the client auto-generates a message (same as MCP).

### Dispatch

`runMerge` checks `mergeJSONFlag` first and dispatches to `runMergeJSON()` before any
interactive logic. This keeps the JSON path completely separate from the interactive
path — no terminal output, no strategy prompts, no branch deletion prompts.

### Non-JSON path

Completely unchanged — interactive prompts, terminal summary, all existing flags work
as before.
