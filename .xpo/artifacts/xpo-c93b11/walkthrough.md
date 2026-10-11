## What was built

Two issues can now have **at most one link** between them, in either direction, whichever issue stores it. `link` refuses a second kind unless the caller passes `overwrite`, which replaces the existing link. `add`/`update` reject a `links` set with two entries for the same target. The three drifted copies of the link validation (MCP, JSON CLI, plain-text CLI) are replaced by one `Client.Link` helper, which also fixes xpo-f94753.

## How the pieces fit together

### Why one issue's view is enough

Since xpo-9d6609, `issue.Links` is the combined view (stored rows plus derived inverses), and `update`'s `links` is the issue's **full** relationship set; `reconcileLinks` removes a left-out link from whichever side stores it. So every link between A and B, whichever issue stores it, shows up in A's set with target B. "One link per pair" therefore reduces to "one entry per target" in a single issue's set. No cross-issue scan is needed.

### Core rule: `internal/exponential/update.go`

- `normalizeUpdateDependencies(id, deps, current)` now also receives `targetIssue.Links`. After stamping `SourceID`, clearing `Derived` and rejecting self-links, it calls `checkOneLinkPerPair`.
- `checkOneLinkPerPair` groups kinds by target (`kindsByTarget`, sorted). A target with two or more kinds is rejected **unless** its sorted kinds equal the current set for that target. This is the grandfathering rule: the 5 pre-existing multi-link pairs don't block unrelated edits such as adding a link to some other issue. But any change to such a pair other than collapsing it is rejected.
- The error replaces the old `duplicate link …`: `only one link is allowed between two issues: A would have k1, k2 to B`. An exact duplicate in the payload is the same violation and gets the same message.
- Changing a kind through `update` needs no flag: send the set with the new kind in place of the old, and reconcile removes the old row from whichever side stored it (`TestUpdateIssue_ChangesKindViaFullSet`).

### Create path: `internal/exponential/add.go`

`AddIssue` runs `checkOneLinkPerPair(id, deps, nil)`, since a new issue has no current links. MCP `add`, CLI `add --json` and the web draft CREATE all go through `AddIssue`, so this closes the gap where create did no duplicate check at all.

### `Client.Link`: `internal/exponential/link.go`

Built like `Client.Unlink` (xpo-9d5d00). It validates the input, resolves both issues and rejects self-links, then splits `src.Links` into links to other targets (`kept`), an `identical` flag, and links to the target with another kind (`replaced`):

- identical and nothing else → `link <k> <tgt> already exists on <src>`, even with overwrite. This keeps the existing behaviour, because there is nothing to replace.
- `replaced` non-empty and no overwrite → `<src> is already linked to <tgt> (<kinds>); set overwrite to replace it with <k>`. The wording says "set overwrite", which works for both `overwrite: true` and `--overwrite`.
- otherwise → `UpdateIssue(src, kept + added)`. Reconcile does the cross-issue removal. On a legacy pair that already includes the requested kind, overwrite collapses it to that one link and reports the rest as replaced.

Because `Links` includes derived rows, `replaced` reports links as seen from the source. Overwriting `B relates_to A` over a stored `A blocks B` reports `blocked_by`.

### Entry points

- **jsonio:** `LinkToolInput.Overwrite`. A new `UnlinkToolInput` keeps `overwrite` out of the `unlink` schema. `LinkOutput.Replaced []LinkRef` uses a separate `LinkRef` type rather than `[]LinkOutput`, to avoid a recursive type in the inferred JSON schema. `NewLinkOutput` and `LinkSummary()` build the JSON and text results shared by MCP and the CLI.
- **MCP `link`:** now a thin wrapper around `Client.Link`. After an overwrite it also broadcasts UPDATE for the target, since its stored row may have been removed.
- **CLI `xpo link`:** both the positional and `--json` paths call `client.Link`; there's a new `--overwrite` flag.
- **Docs:** the `link` tool and CLI help, the `links` schema text on `add`/`update`, and the generated xpo skill (`agents.go`) describe the rule. The checked-in `.claude`/`.codex`/`.opencode` skill copies refresh on the next `xpo init`.

## Non-obvious points

- `TestAddIssueAutoFillsSourceID` used to create an issue that both blocks and is blocked_by the same target, which the rule now forbids. It now uses two targets. The test is only about `SourceID` stamping.
- The web UI needed no change: the sidebar picker already hides linked issues. Changing a kind in the sidebar is xpo-903ca3.
- The read-side dedupe from xpo-2fc583 stays. It still covers the legacy pairs until they are cleaned up, e.g. with `xpo link … --overwrite`.

## Acceptance criteria

- [x] `link` with a different kind on an already-linked pair fails with a clear error and writes nothing: `TestLink_RefusesDifferentKindWithoutOverwrite` (exact message, links unchanged), `TestLinkRefusesSecondKindUnlessOverwrite` (MCP), `TestLinkCLI_SameErrorsAcrossPositionalAndJSON` "conflict" (CLI).
- [x] `link ... overwrite: true` / `--overwrite` replaces the existing link, whichever side stored it: `TestLink_OverwriteReplacesOwnedLink`, `TestLink_OverwriteReplacesLinkStoredOnTarget` (A's stored row removed), `TestLink_OverwriteCollapsesLegacyPair`, MCP `TestLinkRefusesSecondKindUnlessOverwrite`, CLI `TestLinkCLI_OverwritePositional` / `TestLinkCLI_OverwriteJSON`.
- [x] `update`/`add` reject payloads with two links to the same target: `TestUpdateIssue_RejectsTwoKindsToSameTarget`, `TestUpdateIssue_RejectsDuplicateLinks`, `TestAddIssue_RejectsTwoLinksToSameTarget` (issue not created).
- [x] Unrelated updates to issues with legacy multi-link pairs still succeed: `TestUpdateIssue_KeepsUntouchedLegacyMultiLinkPair` (pair split across both sides, plus a new link to C). `TestUpdateIssue_RejectsAddingToLegacyMultiLinkPair` covers the limit.
- [x] CLI and MCP use the same wording for these errors: `TestLinkCLI_SameErrorsAcrossPositionalAndJSON` checks "already exists", self-link and conflict on both CLI paths; MCP calls the same `Client.Link`.
- [x] Go tests cover each path: core rule, grandfathering, `Client.Link` (incl. derived-side overwrite and `TestLink_Errors`), `AddIssue`, MCP handler, plain-text and JSON CLI.

`make test` passes. Approved by the user after review.
