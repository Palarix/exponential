# Spec: error indicator on chat chips when the last turn failed

## What
A chat whose last turn ended in an error shows a red circle with an exclamation mark on its chip and in every recent-chats list. The server records the turn's error in the chat's metadata. The icon stays until the next turn starts.

## Why
Today `endTurn` (`internal/chat/runner.go`) writes an `EventError` to the event log and then sets the state to `idle`. A failed chat looks the same as a finished one. You only see the failure when you open the thread. xpo-f2d4c8 left this state out because `Meta` had no field for it.

A turn fails when:
- the agent cannot start, or `Send` fails (`res == nil` with an error),
- the agent process exits during the turn (`Exited`),
- the agent reports an error result (`res.IsError`: a Claude `is_error` result, a Codex `failed` turn, or a Codex start, resume or thread failure).

A stopped turn (`EventTurnStopped`) is not a failure.

## Acceptance Criteria
- [ ] After a failed turn, the chat's meta has `last_error` set to the error message.
- [ ] A turn that finishes or stops clears `last_error`.
- [ ] A new turn clears `last_error` when it starts.
- [ ] A chip of a chat with `last_error` shows a `--color-error` `CircleAlert` icon with the title "Last turn failed: <message>".
- [ ] An interrupted chat keeps its filled red dot.
- [ ] The status-bar history menu, the Agent view switcher and the Agent view home list show the same indicators.
- [ ] Opening the chat does not clear the error icon.
- [ ] `make test` passes.

## Flow
1. Add `LastError string \`json:"last_error,omitempty"\`` to `Meta` in `internal/chat/event.go`.
2. In `endTurn`, keep the error message of the `res == nil` and `res.IsError` branches in a local variable.
3. In the `endTurn` meta update, set `m.LastError` to that message. The message is empty for finished and stopped turns.
4. Clear `LastError` where a turn sets `StateRunning` (`runner.go:184`, `runner.go:556`, `session.go:248`).
5. Add Go tests in `runner_test.go` for the failed, finished, stopped and next-turn cases.
6. Add `last_error?: string` to `ChatMeta` in `web/src/api/chat.ts`.
7. Add `"failed"` to `ChatIndicator` in `agent-state-utils.ts`. Rank it after `interrupted` and before `unseen`.
8. In `chatStateIcon` (`recent-chats.tsx`), render `failed` as a lucide `CircleAlert` in `--color-error`, centered in the `size` box.
9. Add unit tests for `chatIndicator` and `chatStateIcon` first, then make them green.

## Decisions
- **A message, not a bool.** The icon's tooltip can then say why the turn failed. The field costs nothing when empty.
- **Clear at turn start, not only on success.** While a turn runs, the spinner shows anyway. Interrupted and failed then cannot both apply.
- **Not cleared by viewing.** The icon says the chat is broken, not that a reply is new. The user acts on it by sending again.
- **Failed beats unseen.** A failure is more urgent than a new reply.
- **CircleAlert for failed, dot for interrupted.** The user chose the exclamation circle. Both use the error color, so the shape tells them apart. Interrupted keeps its dot from xpo-f2d4c8.

## Edge Cases
- MEDIUM: an agent exits while idle (no open turn). No turn ends, so `LastError` does not change.
- LOW: chats saved before this change have no `last_error`. They show no error icon.
- LOW: long error messages. The tooltip cuts the message to 200 characters.

## Assumptions
- `endTurn` runs once per turn for every failure path listed in Why.
- The chat list endpoint returns `Meta` as stored, so the new field reaches the frontend without handler changes.
