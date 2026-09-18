# Spec: Extract shared JSON I/O package from MCP server

## What

Create `internal/jsonio/` containing all JSON-serializable output types, input types,
and model-to-JSON converters currently private to `internal/mcpserver/tools.go`. The MCP
server becomes a thin adapter that imports these types rather than owning them.

## Why

This is the foundation for the unified JSON I/O epic (xpo-a4a271). Every downstream story
— `--json` output on read commands, input schema alignment on mutation commands — needs a
shared package to import. Today the types live inside `mcpserver` as unexported structs,
so no other package can use them.

## Current State

- **`internal/mcpserver/tools.go`** owns ~18 output structs, ~13 input structs, 4 converter
  functions, and the `flexStrings` helper — all unexported.
- **`internal/inputs/`** already shares `AddInput`, `UpdateInput`, `CommentInput`, `LinkInput`
  and their validation/conversion helpers between CLI and MCP. Left untouched in this story.
- **`internal/server/responses.go`** (HTTP API) has its own parallel response types —
  out of scope.

## Acceptance Criteria

1. `internal/jsonio/` package exists with exported output types, input types, and converters.
2. `internal/inputs/` is unchanged.
3. `internal/mcpserver/tools.go` imports from `jsonio` — no local type definitions remain
   for types that moved.
4. No behavior change — MCP server produces identical JSON output.
5. `make test` passes.

## Flow

### Step 1: Create `internal/jsonio/` package

Two files:

- **`output.go`** — all output/view types and converters:
  - `IssueSummary` (was `issueSummary`)
  - `CommentSummary` (was `commentSummary`)
  - `EventSummary` (was `eventSummary`)
  - `ListOutput` (was `listOut`)
  - `ShowOutput` (was `showOut`)
  - `HistoryOutput` (was `historyOut`)
  - `AddOutput` (was `addOut`)
  - `UpdateOutput` (was `updateOut`)
  - `CommentOutput` (was `commentOut`)
  - `StartOutput` (was `startOut`)
  - `MergeOutput` (was `mergeOut`)
  - `LinkOutput` (was `linkOut`)
  - `SpecOutput` (was `specOut`)
  - `WalkthroughOutput` (was `walkthroughOut`)
  - `ArtifactEntry` (was `artifactOutEntry`)
  - `ArtifactOutput` (was `artifactOut`)
  - `RationaleOutput` (was `rationaleOut`)
  - `RationaleHit` (was `rationaleHit`)
  - `ToIssueSummary()` (was `toSummary`)
  - `ToArtifactEntries()` (was `toArtifactEntries`)
  - `ToCommentSummaries()` (was `toCommentSummaries`)
  - `ToEventSummaries()` (was `toEventSummaries`)

- **`input.go`** — MCP tool parameter types:
  - `ListInput` (was `listIn`) + `FlexStrings` (was `flexStrings`)
  - `ShowInput` (was `showIn`)
  - `HistoryInput` (was `historyIn`)
  - `UpdateInput` (was `updateIn` — wraps `inputs.UpdateInput` with ID)
  - `CommentInput` (was `commentIn`)
  - `StartInput` (was `startIn`)
  - `MergeInput` (was `mergeIn`)
  - `LinkInput` (was `linkIn`)
  - `SpecInput` (was `specIn`)
  - `WalkthroughInput` (was `walkthroughIn`)
  - `ArtifactInput` (was `artifactIn`)
  - `RationaleInput` (was `rationaleIn`)

  Name collisions with `inputs` package (`CommentInput`, `LinkInput`, `UpdateInput`)
  are accepted — the package qualifier disambiguates (`jsonio.CommentInput` vs
  `inputs.CommentInput`). Downstream stories will migrate CLI commands to `jsonio`
  and then `inputs` can be removed.

### Step 2: Update `internal/mcpserver/tools.go`

- Remove all moved type definitions and converter functions.
- Add `import "github.com/palarix/exponential/internal/jsonio"`.
- Update all references: `issueSummary` → `jsonio.IssueSummary`, `toSummary(...)` →
  `jsonio.ToIssueSummary(...)`, etc.
- The `toolset` struct, tool registration, handler methods, and `textResult` stay in
  `mcpserver`.

### Step 3: Update tests

- `mcpserver` tests that reference moved types update to use `jsonio.*`.
- No new tests needed — existing MCP server tests verify behavior through handler
  round-trips. Converters are exercised indirectly.

## Decisions

1. **New package `jsonio`, leave `inputs` untouched** — `inputs` continues to serve the
   CLI and MCP for mutation payloads. Downstream stories migrate CLI commands to `jsonio`
   and delete `inputs` afterwards.

2. **Accept name collisions** — `jsonio.CommentInput` vs `inputs.CommentInput`,
   `jsonio.LinkInput` vs `inputs.LinkInput`, `jsonio.UpdateInput` vs `inputs.UpdateInput`.
   Package qualifiers disambiguate. No prefixes needed.

3. **Two files, not one per type** — `output.go` and `input.go` is a natural split matching
   the I/O direction. 30+ individual files would be excessive.

4. **`textResult` stays in `mcpserver`** — it wraps `mcp.CallToolResult`, which is
   MCP-SDK-specific. Not a JSON I/O concern.

5. **All types exported** — even types only used by `mcpserver` today, because downstream
   stories need them for CLI `--json` output.

## Edge Cases

- **JSON tag preservation** (HIGH) — all `json:` and `jsonschema:` struct tags must be
  preserved exactly. Any change breaks MCP clients or JSON output contracts.
- **`flexStrings` custom unmarshaler** (LOW) — moves as `FlexStrings`, `UnmarshalJSON`
  unchanged.
- **`updateIn` embedding** (LOW) — `jsonio.UpdateInput` embeds `inputs.UpdateInput` and
  adds the `ID` field, preserving the current composition. `jsonio` imports `inputs`.

## Assumptions

- `internal/inputs/` is left untouched. Downstream stories handle migration and removal.
- HTTP server types in `internal/server/responses.go` are out of scope.
- No new unit tests for `jsonio` — existing MCP server tests provide coverage.
