# Walkthrough: error indicator on chat chips when the last turn failed

## What was built
A chat whose last turn failed shows a red `CircleAlert` icon on its status-bar chip. The status-bar history menu, the Agent view switcher and the Agent view home list show the same icon. The tooltip says "Last turn failed: <message>". The icon stays until the next turn starts.

Before this change, a failed turn wrote an `EventError` to the event log and set the chat to `idle`. The chips could not tell it from a finished turn.

## How the pieces fit together

### Backend: `internal/chat`
- `Meta.LastError` (`last_error`, `event.go`) holds the error message of the last turn when it failed.
- `endTurn` (`runner.go`) keeps the message of the two failure branches in `failure`. The branches are `res == nil` (start, send or process exit errors) and `res.IsError` (an error result from Claude or Codex). The meta update writes `m.LastError = failure`. Finished and stopped turns therefore write an empty string.
- `Send` (`runner.go`) and agent-started turns (`sessionHandler.AgentTurn`, `session.go`) clear `LastError` when they set `StateRunning`.
- The Work Mode hand-over in `endTurn` also sets `StateRunning`. It needs no change: it only runs after a finished turn, so `failure` is already empty.
- `handleListChats` returns `Meta` as stored, so the field reaches the frontend without a handler change.

### Frontend
- `ChatMeta.last_error` (`web/src/api/chat.ts`).
- `chatIndicator` (`agent-state-utils.ts`) returns `failed` after `interrupted` and before `unseen`. The seen records do not change it.
- `chatStateIcon` (`recent-chats.tsx`) renders `failed` as a lucide `CircleAlert` in `--color-error`. The icon fills the `size` box, as the spinner does. The label cuts the message at `ERROR_LABEL_MAX` (200) characters and adds "…".

## Decisions
- **A message, not a bool.** The tooltip says why the turn failed.
- **Clear at turn start.** While a turn runs, the spinner shows anyway. Interrupted and failed therefore cannot both apply.
- **Viewing does not clear it.** The icon says the chat is broken, not that a reply is new. The user chose this.
- **Failed beats unseen.** A failure is more urgent than a new reply.
- **CircleAlert for failed, dot for interrupted.** The user picked the exclamation circle. Both use the error color, so the shape tells them apart.

## Limits
- When an agent exits while no turn is open, no turn ends. `LastError` does not change.
- Chats saved before this change have no `last_error` and show no icon.

## Acceptance criteria
- [x] After a failed turn, the chat's meta has `last_error` set to the error message. Evidence: `TestRunner_ErrorResult` (error result) and `TestRunner_ProcessFailsWithoutResult` (process exit).
- [x] A turn that finishes or stops clears `last_error`. Evidence: `TestRunner_FinishedTurnClearsLastError`; `TestRunner_TurnStartClearsLastError` checks it after the stop; `TestRunner_Cancel`.
- [x] A new turn clears `last_error` when it starts. Evidence: `TestRunner_TurnStartClearsLastError` reads the meta while the second turn runs.
- [x] A chip of a chat with `last_error` shows a `--color-error` `CircleAlert` with the title "Last turn failed: <message>". Evidence: `recent-chats.test.tsx` "shows an error circle with the message for a failed turn" and "cuts a long error message in the label".
- [x] An interrupted chat keeps its filled red dot. Evidence: `recent-chats.test.tsx` "shows an error dot for an interrupted chat" checks that no lucide icon renders.
- [x] The status-bar history menu, the Agent view switcher and the Agent view home list show the same indicators. Evidence: all of them render through `chatStateIcon`.
- [x] Opening the chat does not clear the error icon. Evidence: `agent-state-utils.test.ts` "keeps a failed chat failed once seen".
- [x] `make test` passes. Evidence: exit 0; 71 test files and 993 frontend tests, plus the Go tests.
