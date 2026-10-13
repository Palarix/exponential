# Agent chat: approval cards for tool calls (Claude and Codex)

## What

The agent can ask the user before a tool call runs. The question appears as an approval card in the chat transcript. The user allows or denies it, and the turn continues. This works for Claude and Codex. As the first use, `merge` in a chat asks before it runs.

## Why

xpo-00f0e5 (Chat Mode and Work Mode) needs it. In Chat Mode, `start` asks to enter Work Mode. In Work Mode, `merge` asks before it runs. Today neither agent can ask:

- `claude -p` runs without a permission tool. A call outside `--allowedTools` is denied at once.
- `codex.go` declines every app-server approval request.

Elysium solves this with the Claude Agent SDK `canUseTool` callback. The SDK launches `claude --permission-prompt-tool stdio` and answers `control_request` lines on stdin. xpo is Go, so it speaks the same protocol directly.

## Wire formats (measured 2026-10-13)

- **Claude Code 2.1.296:** `{"type":"control_request","request_id":…,"request":{"subtype":"can_use_tool","tool_name":…,"input":…,"permission_suggestions":[…],"tool_use_id":…}}`. The answer is `{"type":"control_response","response":{"subtype":"success","request_id":…,"response":{"behavior":"allow","updatedInput":…}}}` or `behavior: "deny"` with `message`. Suggestions carry `destination: "localSettings"`. MCP tools carry no suggestions.
- **Claude Code:** a `permissions.ask` rule in `--settings` beats an `--allowedTools` wildcard.
- **Claude Code:** a call the user denied comes back as a `tool_result` with `is_error: true`.
- **codex-cli 0.162.1, commands:** `item/commandExecution/requestApproval` with `itemId`, `command` and `availableDecisions`. The answer is `{"decision": "accept" | "acceptForSession" | "decline" | "cancel"}`. Some requests do not offer `decline`. File changes use `item/fileChange/requestApproval` with the same decisions.
- **codex-cli 0.162.1, MCP tools:** `mcpServer/elicitation/request` with `_meta.codex_approval_kind: "mcp_tool_call"` and `_meta.tool_params`. The tool name is only in the message text. The answer is `{"action": "accept", "content": {}}` or `{"action": "decline"}`.

## Acceptance Criteria

- [ ] A Claude `merge` call in a chat shows an approval card. The turn waits for the decision.
- [ ] "Allow" runs the call. The next identical call asks again.
- [ ] "Allow for this chat" runs the call. Later matching calls in the same chat do not ask, also in later turns.
- [ ] "Deny" returns a denial to the agent. The turn continues.
- [ ] A denied call shows "Merge upe-1 · Denied" in the work list. It does not count as a failed call.
- [ ] A failed call reads "✕ Merge upe-1 · failed: <first line of the error>" on one row. The full error opens in a card on hover, focus or click, and no second (native) tooltip appears.
- [ ] A turn's work list starts collapsed, or expanded when the turn has a failed call. The user's toggle wins. Collapsed, the status line shows "· k failed".
- [ ] In the Agent view, the thread and the reply box share the same right edge.
- [ ] A Codex approval request (command, file change or MCP tool) shows the same card and accepts the same decisions.
- [ ] Cancelling a turn denies all its open requests. The cards show "Cancelled".
- [ ] After a page reload, the transcript shows each card with its decision.
- [ ] A card that is still open after a reload can still be answered.
- [ ] A card whose turn ended without an answer shows "Expired".
- [ ] A pending card is visible without expanding the work list.
- [ ] The chat history list marks a chat while a card waits for an answer.
- [ ] Tool calls on the allowlist run without a card, as today.
- [ ] "Allow for this chat" never writes to the user's Claude Code settings files.
- [ ] An opt-in test against the real `claude` CLI pins the `control_request` and `control_response` JSON.
- [ ] `make test` passes.

## Flow

### 1. Shared types

1. Add `internal/chat/approval.go` with `ApprovalRequest`, the decisions `allow`, `allow_session`, `deny`, `cancelled`, and `Approver`.
2. `ApprovalRequest` holds the tool name, a short detail, the full input as JSON, the tool use ID, and a grant rule.
3. Add `Approve Approver` and `Grants []string` to `TurnInput`. A nil `Approve` denies every request.
4. Add `exponential.ToolDetail(name, input)`. The stream parser and the approval card use the same short detail.

### 2. Claude driver

1. Launch `claude` with `--input-format stream-json` and `--permission-prompt-tool stdio`.
2. Write the user prompt to stdin as a stream-json user message. Keep stdin open.
3. Answer each `control_request` with subtype `can_use_tool` in its own goroutine, through `in.Approve`.
4. Write a `control_response` with `allow` and `updatedInput`, or `deny` and a message.
5. For `allow_session`, add `updatedPermissions`: the suggestions with `destination` set to `session`.
6. Answer other `control_request` subtypes with an error response.
7. Close stdin after the `result` line.
8. Add every grant to `--allowedTools`, so granted calls do not ask in later turns.
9. Add `--settings {"permissions":{"ask":["mcp__xpo__merge"]}}`, so `merge` asks in a chat.

The grant rule is the first suggested rule as `Tool(content)`, or the tool name when there is none.

### 3. Codex driver

1. Forward `item/commandExecution/requestApproval` and `item/fileChange/requestApproval` through `in.Approve`.
2. Map `allow` to `accept`, and `allow_session` to `acceptForSession` when offered.
3. Map `deny` to `decline`, or to `cancel` when Codex does not offer `decline`.
4. Forward `mcpServer/elicitation/request` with `codex_approval_kind: "mcp_tool_call"`. Map `allow` and `allow_session` to `accept`, `deny` to `decline`.
5. Read the MCP tool name from the message text. When the text has another form, name the server only.
6. Answer each request in its own goroutine, so the turn keeps reading.
7. Keep the error answer for every other server request.

### 4. Runner and API

1. Keep open requests per chat in the runner, keyed by a runner-made approval ID.
2. Before a card, allow the request at once when its grant rule is in the chat's grants.
3. Settle `approval_requested` with the approval ID, tool name, detail, input and tool use ID.
4. Add `Runner.Decide(chatID, approvalID, decision)`. It delivers the decision to the waiting driver.
5. For `allow_session`, add the grant rule to `Meta.Grants`.
6. Settle `approval_resolved` with the decision: `allow`, `allow_session`, `deny` or `cancelled`.
7. On cancel, resolve every open request as `cancelled`. The driver answers it as a denial.
8. Settle the tool call of a denied or cancelled approval with status `denied`, not `error`.
9. Add `POST /api/local/chats/{id}/approvals/{approval_id}` with `{decision}`. Return 404 for an unknown or answered request.
10. Add `waiting: true` to a chat in the chat list while it has an open request.

### 5. UI

1. Fold `approval_requested` and `approval_resolved` into the tool step with the same tool ID. Without a tool ID, add an approval step.
2. Show pending approvals as cards below the work list, so they are visible when the work list is collapsed.
3. The card uses the composer's container: `radius-lg`, `border-default`, `surface-2`.
4. The headline says what will happen as an imperative: "Merge upe-1", "Run a shell command", "Edit main.go". Unknown tools read "Use <tool>".
5. The headline ends with the tool name in a muted tone: "Merge upe-1 via `xpo merge`".
6. "Deny", "Allow for this chat" and "Allow" sit on the headline row, right-aligned, in that order, at the composer's control size (`h-7`, `text-xs`). They wrap below a long headline.
7. Below the headline, the card shows only what the headline does not say. For a shell call, it shows the command in a mono block. For other tools, it lists the other arguments as `key value` rows, clamped to three lines with the full value as a tooltip. Without such arguments, the card is one row.
8. While a card waits, the turn's status line shows an amber dot and "Waiting for your approval", without a spinner or a timer.
9. After a decision, the work list row shows the outcome as a badge.
10. A call that did not run (denied, cancelled, expired or failed) reads as what was asked ("Merge upe-1"), not in the past tense. One verb table holds both forms for every known tool; the card headline uses the same table.
11. A call the user denied is not a failure.
12. A failed call's row ends with its reason in red: "failed: <first non-blank line>", or "denied: …" when the agent's own rules blocked it. A trailing "…" marks a message with more lines. The reason is cut at the row's edge with a fade, not an ellipsis, so Safari does not add its own tooltip. The tool detail keeps at least part of its width.
13. The reason opens a card with the full message on hover, on keyboard focus, and on click (for touch). The card uses the popover surface and scrolls long output.
14. A turn's work list is collapsed by default and expanded when the turn has a failed call, also when the failure arrives mid-turn. Once the user toggles it, their choice wins. The toggle carries `aria-expanded`.
15. While the work list is collapsed and the turn has failed calls, the status line shows "· k failed" in red. Expanded, the rows say it, so the count is hidden.
16. A finished turn marks open approvals as expired. A 404 from the API also shows "Expired".
17. Show a "Waiting for your approval" marker in the chat lists (recent chats, Agent view, status-bar pills) for chats with `waiting`.
18. In the Agent view, the thread column is the composer box's width (`max-w-[46rem]`: the composer's `max-w-3xl` minus its `px-4`), so bubbles, answers, work rows and the reply box share both edges.

## Decisions

- **Speak the CLI protocol directly, not through the Agent SDK.** xpo is Go and the SDK is TypeScript. The protocol is small: one request type and one response type.
- **One process per turn stays.** The process lives until the turn's `result` line. Resume between turns works as today.
- **xpo keeps "Allow for this chat" grants in the chat Meta.** Claude Code and Codex forget session permissions when the turn's process exits. The runner passes the grants to each turn.
- **Rewrite suggestion destinations to `session`.** The CLI suggests `localSettings`, which would write `.claude/settings.local.json` in the user's project.
- **`merge` asks in a chat in this issue.** It gives the cards a real first use, and it is safer than today's silent `merge`. xpo-00f0e5 turns it into a deny in Chat Mode.
- **Codex per-tool approval config moves to xpo-00f0e5.** Which tools ask is a mode decision. This issue forwards whatever Codex asks.
- **The real-CLI test is opt-in** (`XPO_TEST_REAL_CLI=1`). It calls a model, so `make test` skips it.
- **A user denial is not a failure.** The work list shows "Denied" next to the call. The red failure style stays for errors and for denials by the agent's own rules.
- **The card headline describes the effect, not the tool.** The user decides about "Merge upe-1"; the tool name is secondary detail.
- **No raw JSON on the card.** The input mostly repeats the headline. Only the parts that add information appear: the shell command, and the arguments the headline does not show.
- **Past tense only for calls that ran.** A failed or denied call had no effect, so "Merged upe-1" would be false.
- **The failure reason lives on the row, not in a box below it.** One line says what went wrong without expanding anything; the full message is one hover away.
- **A failure opens the work list instead of a separate failed list.** One list with one toggle; the failure shows in its context. A long turn with one failure opens fully, which is when the context matters.

## Edge Cases

**MEDIUM**
- The CLI changes the `control_request` shape. The real-CLI test fails and shows the change.
- A decision arrives after the turn ended. The API returns 404 and the card shows "Expired".
- The board restarts while a card waits. The turn is interrupted, and the card shows "Expired".

**LOW**
- Two cards are open at once (parallel tool calls). Each card is answered on its own.
- The browser tab is closed. The request waits until the user returns or cancels the turn.
- A Codex MCP approval has no item ID. Its card is a step of its own.
- A failed call has no message. The row reads "failed" without a card.

## Out of scope

- Which tools ask in which mode, and Codex per-tool approval config: xpo-00f0e5.
- A "Supervised / Auto-accept / Full access" permission picker in the composer.
- The chat popup's 4px offset between thread (`px-4`) and composer (`px-3`).
