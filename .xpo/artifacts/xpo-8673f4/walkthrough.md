# Walkthrough: show denied and failed tool calls in the work list

## What changed

The agent chat used to mark every tool call `done`. The runner settled the call when the model finished writing the tool input, so the tool had not run yet. A denied `Read /etc/hosts` looked like a successful read.

Now the parser reads each tool result from the Claude stream. The runner settles the call when the result arrives. Each call has the status `done`, `error` or `denied`, plus the error text. The work list shows a red `×`, the message, and a failure count on the collapsed header.

## How the pieces fit

### 1. Parser: `internal/exponential/claudestream.go`

The parser learns three new facts from the stream:

- **Tool use ID.** `content_block_start` for a `tool_use` block carries `id`. The parser stores it in `currentToolID` and sets `ToolID` on the `StreamToolStart` and `StreamToolDone` events.
- **Denials.** Claude Code emits `system`/`permission_denied` with `tool_use_id` before the result. The parser records the ID in `denied`.
- **Results.** A `user` message holds `tool_result` blocks. `handleToolResults` emits one `StreamToolResult` per block, with `ToolStatus` and `ToolMessage`.

The status rule: `denied` if `permission_denied` named the ID, or if `tool_result_meta` says `user-rejected`. Otherwise `error` if `is_error` is true. Otherwise `done`. Both denial signals are structured, so a change to Claude Code's wording does not break the detection.

`toolResultText` turns the result content into a message. The content is a string or an array of blocks. The function joins the text blocks, strips Claude Code's `Error: ` prefix, and cuts the text to 500 runes.

A `user` message with a non-null `parent_tool_use_id` comes from inside a subagent. Its results are not steps of this turn, so the parser skips them.

`StreamToolResult` is a new kind. `StreamToolDone` keeps its meaning, because `xpo drive` uses it to clear its activity line. Drive ignores the new kind, so its output does not change.

### 2. Runner: `internal/chat/runner.go`, `event.go`

`Event` gets a `tool_id` field. `Status` now has three values, and `Message` carries the error text for `tool_call`.

The runner no longer settles a call on `StreamToolDone`. It puts the call in `pending`, keyed by tool ID, and records the order in `pendingOrder`. On `StreamToolResult`, it takes the call from `pending`, sets status and message, and settles it. An unknown ID is ignored.

After the scan loop, `settlePending` settles every call without a result as `done`, in the original order. That covers a stopped turn or a crashed process. It matches the behavior before this change.

A `StreamToolDone` without an ID settles at once as `done`. That is the old path, kept for streams that carry no IDs.

One settled event per call holds the final outcome. The alternative was a second `tool_result` event that patches the first. That doubles the event log and the replay logic.

### 3. Frontend: `chat-thread-utils.ts`, `ChatThread.tsx`, `api/chat.ts`

The tool `ChatStep` gets `id`, `status` and `message`. `openToolIndex` finds the step for an event by tool ID. For events without an ID (chats saved before this change) it falls back to the old rule: the last open step with the same name. The ID match is the reason two parallel `Read` calls now settle on the correct rows.

`failedToolCount` counts `error` and `denied` steps. The `Turn` header shows "· N failed" in red when the count is above zero.

`WorkStep` shows a red `X` icon in the bullet slot for a failed step. Below the row it shows the message in `--color-error`, clamped to 3 lines, with the full text in the tooltip. A denied message starts with "Denied:". The message indent is 18px: the 12px bullet slot plus the 6px gap. So the message lines up with the step text.

## Decisions from review

- The collapsed header shows a red failure count. (User decision.)
- Denied and error share the red `×`. A denied message starts with "Denied:". (User decision.)
- The spec named `--color-danger`. The board's token is `--color-error`, and the spec now says so.

## Acceptance criteria

- [x] A denied call settles with `status: "denied"` and the denial text. Evidence: `TestClaudeStreamParser_ToolResults`, `TestRunner_ToolCallsSettleWithResults`.
- [x] A failed call settles with `status: "error"` and the error text. Evidence: `TestClaudeStreamParser_ToolResults` (Bash with array content).
- [x] A successful call settles with `status: "done"` and no message. Evidence: same tests (Grep, `t-a`).
- [x] The step keeps its spinner until the result arrives. Evidence: the runner holds calls in `pending`. `TestRunner_ToolCallsSettleWithResults` shows the settle order follows the result order.
- [x] Parallel calls of one tool settle on the correct steps. Evidence: `TestRunner_ToolCallsSettleWithResults`, and the vitest case "matches parallel calls of one tool by ID".
- [x] A call with no result at turn end settles as `done`. Evidence: `t-c` in `TestRunner_ToolCallsSettleWithResults`.
- [x] The work list shows a red `×` for `error` and `denied`. Evidence: `WorkStep` in `ChatThread.tsx`. The user tested it in the board.
- [x] The expanded list shows the message in red, clamped to 3 lines, with the full text in the tooltip. Evidence: `WorkStep`. The user tested it.
- [x] A denied message starts with "Denied:". Evidence: `WorkStep`. The user tested it.
- [x] The collapsed header shows a red failure count. Evidence: vitest "counts failed and denied tool calls", and the `Turn` header.
- [x] Old chats replay without errors. Evidence: the existing name-based reducer tests still pass. The `TestRunner_TurnPersistsSettledEvents` fixture has no IDs and still settles.
- [x] `xpo drive` output does not change. Evidence: `StreamToolDone` keeps its meaning. Drive's callback has no case for `StreamToolResult`.
- [x] `make test` passes.

The stream shapes come from real `claude -p --output-format stream-json --verbose` traces: a denied `Read /etc/hosts`, a blocked `ls`, and a `--include-partial-messages` run. The run confirmed that `content_block_start` carries the same `id` as the `tool_result`.

## Non-obvious points

- `permission_denied` arrives before the result. The final `result` message also lists `permission_denials`, but it arrives too late to settle a call.
- The parser keeps `denied` for the whole turn. Use one parser per turn, as before.
- A real Bash call with a non-zero exit did not happen in the captured traces, because Claude Code blocked the command first. The `error` path rests on a fixture built from the documented `tool_result` shape.
