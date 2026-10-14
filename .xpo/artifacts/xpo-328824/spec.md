# Spec: MCP update: agents cannot clear an issue's parent

## What

Let agents clear a parent through MCP `update`:

- `parent: null` clears it.
- `parent: ""` clears it. This already works.
- `parent: "\"\""` clears it too. This is a two-character string of literal quote marks, which agents send when trying to express "empty".

The `parent` schema description must document how to clear it.

## Why

The `update` schema advertises `parent` as `["null","string"]` with no description. Agents trying to remove a parent hit dead ends (screenshot from a `cs-` project session, 2026-10-14):

1. `update {id, parent: null}` → `no fields set: provide at least one field to update`
2. Several `InputValidationError`s from the harness while the agent guesses at shapes
3. `update {id, parent: "\"\""}` → `parent: issue "" not found`

## Investigation

I reproduced this against the installed `xpo mcp` (local transport) in a throwaway workspace:

| Call | Result |
|---|---|
| `parent: ""` | Works. The parent is cleared. |
| `parent: null` | `no fields set` |
| `parent: null` + `title` | The title changes and the parent is silently kept |

**Root cause of the null failure:** `jsonio.UpdateInput.Parent` is a `*string`. `encoding/json` decodes both an omitted field and an explicit `null` to `nil`, so the handler can't tell "leave it alone" from "clear it".

**Root cause of `issue "" not found`:** the binary only has the format `issue %s not found` and no `%q`. So the quotes in the error were part of the ID itself. The agent sent the literal two-character string `""`.

## How

1. **`internal/jsonio/input.go`**
   - Add a helper that takes the raw JSON arguments and sets `Parent = &""` when the object contains `"parent": null`. It must be a function and not `UnmarshalJSON` on `UpdateInput`. `UpdateToolInput` embeds `UpdateInput`, so a promoted `UnmarshalJSON` would swallow `id`.
   - In `ToUpdatePayload`, normalize `Parent`: trim whitespace, and treat the literal `""` or `''` as empty.
2. **`internal/mcpserver/tools.go` `update`**: apply the null helper to `req.Params.Arguments` when `req` is non-nil.
3. **`cmd/exponential/update.go` `--json`**: apply the same helper to stdin so the CLI JSON path matches MCP. This follows the shared-schema precedent from xpo-199282.
4. **Schema description** for `parent` on `update`: `Parent issue ID. Pass null or "" to remove the parent and make the issue top-level`.

## Decisions

- **Accept literal `""` as empty.** No issue ID can contain quote characters, so this can never mask a real ID. It turns a confusing error into the obvious intent.

## Acceptance criteria

- [ ] MCP `update {id, parent: null}` clears the parent (end-to-end test through the in-memory MCP client)
- [ ] MCP `update {id, parent: ""}` clears the parent (end-to-end test)
- [ ] MCP `update {id, parent: "\"\""}` clears the parent
- [ ] `parent: null` combined with another field clears the parent and applies the other field
- [ ] Omitting `parent` still leaves it unchanged
- [ ] CLI `update --json` with `{"parent": null}` clears the parent
- [ ] The `update` tool's `parent` schema description mentions null and ""
- [ ] `make test` passes

## Out of scope

`assignee`, `cycle_id`, `priority` and `story_points` have the same null-vs-omitted ambiguity. I'll file a follow-up instead of widening this fix.
