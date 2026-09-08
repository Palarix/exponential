# Walkthrough: Server write-path handler tests

## What was built

New file `internal/server/write_handlers_test.go` with 11 tests covering previously untested HTTP handlers.

## Test inventory

| Handler | Tests | What's verified |
|---|---|---|
| `handleGetArtifact` | 2 | Not found (404), found with correct content (200) |
| `handleListInstances` | 1 | Returns JSON array (200) |
| `handleGetCycles` | 1 | Enabled config returns cycle list with entries |
| `handleCycleProgress` | 1 | Disabled cycles returns 400 |
| `handleTimeline` | 2 | Empty returns 200, with data + limit param returns entries |
| `handleStartWork` | 1 | Nonexistent issue returns error |
| `handleMergeIssue` | 2 | Invalid strategy returns 400, malformed JSON returns 400 |
| `handleCommitDetail` | 1 | Nonexistent SHA returns 404 |

## Remaining untested handlers

The git-dependent handlers (`handleGetIssueCommits`, `handleGetCommitDiff`, `handleGetIssueFiles`, `handleGetIssueDiff`, `handleMergeability`) require a full git repo with branches. Their core logic is tested at the `exponential` package level (merge tests, git tests). The HTTP layer for these handlers is thin routing + JSON serialization.

## Key decisions

- **`TestHandleGetArtifact_Found`** writes a file directly to `.xpo/artifacts/<id>/` to avoid going through the full artifact creation flow — tests the handler's read path in isolation.
- **Git-dependent validation** — tested error paths (bad strategy, bad JSON, nonexistent issue) that don't require a git repo, rather than standing up a full git repo for the HTTP layer when the underlying operations are already tested.
