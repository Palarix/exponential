# Walkthrough: Add length limits to string input fields

## What changed

Two files: `internal/exponential/client.go` and `internal/exponential/artifact.go`.

## Constants

Exported constants defined in `client.go`:

```go
const (
    MaxTitleLen       = 500
    MaxDescriptionLen = 100 * 1024
    MaxAssigneeLen    = 200
    MaxLabelLen       = 100
    MaxCommentLen     = 100 * 1024
)
```

And in `artifact.go`:

```go
const MaxArtifactContentLen = 1024 * 1024
```

## ValidateCreatePayload

Length checks added at the top of the method, before status/parent/dependency validation:

- `len(p.Title) > MaxTitleLen`
- `len(p.Description) > MaxDescriptionLen`
- `len(p.Assignee) > MaxAssigneeLen`
- Each label in `p.Labels` checked against `MaxLabelLen`

## ValidateUpdatePayload

Same checks but for pointer fields (`*string`), guarded by nil checks:

- `p.Title != nil && len(*p.Title) > MaxTitleLen`
- `p.Description != nil && len(*p.Description) > MaxDescriptionLen`
- `p.Assignee != nil && len(*p.Assignee) > MaxAssigneeLen`
- Labels checked directly (slice, not pointer)

## AddComment

The `AddComment` method on `Client` now checks `len(text) > MaxCommentLen` before delegating to the transport. This covers both the HTTP `handleDraft` COMMENT branch and the MCP `comment` tool — both call `client.AddComment`.

## writeArtifact

The `writeArtifact` method on `LocalTransport` checks `len(content) > MaxArtifactContentLen` after filename validation, before writing to disk. Since `WriteSpec`, `WriteWalkthrough`, and `AddArtifact` all delegate to `writeArtifact`, the limit applies to all artifact types.
