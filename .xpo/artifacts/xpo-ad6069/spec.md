# Agent chat: after an approved start, the chat continues in Work Mode

## What

After the agent's `start` call is approved and binds the chat to the issue, the board runs the next turn in Work Mode by itself. The user doesn't have to send "go".

## Why

xpo-00f0e5 changes the mode only between turns, so that each turn runs under one tool set. The agent ends its turn after `start`, and nothing starts the next turn. Each time, the user has to send a filler message.

## How

**Runner (`internal/chat/runner.go`, `mode.go`)**
- `modeTools.apply` already reports when an agent `start` bound the chat. When it did, and the turn finished cleanly (`turn_done`, not stopped and not an error), the run's deferred function starts a continuation turn after it settles the `mode_changed` event.
- The continuation goes through the same path as `Send`: it registers the turn, re-reads Meta, and builds the profile, which is now the Work profile. The provider, model and effort stay as they were on the chat.
- It does not continue after a failed bind ("Could not enter Work Mode"), after a merge, or after a user-initiated `BindWork` (Start Work / Continue Work already prefill a draft for the user to send).
- The continuation prompt is fixed text inside the usual `<xpo-context>` block (`mode: work on <issue>`, `worktree: …`), for example: "Work Mode is on for <issue>. Continue with what the user asked for."

**Event and UI**
- A new settled event `turn_continued` opens the continuation turn. It carries `issue_id`. It is not a `user_message`: the thread shows no user bubble.
- `chatThreadReducer` opens a new turn on `turn_continued`, with no user text. `ChatThread` renders such a turn without a user bubble, so the transcript reads: answer → "Entered Work Mode on …" divider → the agent's next answer.
- Replaying a stored chat gives the same result.

**Prompt (`prompt.md`)**
- Chat Mode line: "After `start` succeeds, end your turn without a summary: the board starts the next turn in Work Mode on its own, and you carry on there."

## Edge cases

- The user presses Stop during the `start` turn: no continuation.
- The user's own message can't race the continuation: the continuation registers the turn before the deferred function releases it, so `Send` gets `ErrBusy` and the composer already shows the turn running. (Implementation: `handOverLocked` registers the next turn while `r.mu` is held; `finish(id, t)` only unregisters the chat if `t` is still its turn.)
- Board shutdown during the hand-over: the continuation is not started (`Runner.closing`, set by `Shutdown`).
- Codex and Claude both continue: the change is in the runner, not the drivers.

## Acceptance Criteria

- [ ] After the user allows an agent `start` card, and the turn ends, a new turn starts in Work Mode with no user message.
- [ ] That turn runs with the Work profile (worktree dir, Edit/Write/Bash) and resumes the same agent session.
- [ ] The transcript shows the "Entered Work Mode on …" divider, then the agent's answer, with no user bubble. A reload shows the same.
- [ ] No continuation after a denied `start`, a failed bind, a stopped or failed turn, or Start Work / Continue Work.
- [ ] A user message sent while the continuation runs gets the usual busy response; the composer shows the turn running.
- [ ] `make test` passes.
