# Walkthrough: approval cards for tool calls (Claude and Codex)

## What was built

A chat agent can now ask before a tool call runs. The board shows the question as an approval card in the transcript. The user picks "Deny", "Allow for this chat" or "Allow", and the turn continues. The first real use is `merge`: in a chat, it now asks before it runs.

The work list also got a polish pass along the way: failed calls read as what was asked, carry their reason on the row, and open the work list.

## How the pieces fit

```
claude / codex  ──request──►  driver  ──Approver──►  runner  ──event──►  SSE  ──►  ApprovalCard
      ▲                                                 ▲                              │
      └──────── response ◄──── driver ◄──── decision ───┴──── POST /approvals/{id} ◄───┘
```

### 1. The protocol (`internal/chat/approval.go`, `claude.go`, `codex.go`)

Both agents already had an approval protocol; xpo never answered it.

- **Claude Code** asks over stdio when it runs with `--permission-prompt-tool stdio`. It writes a `control_request` line (subtype `can_use_tool`) and waits for a `control_response` on stdin. This is what the Claude Agent SDK does under `canUseTool`; Elysium uses it that way. xpo is Go, so the driver speaks the protocol directly.
- To answer, stdin must stay open. The prompt now goes in as a stream-json user message, and the driver closes stdin after the `result` line. Each request is answered in its own goroutine, so parallel tool calls get their own cards. `lineWriter` serializes the writes.
- **Codex** sends JSON-RPC server requests: `item/commandExecution/requestApproval`, `item/fileChange/requestApproval`, and `mcpServer/elicitation/request` with `codex_approval_kind: "mcp_tool_call"`. The driver used to decline all of them. `codexApproval` now maps them to the same `ApprovalRequest`. Other server requests are still declined.
- Both drivers call one function type, `Approver`. A nil approver denies, so code without a runner keeps the old behavior.

The wire formats are in the spec. They were measured against Claude Code 2.1.296 and codex-cli 0.162.1.

### 2. Decisions and grants (`runner.go`)

- `Runner.approver` opens a card: it stores a channel under a runner-made approval ID and settles `approval_requested`. It then waits for `Decide` or for the turn to end. A turn that ends first resolves its cards as `cancelled`.
- **"Allow for this chat" is stored by xpo** in `Meta.Grants`. Both CLIs forget session permissions when the turn's process exits, and xpo starts one process per turn. The runner passes the grants to every turn: Claude Code gets them as extra `--allowedTools` values; a Codex request that matches a grant is allowed without a card.
- **Claude Code suggests rules with `destination: "localSettings"`.** Sent back unchanged, they would write to the user's `.claude/settings.local.json`. `claudeSessionPermissions` rewrites every destination to `session`.
- **Denied calls settle as `denied`.** Claude Code reports a user denial as an ordinary error result. The runner remembers which tool IDs the user denied and corrects the status.
- **`merge` asks in a chat** through `--settings {"permissions":{"ask":["mcp__xpo__merge"]}}`. An `ask` rule beats the `mcp__xpo__*` allow wildcard (measured). xpo-00f0e5 turns this into a hard deny in Chat Mode.

### 3. API (`internal/server/chat.go`)

- `POST /api/local/chats/{id}/approvals/{approval_id}` with `{decision}`. Unknown or answered cards return 404; a bad decision returns 400.
- The chat list adds `waiting: true` for chats with an open card. `Meta.Waiting` is live state: `writeMeta` never stores it.

### 4. UI (`web/src/components/Agent/`)

- **Reducer** (`chat-thread-utils.ts`): approval events fold into the tool step with the same tool ID, as `step.approval`. A settled `tool_call` keeps the approval. A finished or interrupted turn marks open approvals `expired`; a 404 from the API does the same through the `approval_expired` action.
- **`ApprovalCard`**: one row, "Merge upe-1 via `xpo merge`", with the buttons on the right. It shows only what the headline does not say: a shell command in a mono block, or the other arguments as `key value` rows. Pending cards render below the work list, so a collapsed list never hides them.
- **One verb table** (`TOOL_VERBS`) holds the past tense for calls that ran ("Merged") and the imperative for calls that did not ("Merge"). The card headline and the work list rows use it, so they cannot drift apart.
- **Failed rows** end with "failed: <first line>" in red. The full message opens in a Radix tooltip card on hover, focus or click. The reason fades out at the edge instead of ending in an ellipsis: Safari shows its own tooltip for ellipsis-truncated text, which stacked on the card.
- **Work list**: collapsed by default, expanded when the turn has a failed call, also mid-turn. The user's toggle wins (`toggled ?? hasFailures`). Collapsed, the status line shows "· k failed". A call the user denied is not a failure.
- **Chat lists** (recent chats, Agent view, status-bar pills) share `chatStateIcon`: an amber shield for "waiting", a spinner for "running".
- **Alignment**: the Agent view's thread column is now `max-w-[46rem]`, the composer box's width, so bubbles, answers, rows and the reply box share both edges.

## Decisions worth knowing

- **One process per turn stays.** Approvals did not need a long-lived process; stdin just stays open until the result.
- **The Codex MCP tool name comes from the message text** ("Allow the xpo MCP server to run tool "merge"?"). No other field carries it. If Codex rewords it, the card names the server only.
- **Codex sometimes offers no `decline`.** A denial then answers `cancel`, which ends that Codex turn.
- **Which tools ask in which mode is xpo-00f0e5's job**, including Codex per-tool approval config.

## Acceptance Criteria

- [x] A Claude `merge` call in a chat shows an approval card. The turn waits for the decision. — `TestRunner_ApprovalAllowOnce`, `TestChat_Approval`; seen end to end on a board with the real CLI.
- [x] "Allow" runs the call. The next identical call asks again. — `TestRunner_ApprovalAllowOnce` (no grant stored); real-CLI test `allow`.
- [x] "Allow for this chat" runs the call. Later matching calls in the same chat do not ask, also in later turns. — `TestRunner_ApprovalAllowSessionGrantsLaterTurns`; real-CLI test `grant` (the second turn asked nothing).
- [x] "Deny" returns a denial to the agent. The turn continues. — `TestRunner_ApprovalDeny`; real-CLI test `deny`; end to end, Claude replied without merging.
- [x] A denied call shows "Merge upe-1 · Denied" in the work list. It does not count as a failed call. — `TestRunner_DeniedApprovalMarksToolCallDenied`, `stepSummary` and `failedSteps` tests; screenshot.
- [x] A failed call reads "✕ Merge upe-1 · failed: <first line of the error>" on one row. The full error opens in a card on hover, focus or click, and no second (native) tooltip appears. — `failureReason` and `stepSummary` tests; screenshots of row and hover card; the user confirmed in Safari.
- [x] A turn's work list starts collapsed, or expanded when the turn has a failed call. The user's toggle wins. Collapsed, the status line shows "· k failed". — `ChatThread.test.tsx`, `collapsedFailureNote` test; screenshots of both states.
- [x] In the Agent view, the thread and the reply box share the same right edge. — Measured in the browser: 1188px at 1400px wide, 675px at 700px.
- [x] A Codex approval request (command, file change or MCP tool) shows the same card and accepts the same decisions. — `TestCodexDriver_ForwardsApprovals`, `TestCodexApprovalDecisions`, with request shapes recorded from codex-cli 0.162.1. Not seen in the UI: Codex chats send no approvals until xpo-00f0e5 configures them.
- [x] Cancelling a turn denies all its open requests. The cards show "Cancelled". — `TestRunner_ApprovalCancelResolvesOpenRequests`; reducer test "keeps a cancelled decision".
- [x] After a page reload, the transcript shows each card with its decision. — Reducer test "rebuilds a settled request on reload"; approval events are settled and stored.
- [x] A card that is still open after a reload can still be answered. — Open cards live in the runner, not the page; `TestChat_Approval` answers over HTTP.
- [x] A card whose turn ended without an answer shows "Expired". — Reducer tests "expires open requests…" and "marks an approval expired on request".
- [x] A pending card is visible without expanding the work list. — Cards render outside the list; screenshot of a pending card under a collapsed list.
- [x] The chat history list marks a chat while a card waits for an answer. — `TestChat_Approval` checks `waiting`; `chatStateIcon` in all three lists.
- [x] Tool calls on the allowlist run without a card, as today. — `TestRunner_TurnPersistsSettledEvents` (an allowed `list` call, no card); end to end, `show` ran without a card.
- [x] "Allow for this chat" never writes to the user's Claude Code settings files. — `TestClaudePermissionResponse` asserts every destination is `session`.
- [x] An opt-in test against the real `claude` CLI pins the `control_request` and `control_response` JSON. — `TestRealClaude_Approval` (`XPO_TEST_REAL_CLI=1`), passing against Claude Code 2.1.296.
- [x] `make test` passes. — Go and 766+ frontend tests, plus `go test -race ./internal/chat`.

## Testing notes

- The fake `claude` scripts now read one stdin line (the prompt) instead of reading to EOF, because the driver keeps stdin open.
- The Codex fixture's old approval request became `item/permissions/requestApproval`, which still tests the "decline what we do not handle" path.
- Run `XPO_TEST_REAL_CLI=1 go test ./internal/chat -run TestRealClaude_Approval` after a Claude Code upgrade. It calls Haiku three times and takes about 20 seconds.
