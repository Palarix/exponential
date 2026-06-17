# Architecture & Design

This document describes the architecture of xpo as implemented. It is the authoritative reference for how the system works — the old `design_docs/` directory contains early sketches that have diverged from the code.

## Overview

Exponential is an event-sourced issue tracker that stores its data in a git-tracked append-only log (`.xpo/issues.db`). A single Go binary ships the CLI, a web UI, an HTTP API, and an MCP server. In local mode everything reads and writes the event log directly; in distributed mode a headless server exposes the same data over HTTP with SSH-key authentication.

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│   CLI commands   │  │  Web UI (React)  │  │  MCP server      │
│  add, update,    │  │  xpo board     │  │  xpo mcp       │
│  start, merge …  │  │                  │  │  (stdio / HTTP)  │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │                     │                      │
         └─────────────┬───────┘──────────────────────┘
                  ┌────▼────┐
                  │  Client │  ← chooses transport based on config
                  └────┬────┘
            ┌──────────┴──────────┐
            ▼                     ▼
   ┌────────────────┐   ┌─────────────────┐
   │ LocalTransport │   │ RemoteTransport  │
   │  (issues.db)   │   │  (REST /api/*)   │
   └───────┬────────┘   └────────┬─────────┘
           │                     │
           │            ┌────────▼─────────┐
           │            │   HTTP Server     │
           │            │  (xpo serve)    │
           │            │  + Auth (JWT/SSH) │
           │            │  + SSE hub        │
           │            │  + projection     │
           │            │    cache          │
           │            └────────┬─────────┘
           │                     │
           └──────────┬──────────┘
                 ┌────▼──────────┐
                 │ Storage layer │
                 │ (issues.db)   │
                 └───────────────┘
```

## Event sourcing

All mutations are recorded as events — there is no mutable state on disk. An `Event` has an issue ID, a type, a typed payload, a timestamp, and an author:

```
{type: "CREATE",  payload: {title, description, status, labels, ...}}
{type: "UPDATE",  payload: {status: "DOING", assignee: "Alice"}}
{type: "COMMENT", payload: {id: "cmt-...", text: "markdown body"}}
{type: "DELETE",  payload: {reason: "duplicate", cascade: false}}
{type: "MERGE",   payload: {branch, base_sha, merge_sha, strategy}}
```

Events are stored as one JSON object per line (JSONL) in `.xpo/issues.db`, appended with `O_APPEND`. This makes concurrent appends safe on POSIX filesystems and keeps git merge conflicts limited to the tail of the file.

### Projection

The current state of all issues is derived by replaying the event log. `ProjectIssues(events)` walks the log in order and builds a `map[string]*Issue`. Each event type has a deterministic effect:

- **CREATE** initializes an issue with all fields from the payload.
- **UPDATE** patches only the non-nil fields in the payload.
- **DELETE** marks the issue as deleted (soft delete; events are retained).
- **COMMENT** appends to the issue's comment list.
- **MERGE** records branch merge metadata.

`ProjectIssuesWithConfig()` extends this with config-driven post-processing: parent status inference from children, cycle rollover for issues in past cycles, and estimate aggregation for epics.

**Files:** `internal/xpo/projection.go`, `internal/model/types.go`

### Collapsed writes

The web UI batches mutations as in-memory "pending events" before persisting. When the user saves, `AppendEventCollapsed()` optimizes the write by merging uncommitted edits on the same issue — two consecutive UPDATEs to the same issue become one. It also prunes no-op updates by comparing against the committed state (the portion of `issues.db` at `git HEAD`).

**Files:** `internal/storage/collapse.go`

### Archiving

Old DONE and deleted issues accumulate events that slow down projection. `xpo archive` moves their events from `issues.db` to `archive.db`. A backup of `issues.db` is created before the rewrite. Regular commands search the archive as a fallback when an issue isn't found in the active log.

**Files:** `internal/storage/archive.go`, `internal/xpo/maintenance.go`

## Data model

### Issue

The `Issue` struct is the projected view of an issue's event history:

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Globally unique, prefixed (e.g. `xpo-a1b2c3`) |
| Title, Description | string | Markdown content |
| Status | enum | `BACKLOG`, `PLANNED`, `DOING`, `BLOCKED`, `DONE` |
| ParentID | string | Parent issue ID (for epics/sub-issues) |
| Estimate | int | Story points |
| Priority | int | 1 (urgent) through 4 (low), 0 = unset |
| SortOrder | string | Fractional indexing key for drag-and-drop ordering |
| Assignee | string | `"Name <email>"` format |
| CycleID | string | Assigned sprint cycle (`YYYY-MM-DD` start date) |
| Labels | []string | Freeform labels (`bug`, `feature`, `epic`, etc.) |
| Dependencies | []Dependency | Relationship links to other issues |
| Comments | []Comment | Threaded comments |
| BranchStats | *BranchStats | Git branch metadata (commits, files changed, etc.) |
| InferredStatus | bool | True when status was derived from children, not set explicitly |
| Events | []Event | Full event history (attached during projection) |

### Dependencies

Dependencies express directional relationships between issues:

| Kind | Inverse | Meaning |
|------|---------|---------|
| `blocks` | `blocked_by` | Source blocks target from starting |
| `depends_on` | `dependency_of` | Source depends on target being done |
| `duplicates` | `duplicated_by` | Source duplicates target |
| `relates_to` | `relates_to` | Symmetric, informational |

The system enforces `blocked_by` at status transition time — an issue cannot move to DOING if any of its `blocked_by` targets are not DONE.

### Sort order

Issues use fractional indexing for drag-and-drop ordering. Sort keys are strings that maintain lexicographic order, generated by `sortorder.GenerateNKeysBetween()`. This allows inserting between any two adjacent issues without rewriting other keys.

Data model v3 introduced globally consistent sort order (previously per-status-group). The v2→v3 migration walks all issues in visual order and assigns fresh keys.

**Files:** `internal/sortorder/`, `internal/xpo/migrate.go`

## Transport abstraction

All issue operations go through the `Transport` interface:

```go
type Transport interface {
    ListIssues(opts FilterOptions) ([]*Issue, error)
    GetIssue(id string) (*Issue, error)
    FindIssue(id string) (*Issue, []*Issue, bool, error)
    AddIssue(payload CreatePayload) (*Issue, error)
    UpdateIssue(id string, payload UpdatePayload, action string) ([]string, error)
    AddComment(issueID, text string) error
    DeleteIssue(id string, reason string, cascade bool) error
    CheckDuplicates(title string) ([]*Issue, error)
    GetUser() string
    GetInbox(since time.Time) ([]InboxItem, error)
}
```

**LocalTransport** reads/writes `.xpo/issues.db` directly. It resolves issue IDs flexibly — exact match, prefix match (with configured prefix), or substring match. It supports collapsed writes for draft optimization.

**RemoteTransport** proxies to a xpo server via REST. Mutations go through `POST /api/draft`, reads through `GET /api/issues`. Authentication is via Bearer JWT token.

**Client** wraps a Transport and adds git-specific operations (`StartWork`, `MergeIssue`, `Review`) that only make sense locally. `NewClient(cfg)` chooses the transport based on whether `cfg.Remote.URL` is set.

**Files:** `internal/xpo/transport.go`, `internal/xpo/local_transport.go`, `internal/xpo/remote_transport.go`, `internal/xpo/client.go`

## Storage layer

The storage package handles all file I/O for the event log:

| Function | Description |
|----------|-------------|
| `AppendEvent(evt)` | Opens `issues.db` with `O_APPEND`, writes one JSON line |
| `AppendEventCollapsed(evt)` | Merges with uncommitted tail before appending |
| `ReadEvents()` | Scans `issues.db` line-by-line, returns `[]Event` |
| `ReadArchivedEvents()` | Same for `archive.db` |
| `ArchiveEvents(active, archived)` | Backs up, rewrites `issues.db`, appends to `archive.db` |

**Files on disk:**

| Path | Tracked | Purpose |
|------|---------|---------|
| `.xpo/issues.db` | Yes | Active event log (JSONL) |
| `.xpo/archive.db` | Yes | Archived events |
| `.xpo/config.yaml` | Yes | Project configuration |
| `.xpo/authorized_keys` | Yes | SSH public keys for server auth |
| `.xpo/server.key` | No | Ed25519 signing key (auto-generated) |
| `.xpo/issues.snapshot.json` | No (.gitignored) | Projection cache |

**Files:** `internal/storage/`

## HTTP server

The server (`xpo serve` or `xpo board`) exposes a REST API and optionally serves the embedded React frontend.

### Projection cache

The server maintains a read-through cache of projected issues. `GetProjectedIssues()` returns cached results on cache hit (checked via file mtime/size and pending-event generation counter). On miss, it re-reads `issues.db` from disk and reprojects. Pending events (unsaved web UI drafts) are applied at query time without touching disk.

### Pending events

The web UI batches mutations in memory as "pending events" (`Server.pendingEvents`). These are applied on top of the persisted projection at read time, giving instant feedback in the UI. They're only written to disk when the user clicks Save (which calls `POST /api/save`).

### SSE

Real-time updates are pushed to connected web clients via Server-Sent Events (`GET /api/events`). Handlers call `broadcastEvent(type, issueID)` after mutations. The SSE hub manages client channels with non-blocking sends (slow clients get dropped). A 30-second keepalive prevents connection timeouts.

### Proxy mode

`xpo board --remote <url>` runs the web UI locally but reverse-proxies all `/api/*` and `/auth/*` requests to a remote server. This gives the full board experience while data lives on the server. SSE is disabled in proxy mode.

**Files:** `internal/server/`

## Authentication

Authentication is only active in server mode (`xpo serve`). Local mode (`xpo board`) binds to `127.0.0.1` with no auth.

### Flow

1. Client calls `POST /auth/challenge` → server returns a random nonce (256-bit, single-use, TTL-bounded).
2. Client signs the nonce with an SSH private key and sends `POST /auth/verify` with `{public_key, signature, nonce}`.
3. Server validates the nonce, verifies the signature, looks up the public key in `.xpo/authorized_keys`, and returns a JWT (EdDSA-signed, 7-day expiry).
4. Client caches the JWT and sends it as `Authorization: Bearer <token>` on subsequent requests.

### Permissions

When `permissions` is configured in `config.yaml`, the server enforces capability-based access control:

```yaml
permissions:
  roles:
    admin: ["*"]
    member: ["issue.read", "issue.create", "issue.update", "issue.comment", "issue.start"]
  users:
    alice@example.com: admin
    bob@example.com: member
```

Capabilities: `issue.read`, `issue.create`, `issue.update`, `issue.comment`, `issue.start`, `issue.merge`, `issue.delete`, `issue.link`.

When permissions are not configured, all authenticated users have full access.

**Files:** `internal/auth/`

## MCP server

The MCP server (`xpo mcp`) speaks the Model Context Protocol over stdio. It registers tools that map 1:1 to Client methods:

| Tool | Client method |
|------|--------------|
| `list` | `ListIssues()` |
| `show` | `GetIssue()` |
| `history` | (event log for one issue) |
| `add` | `AddIssue()` |
| `update` | `UpdateIssue()` |
| `comment` | `AddComment()` |
| `link` | (dependency creation) |
| `start` | `StartWork()` |
| `merge` | `MergeIssue()` |

### Identity resolution

MCP tool calls need an author identity for event attribution. Resolution order:

1. `$XPO_AGENT_IDENTITY` environment variable (operator-controlled).
2. MCP `clientInfo` from the initialize handshake (e.g. `claude-code/2.x <agent@mcp>`).
3. The configured xpo user (git identity fallback).

Each tool invocation creates a fresh Client with `UserOverride` set to the resolved identity.

**Files:** `internal/mcpserver/`

## Configuration

### Project config (`.xpo/config.yaml`)

Committed to git. Contains project-level settings: prefix, labels, cycles, automations, contributors, permissions.

### User config (`~/.config/xpo/user.yaml`)

Per-user, not committed. Stores server credentials (JWT tokens), remote URLs, and inbox read cursors.

### Key settings

| Setting | Default | Description |
|---------|---------|-------------|
| `prefix` | `<folder>-` | Issue ID prefix |
| `auto_commit` | `false` | Git-commit after each CLI write |
| `estimation_system` | `fibonacci` | `fibonacci`, `linear`, or `tshirt` |
| `count_unestimated` | `true` | Count unestimated issues as 1 point |
| `cycles.enabled` | `false` | Enable sprint cycles |
| `cycles.duration` | — | `1w`, `2w`, `3w`, or `4w` |
| `automations.*` | `false` | Parent/child status cascading rules |

**Files:** `internal/config/`

 
### Core settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `prefix` | string | `xpo-` | Issue ID prefix. Issue IDs become `<prefix>1`, `<prefix>2`, etc. Change this to scope IDs per project (e.g. `api-` produces `api-1`). |
| `version` | integer | — | Configuration format version. Managed automatically by `xpo migrate`. |
| `name` | string | — | Project display name shown in the web UI sidebar. |
| `user` | string | — | Default identity for local operations, in `"Name <email>"` format. |
| `editor` | string | `$EDITOR` or `vim` | Text editor command used when xpo needs interactive input. |
| `auto_commit` | boolean | `false` | When `true`, every xpo write (add, update, comment, etc.) is automatically committed to git. When `false`, changes accumulate in the working tree and you commit on your own schedule. |

### Estimation

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `estimation_system` | string | `fibonacci` | The scale used for story-point estimates (see table below). |
| `count_unestimated` | boolean | `true` | Whether unestimated child issues count toward a parent's story-point rollup. When `false`, only estimated children contribute to the total. |

**Estimation systems:**

| System | Point values | Display |
|--------|-------------|---------|
| `fibonacci` | 1, 2, 3, 5, 8 | Numeric |
| `exponential` | 1, 2, 4, 8, 16 | Numeric |
| `linear` | 1, 2, 3, 4, 5 | Numeric |
| `shirt` | 1, 2, 3, 5, 8 | XS, S, M, L, XL |

The `shirt` system stores numeric values internally but renders them as t-shirt sizes in the UI and CLI. Both forms are accepted as input (e.g. `xpo estimate <id> M` or `xpo estimate <id> 3`).

### Cycles (sprints)

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `cycles.enabled` | boolean | `false` | Enable sprint/iteration tracking. |
| `cycles.duration` | string | — | Cycle length: `1w`, `2w`, `3w`, or `4w`. Required when cycles are enabled. |
| `cycles.start_day` | string | — | Day of the week each cycle begins (e.g. `monday`, `wednesday`). Required when cycles are enabled. |
| `cycles.anchor_date` | string | — | A `YYYY-MM-DD` reference date to pin cycle numbering. If omitted, xpo anchors to the nearest `start_day` before now. Useful for aligning cycles with an existing sprint calendar. |

Cycles are numbered sequentially from the anchor date. Issues not marked DONE by the end of a cycle automatically roll forward to the current cycle; completed issues retain their original cycle ID.

### Labels

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default_labels` | list of strings | `[bug, feature, epic, improvement]` | Labels shown in the UI label picker. If omitted, the built-in set is used. |
| `hide_default_labels` | boolean | `false` | Hide the built-in labels and only show custom ones. |
| `labels` | map of name → hex color | built-in colors | Custom label colors as CSS hex values (e.g. `"#e11d48"`). Entries here override the built-in color for that label name. Label keys are case-sensitive. |

### Contributors

| Key | Type | Description |
|-----|------|-------------|
| `contributors` | list of strings | Team members available for assignment in the UI. Each entry must be in `"Name <email>"` format. |

### Automations

All automations default to `true` and control automatic state propagation between parent and child issues.

| Key | Default | Description |
|-----|---------|-------------|
| `automations.auto_complete_parent` | `true` | Automatically move a parent to DONE when all its children are DONE. |
| `automations.auto_close_sub_issues` | `true` | Automatically move children to DONE when their parent is moved to DONE. |
| `automations.auto_progress_sub_issues` | `true` | Automatically advance children when their parent moves forward (e.g. PLANNED → DOING). |
| `automations.auto_progress_parent` | `true` | Automatically advance a parent when one of its children moves forward. |

### Permissions

Permissions are only enforced in [distributed mode](#distributed-mode). When no permissions block is configured, all operations are allowed.

| Key | Type | Description |
|-----|------|-------------|
| `permissions.roles` | map of role → list of capabilities | Define named roles. Use `["*"]` to grant all capabilities. |
| `permissions.users` | map of email → role | Assign users to roles by email address. |

**Available capabilities:** `issue.read`, `issue.create`, `issue.update`, `issue.comment`, `issue.start`, `issue.merge`, `issue.delete`, `issue.link`.

### Remote

| Key | Type | Description |
|-----|------|-------------|
| `remote.url` | string | URL of a remote xpo server for distributed mode. |
| `remote.token` | string | Authentication token. If omitted, xpo resolves it from `~/.config/xpo/user.yaml` (populated by `xpo login`). |

### Style

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `style.theme` | string | `default` | UI theme name for the web board. |

## Branch workflows

`xpo start <id>` transitions an issue to DOING and creates a git branch (`<prefix><short-id>/<slug>`). The operation is locked with a file-based git lock to prevent concurrent branch creation races.

`xpo merge <id>` squash-merges the issue branch back to the default branch, appends a MERGE event, and transitions the issue to DONE. The working tree must be clean. Supported strategies: squash (default), merge commit, fast-forward.

`xpo review <id>` shows the unified diff between the issue branch and the default branch.

**Files:** `internal/xpo/start.go`, `internal/xpo/merge.go`, `internal/xpo/gitlock.go`

## Data model versioning

The data model version is stored in `config.yaml` as `version`. The current version is **3**.

| Version | Change |
|---------|--------|
| v1 | Initial schema (typed issues: TASK, BUG, EPIC) |
| v2 | Unified Issue type with labels, assignee, estimation, automations |
| v3 | Globally consistent sort_order keys across status groups |

`xpo migrate` runs pending migrations automatically. Migrations append corrective events to `issues.db` — they don't modify existing events.

**Files:** `internal/xpo/migrate.go`, `internal/version/version.go`
