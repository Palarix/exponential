# Walkthrough: unlink

## What was built and why

`unlink` is the counterpart of `link`. It removes one relationship between two issues without rewriting the rest of the list through `update { links }`. It's available as an MCP tool (`unlink { source, target, type }`) and a CLI command (`xpo unlink <source> <target> -t <type>`, plus `--json`).

## How the pieces fit together

### `Client.Unlink` (`internal/exponential/unlink.go`)

The MCP tool and the CLI both call this one function, so the logic exists only once (unlike `link`, which is written out in both places).

1. Validate the arguments and normalize the kind with `NormalizeDependencyKind`. `depends-on`, `DependsOn` and similar spellings all work.
2. Resolve the source with `GetIssue`.
3. Resolve the target with `resolveLinkTarget` (below).
4. Find the entry in `src.Links` with that target and kind. Error `no link <kind> <target> on <source>` if there isn't one.
5. Write `src.Links` minus that entry back with `UpdateIssue(..., "unlink")`.

Step 5 is the key point. Since xpo-9d6609, `update`'s dependency list is the issue's **full relationship set**, and `reconcileLinks` removes a link that's left out from whichever issue stores it. `unlink` therefore needs no logic of its own to tell owned links from derived ones:
- `unlink { A, B, blocks }`, with the row stored on A: A's owned list shrinks.
- `unlink { B, A, blocked_by }`, with the row stored on A: B's payload drops the derived entry, so reconciliation writes an UPDATE on A that removes its row.
- Stored on both sides: both rows go.

The remaining list is never nil (`make(..., 0, ...)`). Removing the last link therefore sends `[]`, which clears the list, and not "field unset".

### `resolveLinkTarget`

It tries `GetIssue` first. If no live issue matches (the target was deleted or purged), it falls back to the targets in the source's `Links`: an exact ID, or a unique substring match, the same rule `resolveIssue` uses. Without this, a link to a vanished issue could never be removed. If nothing matches, the original `GetIssue` error is returned, prefixed with `target:`.

### Rows with an empty `source_id`

Nothing special is needed. Projection already repairs these on read (xpo-080fc4), so they appear in `Links` like any other row.

### MCP (`internal/mcpserver/tools.go`)

`unlink` is registered right after `link`. It reuses `LinkToolInput` and `LinkOutput`, since the fields are identical and the two tools stay symmetric. It broadcasts UPDATE for both the source and the target, because either one's stored rows may have changed.

### CLI (`cmd/exponential/unlink.go`)

It mirrors `link`: positional `<source> <target> -t <type>`, or `--json` reading `{source, target, type}` from stdin and writing a `LinkOutput`. Errors in `--json` mode go through `exitJSONError`.

### Docs

`unlink` is added to the tool table in the managed skill template (`agents.go`) and to the README. The README's tool list was also brought up to date: it now says fourteen tools and includes `rationale`, which it was missing.

## Key decisions

- **Built on reconciliation, not on row surgery.** `unlink` edits the combined view and lets `update` decide which stored rows go. This keeps one code path for "which side stores it" and guarantees `unlink` and the web sidebar behave the same way.
- **Reuse `LinkToolInput`** rather than adding an identical `UnlinkToolInput`. This is a departure from the original spec.

## Acceptance criteria

- [x] `unlink` removes a relationship stored on the source. Evidence: `TestUnlink_RemovesOwnedLinkKeepsOthers`, `TestUnlinkCLI_JSON`.
- [x] `unlink` removes a relationship stored only on the target (as its inverse). Evidence: `TestUnlink_RemovesLinkStoredOnTarget`, MCP `TestUnlink`, `TestUnlinkCLI_Positional`.
- [x] Removing the last link leaves the issue with no links. Evidence: `TestUnlink_RemovesLastLink`.
- [x] Rows written with an empty `source_id` (before xpo-080fc4) can be removed. Evidence: `TestUnlink_RemovesRowWithEmptySourceID`.
- [x] A link to a target that no longer resolves can still be removed by its ID. Evidence: `TestUnlink_TargetNoLongerResolves` (removed by suffix after the target was deleted).
- [x] Error when no matching relationship exists. Evidence: `TestUnlink_Errors` (missing args, bad kind, no such link, unknown target, self), MCP `TestUnlink` second call, `TestUnlinkCLI_JSON` second call.
- [x] CLI works with both positional arguments and `--json`. Evidence: `TestUnlinkCLI_Positional`, `TestUnlinkCLI_JSON` (end to end against the built binary).
- [x] Tests cover all of the above; `make test` passes.
