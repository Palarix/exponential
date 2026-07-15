# Add length limits to string input fields

## Problem

No length limits on any string field. An agent or malformed request could store arbitrarily large strings.

## Limits

| Field | Limit |
|---|---|
| Title | 500 chars |
| Description | 100 KB |
| Comment body | 100 KB |
| Assignee | 200 chars |
| Label name | 100 chars |
| Artifact/spec/walkthrough content | 1 MB |

## Implementation

### `internal/exponential/client.go`

Add checks in `ValidateCreatePayload` and `ValidateUpdatePayload` for title, description, assignee, and label names.

### `internal/exponential/artifact.go`

Add a content size check in `writeArtifact` (covers specs, walkthroughs, and generic artifacts).

### `internal/server/handlers.go`

Add comment body length check in the COMMENT branch of `handleDraft`.

### MCP tools

The `comment` tool handler in `tools.go` should check body length. The `spec`, `walkthrough`, and `artifact` tools delegate to `writeArtifact` which will have the check.

## Acceptance Criteria

- [ ] Title > 500 chars rejected.
- [ ] Description > 100KB rejected.
- [ ] Comment > 100KB rejected.
- [ ] Assignee > 200 chars rejected.
- [ ] Label > 100 chars rejected.
- [ ] Artifact content > 1MB rejected.
- [ ] Tests pass.
