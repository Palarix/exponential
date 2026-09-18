## Walkthrough: Code review of Unified JSON I/O epic

## Scope

Full review of the combined diff across 14 commits in the Unified JSON I/O epic
(xpo-a4a271). 53 files changed, +3176/-767 lines. Reviewed across four dimensions
using parallel review agents.

## Findings summary

### Error contract — CLEAN

Every error path reachable when `--json` is active calls `exitJSONError` or flows
through `main()`'s centralized handler. Verified across all 13 CLI commands plus
cobra arg validation and PersistentPreRunE pre-handler errors.

### Consistency — cosmetic only

Flag variable naming splits two conventions: `xxxJSONFlag` (11 commands) vs `xxxJSON`
(history, pulse — predating the epic). Help text varies across commands. One lowercase
initial (`pulse.go`). No functional impact.

### MCP parity — one bug, two minor gaps

**Bug (xpo-eb0d61):** `comment --json` decodes into `CommentInput` (body only) while
the MCP tool sends `CommentToolInput` (id + body). `DecodeStrict` rejects the MCP
payload due to the unknown `id` field, breaking the "one contract" principle.

**Minor:** `update` empty-payload error message differs from MCP wording. `list` doesn't
validate status filters like MCP does. `show` and `history` outputs are strict supersets
of MCP (intentional — CLI has richer data sources).

### Dead code — one cleanup needed

`internal/inputs` shim has zero production consumers. Only
`internal/mcpserver/tools_test.go` still imports it. The package should be deleted and
its test coverage migrated to `internal/jsonio/` (which currently has no test files).

## Filed follow-ups

| Issue | Priority | Description |
|-------|----------|-------------|
| xpo-aebfc5 | P1 | Remove `internal/inputs` shim, migrate tests to jsonio |
| xpo-eb0d61 | P2 | Fix `comment --json` to accept MCP CommentToolInput |
| xpo-66b8af | P3 | Cosmetic consistency pass (flag names, help text, etc.) |
