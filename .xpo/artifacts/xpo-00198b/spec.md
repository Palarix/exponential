# Spec: Add `mode` parameter to `xpo start` CLI and MCP tool

## What

Add a `mode` parameter (`worktree` | `branch`) to both the CLI and MCP `start` interfaces, allowing per-invocation override of the global `worktrees` config setting.

## Why

The global `worktrees` config in `.xpo/config.yml` is all-or-nothing. There is no way to create a plain branch when worktrees are enabled, or force a worktree when they are disabled. The CLI has `--no-wt` to negate the config, but no positive override — and the MCP tool has no override at all.

## How

### CLI (`cmd/exponential/update.go`)

- Remove the `startNoWorktree` bool var and `--no-wt` flag.
- Add a `startMode` string var and `--mode` flag accepting `worktree` or `branch`.
- In `startCmd.Run`, validate the flag value and override `cfg.Worktrees` before calling `StartWork`:
  - `"worktree"` → `cfg.Worktrees = true`
  - `"branch"` → `cfg.Worktrees = false`
  - empty → no override (use global config)
  - anything else → error

### MCP (`internal/mcpserver/tools.go`)

- Add `Mode string` field to `startIn` struct with jsonschema description.
- In the `start` handler, validate and apply the override before calling `StartWork`:
  - `"worktree"` → set `cfg.Worktrees = true` on the client's config
  - `"branch"` → set `cfg.Worktrees = false`
  - empty → no override
  - anything else → return error

### Core

No changes to `StartWork` — it already branches on `c.Config.Worktrees`.

## Acceptance Criteria

- [ ] `xpo start --mode worktree <id>` creates a worktree regardless of global config
- [ ] `xpo start --mode branch <id>` creates a branch regardless of global config
- [ ] `xpo start <id>` (no flag) uses global config as before
- [ ] `xpo start --mode invalid <id>` returns an error
- [ ] MCP `start` with `mode: "worktree"` overrides config to use worktrees
- [ ] MCP `start` with `mode: "branch"` overrides config to use branches
- [ ] MCP `start` with no `mode` uses global config as before
- [ ] MCP `start` with invalid `mode` returns an error
- [ ] `make test` passes
