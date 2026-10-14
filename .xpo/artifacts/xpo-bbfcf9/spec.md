# Spec: MCP update: null is ignored for assignee, cycle_id, priority and story_points

## What

Make an explicit `null` on `assignee`, `cycle_id`, `priority` and `story_points` clear that field in MCP `update` and `xpo update --json`. Document clearing in each field's schema description.

## Why

The `update` schema advertises these fields as nullable. Today `null` decodes to `nil`, the same as an omitted field. A call with only a null field fails with "no fields set". A call with another field leaves the null field unchanged without an error. xpo-328824 fixed this for `parent` only.

## How

1. In `jsonio.ApplyExplicitNulls`, map each null field to its cleared value:

   | Field | Cleared value | Existing meaning |
   |---|---|---|
   | `assignee` | `""` | Unassigned. `validateAssigneeFormat("")` accepts it. |
   | `cycle_id` | `""` | No cycle. `ValidateUpdatePayload` skips the cycle check for `""`. |
   | `priority` | `0` | "None" (0=None, 1=Urgent … 4=Low). |
   | `story_points` | `0` | Unestimated. `ValidateUpdatePayload` skips the estimate check for `0`. |

2. Drive the mapping from a table in `input.go`, so the next nullable field is one row.
3. Add schema descriptions:
   - `assignee`: "Assignee in 'Name <email>' format. Pass null or \"\" to unassign"
   - `cycle_id`: append "Pass null or \"\" to remove the issue from its cycle"
   - `priority`: "Priority: 0=None, 1=Urgent, 2=High, 3=Medium, 4=Low. Pass null or 0 to clear". The jsonschema tag must not begin with `WORD=`, so the description starts with "Priority:".
   - `story_points`: "Effort estimate in story points. Pass null or 0 to clear"
4. In `internal/ui/timeline.go`, describe cleared values in words instead of printing an empty or zero value:
   - "unassigned" instead of "assigned to "
   - "cleared priority" instead of "set priority to 0"
   - "cleared estimate" instead of "set estimate to 0"

## Not changed

- `title`, `status` and `description`: `null` stays "unchanged". Title and status can't be empty. See Decisions for `description`.
- `labels` and `links`: `[]` already clears them. `null` stays "unchanged". See Decisions.
- The literal quoted-empty mapping from xpo-328824 stays limited to `parent`. Only `parent` was seen failing that way.
- The web activity timeline has the same problem with cleared values. xpo-ca763f tracks it.

## Acceptance criteria

- [ ] MCP `update {id, assignee: null}` unassigns the issue (end-to-end test)
- [ ] MCP `update {id, cycle_id: null}` removes the issue from its cycle (end-to-end test)
- [ ] MCP `update {id, priority: null}` sets priority to 0 (end-to-end test)
- [ ] MCP `update {id, story_points: null}` sets story points to 0 (end-to-end test)
- [ ] A null field combined with another field clears the null field and applies the other field
- [ ] An omitted field stays unchanged
- [ ] `description: null`, `labels: null` and `links: null` with another field leave description, labels and links unchanged
- [ ] CLI `update --json` with a null field clears that field
- [ ] Each of the four fields has a schema description that explains how to clear it
- [ ] The timeline shows "unassigned", "cleared priority" and "cleared estimate" for cleared values
- [ ] `make test` passes

## Decisions

- **Omitted means unchanged. `null` clears scalar fields.** This follows JSON Merge Patch (RFC 7396) and Rails `update(field: nil)`. The handler can tell the two apart on the wire: `ApplyExplicitNulls` reads the raw arguments.
- **`null` does not clear `description`, `labels` or `links`.** Only `""` or `[]` clears them. Some tool callers send `null` for every unused field, for example when a client rewrites a schema for OpenAI strict mode. Under "`null` clears", such a caller would erase content. On scalar fields, a mistaken clear is small and easy to undo.
- **The strict-mode risk is low for current clients.** Codex sends MCP tools with `strict: false` (`codex-rs/tools/src/responses_api.rs`, checked 2026-10-14). Claude Code leaves unused fields out. Our schema lists only `id` as required, so strict mode needs a deliberate schema rewrite by the client.
