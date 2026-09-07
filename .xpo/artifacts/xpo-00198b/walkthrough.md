# Walkthrough: Add `mode` parameter to `xpo start`

## What changed

The `xpo start` command previously had no way to override the global `worktrees` config on a per-invocation basis. The CLI offered only a `--no-wt` flag (negate worktrees), and the MCP tool had no override at all. This change replaces `--no-wt` with a bidirectional `--mode` flag and adds the same parameter to the MCP tool.

## Files modified

### `cmd/exponential/update.go`

- Replaced `startNoWorktree bool` variable and `--no-wt` boolean flag with `startMode string` and `--mode` string flag.
- The `startCmd.Run` handler uses a switch on `startMode`: `"worktree"` sets `cfg.Worktrees = true`, `"branch"` sets `cfg.Worktrees = false`, empty string is a no-op (global config applies), anything else exits with an error.

### `internal/mcpserver/tools.go`

- Added `Mode string` field to the `startIn` struct with a jsonschema description for the tool catalog.
- In the `start` handler, after `clientFor` returns a client, a switch on `in.Mode` validates the value. When set, a **shallow copy** of the client's config is created with `Worktrees` overridden, and the copy is assigned back to `c.Config`.

## Key decision: config copy in MCP handler

The MCP server's `toolset` holds a single shared `*config.Config`. The CLI can safely mutate `cfg.Worktrees` because each CLI invocation is its own process. The MCP server handles concurrent requests, so mutating the shared config would be a race condition. The shallow copy (`cfgCopy := *c.Config`) gives the handler a private config for this one call without affecting other concurrent requests.
