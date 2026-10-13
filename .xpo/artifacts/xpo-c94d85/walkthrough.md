## What changed

The context chip now shows the context the next message carries. It is no longer a filter over the issue you're viewing.

- `agent-state-utils.ts`: new pure helpers.
  - `ViewedIssue` / `nextViewed` record the last issue the user opened, with a `seq` that counts each time they open a different one. Views without an issue keep the last one.
  - `pendingContextIssue(viewed, consumed, workIssue)` gives the pending issue: the viewed one, if its seq is newer than what the chat consumed, unless it is the chat's Work Mode issue.
  - `ConsumedContext`, `consumedFor`, `consumeContext`, `adoptNewChatContext` store, per chat, the seq up to which the chat sent or removed its context. A new chat stores it under `""`, and `chatCreated` moves it to the chat's ID.
- `AgentChatProvider.tsx` / `agent-context.ts`: the provider tracks `viewed`, adjusting state during render when `context.issue_id` changes, and stores `consumed`. It exposes `pendingIssueFor(chatId)` and `consumeContext(chatId)`. The state outlives the popup, but not a page reload.
- `ChatPopup.tsx`: the chip and the `<xpo-context>` issue both come from `pendingIssueFor`. Sending, or removing the chip, calls `consumeContext`. The title comes from the issue list, so the chip keeps its title after you switch to a view without an issue, such as Agent.

## Acceptance Criteria

- [x] A removed chip stays removed after you close and reopen the popup on the same issue. The consumed record lives in the provider. Unit test: "clears it once sent or removed, until another issue is opened".
- [x] After a message is sent, the chip is gone, and the next turn carries no issue context. `sendText` consumes before sending; same unit test.
- [x] Opening a different issue shows the chip again, including going back to an earlier issue. Same unit test (`back`).
- [x] In Work Mode, the chip never shows the bound issue. Unit test: "never adds the issue a Work Mode chat is bound to".

## Verification

`make test` passes (exit 0): lint, 824 frontend tests, Go tests. `tsc -b` is clean. Manual testing in the UI is still to do.
