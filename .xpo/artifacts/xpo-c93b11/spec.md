## What

Two issues can have **at most one link** between them, in either direction, whichever issue stores it. `link` refuses to add a second link of a different kind unless the caller passes `overwrite`. In that case the existing link is replaced. `update`/`add` reject a `links` set with two entries for the same target.

## Why

Today the only duplicate check is on the exact (source, target, kind) triple, so A blocks B + A relates_to B, and even contradictory A blocks B + A blocked_by B, are all accepted. Only the web sidebar picker prevents this, and only in the UI. The board already has 5 multi-link pairs, which caused the double edges fixed on the read side in xpo-2fc583.

Also absorbs xpo-f94753: three copies of the link validation (MCP, JSON CLI, plain-text CLI) have drifted apart and word their errors differently.

## Prior rationale

- xpo-9d6609: links are bidirectional. `issue.Links` is the combined view (stored and derived), and `update`'s `links` is the issue's **full** relationship set; `reconcileLinks` removes a left-out link from whichever side stores it. So "one link per pair" can be checked entirely from one issue's perspective: every link to B, whichever side stores it, appears in A's `Links` with target B.
- xpo-9d5d00: `Client.Unlink` is the shared helper pattern. It edits `src.Links` and writes it back via `UpdateIssue`, letting reconcile handle the other side.

## How

### Core rule: `normalizeUpdateDependencies` (`internal/exponential/update.go`)

- It now also takes the issue's current `Links`. For each target, it counts the payload's entries. If a target has more than one entry, the payload is rejected, **unless** that target's set of kinds is identical to the current `Links` for that target. That is the grandfathering rule: an untouched legacy pair doesn't block unrelated edits such as a link to some other issue.
- One error message for this, replacing `duplicate link …`:
  `only one link is allowed between two issues: xpo-A would have <k1>, <k2> to xpo-B`
- Self-link check unchanged: `cannot link an issue to itself`.

### Create path: `AddIssue` (`internal/exponential/add.go`)

- Run the same per-target check on `payload.Dependencies`, with no current links since the issue is new. This covers MCP `add`, CLI `add --json` and the web draft CREATE, which all call `AddIssue`. It closes the gap where create did no duplicate check at all.

### Shared `Client.Link` (new `internal/exponential/link.go`, mirrors `Unlink`)

`Link(source, target, kind string, overwrite bool) (added model.Dependency, replaced []model.Dependency, msgs []string, err error)`:

1. Validate required fields and the kind, resolve both issues, reject a self-link (same wording as today).
2. Collect `existing` = entries in `src.Links` whose target is `tgt`. This includes derived ones, so a link stored on B counts.
3. Then:
   - **Identical link already present and it's the only one:** error `link <kind> <tgt> already exists on <src>` (current MCP wording, now used by all entry points). Same with overwrite, since there is nothing to replace.
   - **Other link(s) present, no overwrite:** error `xpo-A is already linked to xpo-B (blocks); set overwrite to replace it with relates_to`. Several legacy links are listed comma-separated.
   - **Otherwise:** write `src.Links` minus all `existing`, plus the new link, via `UpdateIssue(..., "link")`. Reconcile removes the old row from whichever issue stored it. `replaced` = `existing` minus an identical entry, if any. So overwrite on a legacy pair that already includes the requested link collapses it to just that link.

### Entry points

- **MCP `link`:** gets a new `LinkToolInput.Overwrite bool` (`overwrite`). `unlink` gets its own `UnlinkToolInput` (source/target/type), so its schema doesn't advertise `overwrite`. The handler calls `c.Link`. `LinkOutput` gains `replaced []LinkOutput` (omitempty). The text result is e.g. `Linked xpo-A relates_to xpo-B (replaced blocks)`. The tool description explains the one-link rule and `overwrite`.
- **CLI `xpo link`:** gets a `--overwrite` flag. The JSON payload accepts `overwrite`. The plain-text and `--json` paths both call `client.Link`, so all three entry points share wording (xpo-f94753).
- **`update` `links` schema description:** add "at most one link per target issue".
- **Web:** no UI change. The sidebar picker already hides linked issues, and draft errors surface as they do today. Changing a kind in the sidebar is xpo-903ca3.

### Housekeeping

- When merging, close xpo-f94753 with a comment that it was fixed by this issue.
- The 5 existing multi-link pairs are left as they are for the user to clean up by hand. `link … overwrite` is the tool for it.

## Acceptance criteria

- [ ] `link` with a different kind on an already-linked pair fails with a clear error and writes nothing
- [ ] `link ... overwrite: true` / `--overwrite` replaces the existing link, whichever side stored it
- [ ] `update`/`add` reject payloads with two links to the same target
- [ ] Unrelated updates to issues with legacy multi-link pairs still succeed
- [ ] CLI and MCP use the same wording for these errors (incl. xpo-f94753's "already exists" and self-link)
- [ ] Go tests cover each path (core rule, grandfathering, `Client.Link` incl. derived-side overwrite, `AddIssue`, MCP handler, plain-text CLI)
