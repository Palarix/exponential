# Walkthrough: Extract shared JSON I/O package from MCP server

## What changed

Created `internal/jsonio/` as the single source of truth for all JSON-serializable types
shared between the MCP server and (eventually) CLI `--json` mode. Refactored `internal/mcpserver/tools.go`
to import from it, and reduced `internal/inputs/` to a backward-compatibility shim.

## Files

### `internal/jsonio/output.go` (new)

All output/view types that represent JSON responses from MCP tools and will be reused for
CLI `--json` output in downstream stories:

- **Summary types**: `IssueSummary`, `CommentSummary`, `EventSummary` — lightweight
  projections of `model.Issue`, `model.Comment`, `model.Event` with timestamps formatted
  as RFC3339 strings (not `time.Time`).
- **Envelope types**: `ListOutput`, `ShowOutput`, `HistoryOutput`, `AddOutput`, `UpdateOutput`,
  `CommentOutput`, `StartOutput`, `MergeOutput`, `LinkOutput`, `SpecOutput`, `WalkthroughOutput`,
  `ArtifactOutput`, `ArtifactEntry`, `RationaleOutput`, `RationaleHit`.
- **Converter functions**: `ToIssueSummary()`, `ToArtifactEntries()`, `ToCommentSummaries()`,
  `ToEventSummaries()` — each takes the corresponding `model.*` type and returns the
  JSON-serializable equivalent.

Dependencies: `internal/model` (for `BranchStats`, `Dependency`, `Issue`, `ArtifactSummary`,
`Comment`, `Event`) and `time` (for RFC3339 formatting). No MCP SDK dependency.

### `internal/jsonio/input.go` (new)

Two layers of input types:

**Shared payload types** (used by both CLI `--json` and MCP tools):
- `AddInput`, `UpdateInput`, `CommentInput`, `LinkInput` — these were previously in
  `internal/inputs/`. They define the agent-facing JSON schemas with `json:` and `jsonschema:`
  struct tags. Methods `ToCreatePayload()` and `ToUpdatePayload()` validate and convert to
  the internal `model.CreatePayload` / `model.UpdatePayload`.
- Helper functions: `DecodeStrict()`, `ValidateStatus()`, `LinksToDependencies()`, `UpdatePayloadEmpty()`.

**MCP tool parameter types** (the full tool input including issue ID):
- `ListToolInput`, `ShowToolInput`, `HistoryToolInput`, `UpdateToolInput`, `CommentToolInput`,
  `StartToolInput`, `MergeToolInput`, `LinkToolInput`, `SpecToolInput`, `WalkthroughToolInput`,
  `ArtifactToolInput`, `RationaleToolInput`.
- The `*ToolInput` suffix distinguishes these from the shared payload types. For example,
  `UpdateToolInput` embeds `UpdateInput` and adds an `ID` field — the MCP SDK derives the
  JSON schema from the combined struct.
- `FlexStrings` — custom `[]string` type with `UnmarshalJSON` that accepts either a single
  string or an array, used by `ListToolInput.Status`.

### `internal/inputs/inputs.go` (rewritten → shim)

Reduced from ~165 lines to ~32 lines. Now contains:
- Type aliases: `type AddInput = jsonio.AddInput`, etc.
- Function wrappers: `func DecodeStrict(...) { return jsonio.DecodeStrict(...) }`, etc.

The three CLI commands (`cmd/exponential/add.go`, `update.go`, `comment.go`) continue to
import `inputs` and compile without changes. Downstream stories will migrate them to import
`jsonio` directly, after which `inputs` can be deleted.

### `internal/mcpserver/tools.go` (refactored)

Removed: all type definitions (~235 lines of structs), all converter functions (~70 lines),
`FlexStrings` type and unmarshaler. The `encoding/json` and `time` imports are also removed.

Retained: `toolset` struct, `register()` method, all 14 handler methods, `textResult()` helper.

Each handler signature changed from private types to `jsonio.*ToolInput` / `jsonio.*Output`.
The `inputs` import was removed — `UpdatePayloadEmpty` is now called via `jsonio.UpdatePayloadEmpty`.

### Test files (3 updated)

All type references updated to `jsonio.*ToolInput` / `jsonio.*Output`. The `jsonio` import
was added. No test logic changed — only type names.

## Key decisions

1. **Alias direction**: real code in `jsonio`, shim in `inputs` (not the reverse). This was
   a course correction during review. The original implementation had `jsonio.AddInput = inputs.AddInput`,
   which would have required moving code *again* when `inputs` is deleted. With the current
   direction, downstream stories just delete the shim — no code moves.

2. **`*ToolInput` naming**: MCP tool parameter types (which add `ID` to a shared payload)
   use the `ToolInput` suffix to avoid collisions within the `jsonio` package. The shared
   payload types keep clean names (`AddInput`, `UpdateInput`) since those are the canonical
   schemas that CLI and MCP both converge on.

3. **`add` handler uses `jsonio.AddInput` directly**: unlike other mutation tools, the `add`
   handler doesn't need an `AddToolInput` wrapper because creating an issue doesn't require
   an existing issue ID. The `AddInput` payload *is* the complete tool input.
