# Add unlink tool to remove a single dependency

> Blocked by xpo-9d6609 (links are bidirectional). This spec assumes that model.

## What

A new `unlink { source, target, type }` MCP tool and an `xpo unlink <source> <target> -t <type>` CLI command (with `--json` support, like `link`). It removes one relationship between two issues, whichever side stores it.

## Why

`update { links }` replaces the whole list. `unlink` is the targeted counterpart of `link`.

## How

- **Shared core:** add `Client.Unlink(sourceID, targetID, kind string)` in `internal/exponential`. It is called by both the MCP tool and the CLI.
  1. Normalize the kind with `model.NormalizeDependencyKind`. Reject unknown kinds.
  2. Resolve the source with `GetIssue`. Resolve the target with `GetIssue`. If the target doesn't resolve (deleted or purged), fall back to matching the raw string against the source's link targets.
  3. Find the relationship: either `source kind target` stored on the source, or `target InverseKind(kind) source` stored on the target. Remove every stored row for it (both, if both sides recorded it).
  4. If neither exists, return `no link <kind> <target> on <source>`.
- **MCP:** register `unlink` next to `link` and broadcast `UPDATE` for each issue that changed.
- **CLI:** `cmd/exponential/unlink.go`, mirroring `link` (positional arguments, `-t`, `--json`).

## Acceptance criteria

- [ ] `unlink` removes a relationship stored on the source.
- [ ] `unlink` removes a relationship stored only on the target (as its inverse).
- [ ] Removing the last link leaves the issue with no links.
- [ ] Rows written with an empty `source_id` (before xpo-080fc4) can be removed.
- [ ] A link to a target that no longer resolves can still be removed by its ID.
- [ ] Error when no matching relationship exists.
- [ ] CLI works with both positional arguments and `--json`.
- [ ] Tests cover all of the above; `make test` passes.
