# Walkthrough: AskUserQuestion with an in-chat question card

## What was built

Claude chat turns can call `AskUserQuestion` in Chat Mode and Work Mode. The call opens a question card in the chat. The user picks options or types an answer under "Other…", and the answers go back to the agent as the tool result. Skip denies the call. The thread keeps each question with its answer.

## How the call reaches us

A probe of claude 2.1.300 settled the open question in the issue. Under `--permission-prompt-tool stdio`, `AskUserQuestion` arrives as an ordinary `can_use_tool` control_request (with `requires_user_interaction: true`). There is no special protocol. The answer goes back as an allow whose `updatedInput` is the original input plus `answers`, keyed by question text:

```json
{"behavior":"allow","updatedInput":{"questions":[…],"answers":{"Which color?":"Red"}}}
```

Claude Code then gives the model `The user answered: "Which color?"="Red". …`. `TestRealClaude_Question` (run with `XPO_TEST_REAL_CLI=1`) pins this protocol.

The design follows from this. A question rides the existing approval pipeline. Cards, the chat's waiting state, Stop and expiry after a restart already work for it. Only the card body and the reply differ.

## The pieces

### Profile (`internal/chat/profile.go`)

`AskUserQuestion` joins `claudeSharedTools`, which `claudeLookupTools` was renamed to because it no longer holds only lookups. It is on `--tools` and **never** on `--allowedTools`. If it were allowed, Claude Code would run the tool without asking xpo, and nobody could answer. `TestProfile_AskUserQuestionNeverAllowed` guards this.

### Verdict and the runner (`approval.go`, `runner.go`)

`Approver` now returns `Verdict{Decision, Answers}` rather than a bare string, so the answers can travel from the runner back to the Claude driver. Codex reads only `.Decision`.

The runner's open cards are `openCard{reply chan Verdict, question bool}`. There are two entry points:

- `Decide(chat, id, decision)`: unchanged for approval cards. On a question card it accepts only `deny`, which is Skip.
- `Answer(chat, id, answers)`: question cards only. It needs a non-empty map with no blank answers.

`reply` checks that the card takes the verdict: `deny` always, `answer` only on a question, allows only on an approval. A question also skips the chat-grant shortcut. "Allow for this chat" is never offered for it, and a grant must never answer it.

The resolved event (`approval_resolved`) carries `answers`. The transcript rebuilds from it on reload.

### Claude response (`claudePermissionResponse`)

- `answer`: `allow`, with the decoded input plus `answers`.
- `deny` on `AskUserQuestion`: "The user skipped the questions without answering." The agent can then go on or ask in prose.
- `cancelled` keeps "The user cancelled the turn."

### Step detail (`internal/exponential/claudestream.go`)

The generic detail keys include `description`. While the input streams in, the partial scan found an option's description ("Warm") and showed it as the step's detail. `AskUserQuestion` now uses its first question's text, both from the partial buffer and from the final input.

### Server

`POST /api/local/chats/{id}/approvals/{approval}` sends `{"decision":"answer","answers":{…}}` to `Runner.Answer`. Every other decision still goes to `Decide`.

### Frontend

- **`chat-thread-utils.ts`**: `parseQuestions` reads the input defensively and drops questions without text. The selection state is pure functions over `QuestionPick{picked, other}`: `pickOption`, `toggleOther`, `setOtherText` and `questionAnswers`. The board has no DOM testing library, so this keeps the card's logic unit-testable. `other === null` means the field is closed. A question without options starts with it open. `questionAnswers` builds Claude Code's own format: picked labels in option order, then the free text, joined with ", ". It returns null until every question has an answer.
- **`QuestionCard.tsx`**: it uses `role="radiogroup"` with radio rows for single-select questions, and checkbox rows for multi-select ones. "Other…" is one more row. On a single-select question it replaces the pick, and on a multi-select one it adds to the picks. The card hides an option's description when it repeats the label. Enter in the field sends. The footer has Skip, then Send (primary, last), as on `ApprovalCard`.
- **`ChatThread.tsx`**: an *answered* question becomes its own turn segment (`kind: "question"`) in place, rendered as `AnsweredQuestion`: each question in muted text with the answer under it, behind a left rule. It shows "Thinking" while it is the last segment, because the agent is working on the answer. Open and skipped questions stay rows in their tool group, so the active line can say "Waiting for your answer" and the row can say "Skipped".

## Acceptance criteria

- [x] `--tools` includes `AskUserQuestion` in both modes. Evidence: `chatTools`/`workTools` in `profile_test.go`, and `TestProfile_AskUserQuestionNeverAllowed`.
- [x] `--allowedTools` never includes it, and a grant can never add it. Evidence: `TestProfile_AskUserQuestionNeverAllowed`. `TestRunner_QuestionAnswer` shows `allow`/`allow_session` rejected and no grants added. The approver skips `granted()` for questions.
- [x] A question card, not an approval card, with header, text, options with descriptions, and "Other…". Evidence: `QuestionCard.test.tsx`, and `ChatThread.test.tsx` "shows a question card…".
- [x] The free-text field appears only after "Other…", and at once without options. Evidence: `QuestionCard.test.tsx`, and "opens the free-text field only on Other…" and "opens the field at once…" in the utils tests.
- [x] Single-select takes one option, multi-select several. Evidence: "takes one option…" and "toggles options…" tests, and the radio/checkbox roles in `QuestionCard.test.tsx`.
- [x] Free text replaces the option on a single-select question and goes after the options on a multi-select one. Evidence: the utils tests ("Teal"; "Cat, a parrot").
- [x] Send stays disabled until every question has an answer, and returns `allow` with `updatedInput.answers`. Evidence: "needs an answer to every question", the disabled Send in `QuestionCard.test.tsx`, `TestClaudePermissionResponse_Answer`, and `TestRealClaude_Question` against claude 2.1.300.
- [x] Skip denies with the skip message. Evidence: `TestClaudePermissionResponse_SkipQuestion` and `TestRunner_QuestionSkip`.
- [x] Stop resolves an open card as cancelled. Evidence: the shared approver path (`TestRunner_ApprovalCancelResolvesOpenRequests`) and the cancel case in `TestClaudePermissionResponse_SkipQuestion`.
- [x] The transcript shows each question with its answer, and so does a reload. Evidence: `approval_resolved.answers` in `TestRunner_QuestionAnswer`/`TestChat_QuestionAnswer`, and `ChatThread.test.tsx` "shows each question with its answer, in place", which is built from settled events.
- [x] The thread shows "Waiting for your answer", and the chat lists mark the chat as waiting. Evidence: `ChatThread.test.tsx` and `waitingChats` in `TestChat_QuestionAnswer`.
- [x] The system prompt mentions AskUserQuestion. Evidence: `args_test.go`.
- [x] `make test` passes. The user tried it in the board and approved it.

## Notes for later

- Codex has `item/tool/requestUserInput` (experimental). xpo-eafa9b tracks mapping it onto this card. The card already works from parsed questions, so most of the work is in `codex.go`.
- Answers are keyed by question text, which is Claude Code's format. Two identical questions in one call would collide. We accept that.
