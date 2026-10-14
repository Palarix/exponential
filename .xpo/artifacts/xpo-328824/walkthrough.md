# Walkthrough: MCP update: agents cannot clear an issue's parent

## What was built

Agents can now remove an issue's parent through MCP `update`. `parent: null`, `parent: ""` and the literal string `"\"\""` all clear it. CLI `xpo update --json` behaves the same way, because it shares `jsonio.UpdateInput` with MCP.

## The two failures

An agent in a `cs-` project tried to make three epics top-level. It failed in two ways:

1. **`parent: null` → "no fields set".** `UpdateInput.Parent` is a `*string`. `encoding/json` decodes an explicit `null` and an omitted field to the same `nil`. The handler saw no fields and rejected the call. When another field was set too, the update succeeded and the parent stayed unchanged.
2. **`parent: "\"\""` → `parent: issue "" not found`.** The format string is `issue %s not found`, with no `%q`. So the quotes in the message were part of the ID. The agent sent two quote characters while it tried to say "empty". A real empty string already worked, because `ValidateUpdatePayload` skips the lookup for `""`.

## How the pieces fit

```
raw JSON args ─┬─> SDK decode ─> UpdateToolInput (Parent nil for null and omitted)
               └─> jsonio.ApplyExplicitNulls(raw, &in) ─> Parent = &"" when "parent": null
                                    │
                                    v
                 UpdateInput.ToUpdatePayload ─> normalizeParent: trim; `""` / `''` -> ""
                                    │
                                    v
                 ValidateUpdatePayload (skips lookup for "") ─> UpdateIssue clears parent
```

- `internal/jsonio/input.go`
  - `ApplyExplicitNulls` decodes the raw object into `map[string]json.RawMessage`. When `parent` is `null`, it sets `Parent` to `""`.
  - `normalizeParent` trims the value. It maps a quoted empty string to `""`.
  - The `parent` field gets a schema description that says how to clear it.
- `internal/mcpserver/tools.go`: `update` calls `ApplyExplicitNulls` with `req.Params.Arguments`. Unit tests call the handler with a `nil` request, so the handler checks `req` first.
- `cmd/exponential/update.go`: the `--json` path calls `ApplyExplicitNulls` on stdin after `DecodeStrict`.

## Decisions

- **A helper function, not `UnmarshalJSON`.** `UpdateToolInput` embeds `UpdateInput`. A custom `UnmarshalJSON` on `UpdateInput` would be promoted to `UpdateToolInput`, which would then stop decoding `id`.
- **Read the raw arguments, not a custom nullable type.** The SDK generates the input schema from the Go types. A custom struct type would change the schema from `["null","string"]` to an object.
- **Accept the literal quoted empty string.** No issue ID contains quote characters, so this mapping never hides a real ID. It turns a confusing error into the result the agent intended.
- **Only `parent` in this issue.** `assignee`, `cycle_id`, `priority` and `story_points` have the same null problem. xpo-bbfcf9 tracks them and extends `ApplyExplicitNulls`.

## Tests

- `internal/mcpserver/integration_test.go`
  - `TestEndToEndUpdateClearsParent` goes through the in-memory MCP client, so the raw arguments reach the handler. It covers null, `""`, quoted `""`, null with another field, and an omitted parent. Before the fix, it reproduced both errors from the agent session exactly.
  - `TestUpdateParentSchemaDocumentsClearing` checks the schema description.
- `internal/jsonio/jsonio_test.go`: unit tests for `ApplyExplicitNulls` and `normalizeParent`.
- `cmd/exponential/parent_test.go`: `update --json` with `{"parent": null}`.

## Acceptance criteria

- [x] MCP `update {id, parent: null}` clears the parent. Evidence: `TestEndToEndUpdateClearsParent/null`.
- [x] MCP `update {id, parent: ""}` clears the parent. Evidence: `TestEndToEndUpdateClearsParent/empty_string`.
- [x] MCP `update {id, parent: "\"\""}` clears the parent. Evidence: `TestEndToEndUpdateClearsParent/quoted_empty_string`.
- [x] `parent: null` with another field clears the parent and applies the other field. Evidence: `TestEndToEndUpdateClearsParent/null_with_another_field`.
- [x] An omitted `parent` stays unchanged. Evidence: `TestEndToEndUpdateClearsParent/omitted_parent_is_unchanged`.
- [x] CLI `update --json` with `{"parent": null}` clears the parent. Evidence: `TestUpdateJSONNullParentClears`.
- [x] The `update` tool's `parent` schema description mentions null and `""`. Evidence: `TestUpdateParentSchemaDocumentsClearing`.
- [x] `make test` passes. Evidence: exit code 0.
