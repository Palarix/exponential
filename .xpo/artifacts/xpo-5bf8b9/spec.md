# Spec: Server write-path handler tests

## What

Tests for the 12 untested HTTP handlers in `internal/server/handlers.go`.

## Approach

Split into two categories:
1. **Non-git handlers** — can use `setupTestServer` (no git repo needed)
2. **Git-dependent handlers** — test input validation / error paths that don't require an actual git repo; full integration tested at the `exponential` layer already

## Tests

### Non-git handlers
1. **TestHandleGetArtifact_NotFound** — no artifact exists, returns 404
2. **TestHandleGetArtifact_MissingParams** — missing ID/filename, returns 400
3. **TestHandleListInstances** — returns JSON array (may be empty)
4. **TestHandleGetCycles_Enabled** — cycles enabled with issues, returns cycle list
5. **TestHandleCycleProgress_Disabled** — cycles not enabled, returns 400
6. **TestHandleCycleProgress_MissingID** — no cycle ID, returns 400
7. **TestHandleTimeline_Empty** — no events, returns empty timeline
8. **TestHandleTimeline_WithLimit** — limit param respected

### Git-dependent (validation only)
9. **TestHandleStartWork_MissingID** — no ID, returns 400
10. **TestHandleMergeIssue_MissingID** — no ID, returns 400
11. **TestHandleMergeIssue_InvalidStrategy** — bad strategy, returns 400
12. **TestHandleMergeIssue_BadJSON** — malformed body, returns 400
13. **TestHandleCommitDetail_MissingSHA** — no SHA, returns 400

## Acceptance Criteria

- [ ] All 13 tests pass
- [ ] `make test` passes
- [ ] No production code changes
