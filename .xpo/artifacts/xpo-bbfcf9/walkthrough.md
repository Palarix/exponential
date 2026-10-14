# Walkthrough: MCP update: null is ignored for assignee, cycle_id, priority and story_points

## What was built

An explicit `null` now clears `assignee`, `cycle_id`, `priority` and `story_points` in MCP `update` and `xpo update --json`. xpo-328824 did the same for `parent`. `null` on `description`, `labels` and `links` keeps meaning "unchanged".

| Field | `null` sets | Same as |
|---|---|---|
| `parent`, `assignee`, `cycle_id` | `""` | sending `""` |
| `priority`, `story_points` | `0` | sending `0` |
| `description`, `labels`, `links` | nothing | — |

## How the pieces fit

`internal/jsonio/input.go` holds a `nullClears` table. Each row maps a JSON field name to a setter for its cleared value. `ApplyExplicitNulls` decodes the raw arguments into `map[string]json.RawMessage`. It runs the setter for every field that has a row and is `null`. A generic `ptr[T]` helper keeps each row on one line.

The callers did not change. MCP `update` in `internal/mcpserver/tools.go` and the `--json` path in `cmd/exponential/update.go` already call `ApplyExplicitNulls` since xpo-328824.

The cleared values pass the existing checks without changes:

- `validateAssigneeFormat("")` returns nil.
- `ValidateUpdatePayload` skips the cycle check for `""` and the estimate check for `0`.
- The priority range check accepts `0` ("None").

`internal/ui/timeline.go` (`describeUpdate`) now writes "unassigned", "cleared priority" and "cleared estimate" for cleared values. It used to print "assigned to ", "set priority to 0" and "set estimate to 0".

## Decisions

- **Omitted means unchanged. `null` clears scalar fields.** This follows JSON Merge Patch (RFC 7396) and Rails `update(field: nil)`. The handler tells the two apart by reading the raw arguments, so the convention costs nothing.
- **`null` does not clear `description`, `labels` or `links`.** Some tool callers send `null` for every unused field. For example, a client can rewrite a schema for OpenAI strict mode, which makes every field required and nullable. Under "`null` clears", that caller would erase content. On scalar fields, a mistaken clear is small and easy to undo. The fields are left out of `nullClears`, and the table's comment says why.
- **The strict-mode risk is low today.** Codex sends MCP tools with `strict: false` (`codex-rs/tools/src/responses_api.rs`, checked 2026-10-14). Claude Code leaves unused fields out. Our schema requires only `id`, so strict mode needs a deliberate schema rewrite by the client.
- **The priority description starts with "Priority:".** The jsonschema tag parser rejects a tag that begins with `WORD=`, and "0=None…" matched that pattern. The server panicked at tool registration until the text changed.

## Follow-up

xpo-ca763f: the web activity timeline shows "set estimate to 0" and nothing for a cleared assignee. It also never describes priority, parent or cycle changes.

## Tests

- `internal/mcpserver/null_fields_test.go`
  - `TestEndToEndUpdateNullClearsScalarFields` goes through the in-memory MCP client. It sets all four fields, then clears each one alone and together with a title change. It also checks that omitted fields stay unchanged, and that `null` on description, labels and links erases nothing.
  - `TestUpdateSchemaDocumentsClearingScalarFields` checks the four descriptions.
- `internal/jsonio/jsonio_test.go`: `TestApplyExplicitNullsClearsScalarFields` and `TestApplyExplicitNullsLeavesContentFields`.
- `cmd/exponential/parent_test.go`: `TestUpdateJSONNullScalarFieldsClear` for the CLI path.
- `internal/ui/timeline_test.go`: `TestDescribeEvent_UpdateClearedScalarFields`.

## Acceptance criteria

- [x] MCP `update {id, assignee: null}` unassigns the issue. Evidence: `TestEndToEndUpdateNullClearsScalarFields/assignee`.
- [x] MCP `update {id, cycle_id: null}` removes the issue from its cycle. Evidence: `TestEndToEndUpdateNullClearsScalarFields/cycle_id`.
- [x] MCP `update {id, priority: null}` sets priority to 0. Evidence: `TestEndToEndUpdateNullClearsScalarFields/priority`.
- [x] MCP `update {id, story_points: null}` sets story points to 0. Evidence: `TestEndToEndUpdateNullClearsScalarFields/story_points`.
- [x] A null field with another field clears the null field and applies the other field. Evidence: the `… with another field` subtests.
- [x] An omitted field stays unchanged. Evidence: `TestEndToEndUpdateNullClearsScalarFields/omitted_fields_are_unchanged`.
- [x] `description: null`, `labels: null` and `links: null` leave those fields unchanged. Evidence: `TestEndToEndUpdateNullClearsScalarFields/null_leaves_description_labels_and_links_unchanged` and `TestApplyExplicitNullsLeavesContentFields`.
- [x] CLI `update --json` with a null field clears that field. Evidence: `TestUpdateJSONNullScalarFieldsClear`.
- [x] Each of the four fields has a schema description that explains how to clear it. Evidence: `TestUpdateSchemaDocumentsClearingScalarFields`.
- [x] The timeline shows "unassigned", "cleared priority" and "cleared estimate". Evidence: `TestDescribeEvent_UpdateClearedScalarFields`.
- [x] `make test` passes. Evidence: exit code 0.
