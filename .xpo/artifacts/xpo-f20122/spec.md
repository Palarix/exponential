# Spec: Add test coverage for local_transport.go

## What

Unit tests for the core state-access functions in `local_transport.go`.

## Why

Zero direct tests. Every CLI command and MCP tool resolves issues through `resolveIssue`. Its prefix-matching, short-hash fallback, and ambiguity detection are only tested indirectly.

## Tests

### resolveIssue
1. **ExactMatch** — full ID matches directly
2. **PrefixedMatch** — bare hash resolves with config prefix prepended (e.g. "abc123" → "xpo-abc123")
3. **SubstringMatch** — partial ID substring matches a single issue
4. **Ambiguous** — partial ID matches multiple issues, returns ambiguity error
5. **NotFound** — no match at all, returns not-found error
6. **PrefixedAlreadyIncluded** — ID already includes prefix, no double-prefix

### GetIssue
7. **GetIssue_Found** — issue exists in events, returns correct state
8. **GetIssue_NotFound** — no matching issue, returns error

### FindIssue
9. **FindIssue_ActiveWithChildren** — found in active store, returns children
10. **FindIssue_NotFound** — not in active or archive

### appendEvent
11. **AppendEvent_Basic** — event is written and readable
12. **AppendEvent_WithSource** — Source field is stamped on the event
13. **AppendEvent_WithOnBehalfOf** — OnBehalfOf field is stamped

## Approach

- `resolveIssue` tests can use a pre-built `map[string]*model.Issue` directly — no filesystem needed
- `GetIssue`, `FindIssue`, `appendEvent` need a temp dir with `.xpo/` initialized and `storage.ResetHubRoot()`

## Acceptance Criteria

- [ ] All 13 tests pass
- [ ] `make test` passes
- [ ] No production code changes
