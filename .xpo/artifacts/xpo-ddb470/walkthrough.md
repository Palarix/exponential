# Walkthrough: handleDraft capability mismatch

## What changed

Three files: `server.go` (new helper), `handlers.go` (per-branch checks), `router.go` (route registration).

## Problem

The `handleDraft` HTTP endpoint was registered with a single `issue.create` capability but acted as a multiplexer for four distinct operations: CREATE, UPDATE, COMMENT, and DELETE. A user whose role only granted `issue.create` could also update, comment on, and delete issues through this endpoint — the capability check happened once at the route level before the handler determined which operation was being performed.

## Solution

### `requireCapability` helper (`internal/server/server.go`)

Added a method on `Server` that checks whether the current request's user has a specific capability:

```go
func (s *Server) requireCapability(w http.ResponseWriter, r *http.Request, capability string) bool
```

It returns `true` if the check passes (including when auth is disabled). On failure it writes a 403 response and returns `false`, so the caller can simply `return` after a false result.

This reuses the same `auth.UserFromContext` and `perms.HasCapability` logic as the `RequireCapability` middleware, but can be called inline within a handler branch.

### Per-branch checks (`internal/server/handlers.go`)

Each branch of the `handleDraft` switch now calls `requireCapability` with the matching capability before processing the payload:

| Branch | Capability |
|---|---|
| CREATE | `issue.create` |
| UPDATE | `issue.update` |
| COMMENT | `issue.comment` |
| DELETE | `issue.delete` |

### Route registration (`internal/server/router.go`)

The route was changed from `issue.create` to `issue.read`:

```go
handleCap("POST /api/draft", "issue.read", s.handleDraft)
```

This serves as a baseline authentication gate — any authenticated user can reach the endpoint, but each branch enforces the specific capability. A read-only user (viewer role with only `issue.read`) can hit the endpoint but will get 403 from every write branch.
