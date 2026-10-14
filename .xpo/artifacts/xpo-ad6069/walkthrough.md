# Walkthrough: after an approved start, the chat continues in Work Mode

## What was built

The agent can call `start` from Chat Mode, and the user can allow it. The board then starts the next turn in Work Mode by itself. Before this change, the chat stopped after the `start` turn. The user had to send "go" before any work happened.

## Why it stopped

xpo-00f0e5 changes the mode only between turns, so each turn runs under one tool set. `modeTools.apply` binds the chat in the run's deferred function, after the turn ends. `prompt.md` told the agent to end its turn after `start`. Nothing started the next turn.

## How the pieces fit

### 1. The runner hands the turn over (`internal/chat/runner.go`)

The run's deferred function already calls `modes.apply` under `r.mu`. It now also checks four conditions:

- `apply` returned a `mode_changed` event with `Mode == ModeWork`. Only an agent `start` that bound the chat gives this result.
- The turn finished (`resolved != nil`), so it was not an error.
- The user did not stop the turn (`t.ctx.Err() == nil`).
- `Shutdown` did not begin (`!r.closing`).

When all four are true, `handOverLocked` registers a new `turn` for the chat in place of the old one, and calls `r.wg.Add(1)`. Meta stays `running`. After it unlocks, the function settles `mode_changed`, then `turn_continued`. Then it calls `go r.run(...)` with the Work profile and `continuePrompt`.

The hand-over happens while `r.mu` is held, so the chat is never unregistered between the two turns. A user message in that gap gets `ErrBusy`. It does not start a second turn.

`finish` now takes the turn: `finish(id, t)`. It always cancels `t` and closes `t.done`. It deletes the chat's entry only if `r.turns[id] == t`. After a hand-over, the entry belongs to the next turn, so `finish` leaves it.

`Shutdown` sets `r.closing` before it cancels turns. Without this flag, a hand-over could start a turn after `Shutdown` took its list of turns. Nothing would cancel that turn, and `Wait` would block until the agent finished.

### 2. The continued turn's prompt

`continuePrompt` returns "Work Mode is on for <issue>. Continue with what the user asked for." `buildPrompt` wraps it in the usual `<xpo-context>` block, with `mode: work on <issue>` and the `worktree` line. The turn resumes the same session, so the agent still has the user's request.

### 3. A new event: `turn_continued` (`internal/chat/event.go`)

`turn_continued` is a settled event that carries `issue_id`. It opens the continued turn in the transcript. It is not a `user_message`, because the user did not write anything.

### 4. Prompt (`internal/chat/prompt.md`)

The Chat Mode line now says: after `start` succeeds, end your turn without a summary. The board starts the next turn in Work Mode, and the agent carries on there. This removes the "The next turn runs in Work Mode" message that ended each `start` turn.

### 5. UI (`web/src/components/Agent`)

- `chatThreadReducer` opens a running turn on `turn_continued` and sets `continued: true`. The turn keeps the `user.time` field, so the duration and the day label still work.
- `ChatThread` draws no user bubble for a continued turn. The transcript reads: answer, then the "Entered Work Mode on …" divider, then the agent's next answer.
- `useChatThread` treats `turn_continued` as a meta event, so the chat list shows the chat as running.

## Decisions

- **No user bubble.** The user chose this over a muted "Continuing in Work Mode" line. The divider already tells the user what happened.
- **No continuation after Start Work or Continue Work.** These buttons prefill a draft for the user to send. They do not run a turn, so the runner has nothing to continue.
- **Same provider, model and effort.** The continued turn uses the Meta of the `start` turn.

## Known gaps

- Nobody ran Codex end to end with this change. The change is in the runner, not in the drivers, so Codex takes the same path.

## Acceptance Criteria

- [x] After the user allows an agent `start` card, and the turn ends, a new turn starts in Work Mode with no user message. *`TestRunner_StartEntersWorkModeAfterTheTurn`: events are `turn_done`, `mode_changed`, `turn_continued`, then a second `turn_done`, with one `user_message`.*
- [x] That turn runs with the Work profile (worktree dir, Edit/Write/Bash) and resumes the same agent session. *Same test: `--tools` has Edit, Write and Bash, `--resume sess-7`, the working dir is the worktree, and the prompt has `mode: work on xpo-abc`.*
- [x] The transcript shows the "Entered Work Mode on …" divider, then the agent's answer, with no user bubble. A reload shows the same. *`ChatThread.test.tsx` "shows no user bubble for a turn the board continued"; reducer test in `chat-thread-utils.test.ts`. A reload replays the same settled events.*
- [x] No continuation after a denied `start`, a failed bind, a stopped or failed turn, or Start Work / Continue Work. *`TestRunner_FailedStartKeepsChatMode`, `TestRunner_StoppedStartTurnDoesNotContinue`, `TestRunner_MergeLeavesWorkMode`. `BindWork` runs no turn.*
- [x] A user message sent while the continuation runs gets the usual busy response; the composer shows the turn running. *`TestRunner_ContinuedTurnHoldsTheChat`: `Send` returns `ErrBusy` and `Running` is true. The reducer marks the turn running.*
- [x] `make test` passes. *Run before commit.*
