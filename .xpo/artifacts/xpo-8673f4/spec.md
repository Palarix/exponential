# Agent chat: show denied and failed tool calls in the work list

## What

The chat runner reads each tool result from the Claude stream. It settles every `tool_call` event with a real outcome: `done`, `error`, or `denied`, plus a short `message`. The work list shows a red marker and the error text for a failed or denied step.

## Why

Follow-up to xpo-56ae3a, part of xpo-3798b7. Today the parser emits `StreamToolDone` at `content_block_stop`. That is when the model finishes writing the tool input, before the tool runs. The runner settles `tool_call` with `status: "done"` at that point. So a denied `Read /etc/hosts` looks like a successful read.

## Stream facts (captured from `claude -p --output-format stream-json --verbose`, Claude Code current)

- A `tool_use` content block carries `id` (`toolu_…`). `content_block_start` has it in `content_block.id`.
- When Claude Code denies a call, it first emits `{"type":"system","subtype":"permission_denied","tool_use_id":…,"message":…}`.
- Every result arrives as `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":…,"is_error":true,"content":…}]}}`.
- `content` is a string, or an array of blocks (`{"type":"text","text":…}`, images).
- A denied result also has `tool_result_meta[].non_execution_kind: "user-rejected"`.
- The final `result` message lists `permission_denials[]` with `tool_use_id`. It arrives too late to be useful here.
- Messages from inside a subagent have a non-null `parent_tool_use_id`.

## Acceptance Criteria

- [ ] A denied call ("read /etc/hosts") settles as `tool_call` with `status: "denied"` and the denial text in `message`.
- [ ] A failed call (Bash with non-zero exit, MCP tool error) settles with `status: "error"` and the error text in `message`.
- [ ] A successful call settles with `status: "done"` and no `message`.
- [ ] The step keeps its spinner while the tool runs. It settles when the result arrives, not when the input is complete.
- [ ] Two parallel calls with the same tool name settle on the correct steps. Matching uses the tool use ID.
- [ ] A call with no result at turn end (stop, crash) settles with `status: "done"`, as today.
- [ ] The work list shows a red `×` marker for `error` and `denied` steps.
- [ ] The expanded work list shows the message below the step, in red, clamped to 3 lines. The full text is in the tooltip.
- [ ] A denied step's message starts with "Denied:".
- [ ] The collapsed header shows a red failure count, for example "Worked for 4 seconds · 1 failed".
- [ ] Chats saved before this change replay without errors. Their tool steps show as done.
- [ ] `xpo drive` output does not change.
- [ ] `make test` passes.

## Flow

### Parser (`internal/exponential/claudestream.go`, `agent.go`)

1. Add `ToolID`, `ToolStatus` and `ToolMessage` fields to `StreamEvent`.
2. Add a new kind `StreamToolResult` after the existing kinds.
3. In `content_block_start`, read `id` for a `tool_use` block. Store it in `p.currentToolID`.
4. Set `ToolID` on the `StreamToolStart` and `StreamToolDone` events of that block.
5. Track denied IDs in a `denied map[string]string` (ID to message). Fill it from `system`/`permission_denied` messages.
6. Parse `user` messages. For each `tool_result` block, emit one `StreamToolResult` event.
7. Set `ToolStatus` to `denied` if the ID is in `denied`, or if `tool_result_meta` marks it `user-rejected`.
8. Otherwise set `ToolStatus` to `error` when `is_error` is true, else `done`.
9. Set `ToolMessage` for `error` and `denied` only. Use `toolResultText`.
10. Skip `user` messages with a non-null `parent_tool_use_id`.
11. Add `toolResultText`: join the text blocks, trim, strip a leading `Error: ` prefix, and cut to 500 runes with `…`.

### Runner (`internal/chat/runner.go`, `event.go`)

12. Add `ToolID string json:"tool_id,omitempty"` to `Event`. Use it on `tool_start` and `tool_call`.
13. Document the `Status` values `done`, `error`, `denied`. Document `Message` for `tool_call`.
14. On `StreamToolDone` with a tool name, store the call in a `pending` map keyed by ID. Keep the insertion order. Do not settle yet.
15. On `StreamToolResult`, take the call from `pending`. Settle `tool_call` with the status and message. Ignore unknown IDs.
16. After the scan loop ends, settle each remaining pending call with `status: "done"`, in order.
17. A `StreamToolDone` with an empty `ToolID` settles at once with `done`. This keeps the old path for agents without IDs.

### Frontend (`web/src/api/chat.ts`, `chat-thread-utils.ts`, `ChatThread.tsx`)

18. Add `tool_id?: string` to `ChatEvent`. Document the `status` values.
19. Add `id?: string`, `status?: "done" | "error" | "denied"` and `message?: string` to the tool `ChatStep`.
20. In `tool_start` and `tool_call`, match the open step by `id` when the event has one. Fall back to the name match.
21. In `WorkStep`, show a red `×` in the bullet slot for `error` and `denied`.
22. Below a failed step, show `message` in a red, 3-line-clamped block. Indent it to the text column.
23. Prefix the message of a `denied` step with "Denied: ".
24. In the collapsed header, add " · N failed" in red when N > 0 steps have `error` or `denied`.
25. Use the existing `--color-error` token.

### Tests (TDD, write first)

26. `claudestream_test.go`: denied Read, failed Bash, success, array content, subagent result skipped, ID on start/done events.
27. `runner_test.go`: settle at result time, parallel same-name calls, pending flush at turn end.
28. `chat-thread-utils.test.ts`: match by ID, status and message on the step, old events without ID.

## Decisions

- **Settle at result time, not at block stop.** One settled event per call holds the final outcome. The alternative is a second `tool_result` event that patches the first. That doubles the event log and the replay logic.
- **New `StreamToolResult` kind, not new `StreamToolDone` semantics.** `xpo drive` uses `StreamToolDone` to clear its activity line. Drive does not change.
- **Denied detection uses `permission_denied` and `tool_result_meta`, not text matching.** Both are structured. Text heuristics break when Claude Code rewords messages.
- **Unresolved calls at turn end show as `done`.** This matches today's behavior. An "unknown" status adds UI for a rare case.
- **The collapsed header shows a red failure count.** You see failures without expanding the list. (User decision.)
- **Denied and error share the red `×`. A denied message starts with "Denied:".** One marker keeps the list calm. (User decision.)

## Edge Cases

- HIGH: none.
- MEDIUM: A tool result can be very long (Bash stderr). The parser cuts it to 500 runes. The UI clamps it to 3 lines.
- MEDIUM: A turn stops while a tool runs. The pending call settles as `done` after the scan loop.
- LOW: A result for an unknown ID (subagent, replay) is ignored.
- LOW: Old chat logs have no `tool_id` and no `error` status. The name fallback handles them.

## Assumptions

- Only Claude is a chat agent today. Other agents keep the old path (step 17).
- `--include-partial-messages` streams include `id` in `content_block_start` for `tool_use`. The API documents this.

## Open Questions

None. The user answered both questions on 2026-10-12.
