# Spec: AskUserQuestion with an in-chat question card

## What

Claude chat turns get the `AskUserQuestion` tool in both modes. When the agent calls it, the chat shows a question card: each question with its options as buttons, plus an "Other…" button that opens a free-text field. The user's answers go back to the agent as the tool result. The user can also skip the card, which denies the call.

## Why

The agent cannot ask the user a structured question today. It writes the question as prose and ends its turn, and the user types an answer. A card is quicker to answer, and it keeps the agent's turn going, so the agent continues with its plan.

## Verified against claude 2.1.300 (2026-10-14)

- A probe ran `claude -p --permission-prompt-tool stdio --tools AskUserQuestion`. The call arrives as a normal `can_use_tool` control_request with `tool_name: "AskUserQuestion"` and `requires_user_interaction: true`.
- Input: `{questions: [{question, header, options: [{label, description}], multiSelect}]}`.
- The answer goes back as an allow: `{behavior: "allow", updatedInput: {...input, answers: {"<question text>": "<answer>"}}}`. A multi-select answer is the labels joined with ", ". Free text is passed through as the answer.
- The model then sees: `The user answered: "Q1"="A1", "Q2"="A2". Read the answers carefully …`.
- The tool must not be on `--allowedTools`. If it is, Claude Code runs it without asking us, and there are no answers to return.

## Codex

Codex app-server 0.162.1 has `item/tool/requestUserInput`, but its schema marks it EXPERIMENTAL. This issue is Claude-only. Follow-up: xpo-eafa9b.

## Acceptance Criteria

- [ ] `--tools` includes `AskUserQuestion` in Chat Mode and in Work Mode.
- [ ] `--allowedTools` never includes `AskUserQuestion`. A chat grant can never add it.
- [ ] An `AskUserQuestion` call opens a question card, not an approval card. The card shows each question's header, text, options (label and description), and an "Other…" button.
- [ ] The free-text field shows only after the user clicks "Other…". A question without options shows the field at once.
- [ ] Single-select questions take one option. Multi-select questions take several.
- [ ] Free text replaces the option for a single-select question, and is added after the options for a multi-select question.
- [ ] "Send" is disabled until every question has an answer. It returns `allow` with `updatedInput.answers`, keyed by question text.
- [ ] "Skip" denies the call with the message "The user skipped the questions without answering."
- [ ] Stop resolves an open card as cancelled, the same as an approval card.
- [ ] After an answer, the transcript shows each question with its answer. A reload shows the same answers.
- [ ] While a question card is open, the turn shows "Waiting for your answer", not "Waiting for your approval". The chat lists mark the chat as waiting, as they do for approvals.
- [ ] The Claude system prompt tells the agent it can ask with AskUserQuestion.
- [ ] `make test` passes.

## Flow

### 1. Profile (`internal/chat/profile.go`)

1. Add `AskUserQuestion` to `claudeTools()` in both modes.
2. Do not add it to `claudeAllowed()`. The comment says why.

### 2. Answers through the approval pipeline (`internal/chat`)

1. Add `Verdict{Decision string; Answers map[string]string}`. `Approver` returns a `Verdict`, not a string. Codex reads only `Decision`.
2. Add `DecisionAnswer = "answer"` and `Runner.Answer(chatID, approvalID, answers)`. `Runner.Decide` keeps its string decision. `Answer` needs at least one answer, and no blank ones, or it returns `ErrBadDecision`. `answer` is only valid on a question card, and `allow` and `allow_session` are not valid on one. The runner remembers which open cards are questions.
3. The runner's approver skips the grant check for `AskUserQuestion`.
4. `claudePermissionResponse`: for `answer`, return `allow` with the input plus `answers`. For a denial of `AskUserQuestion`, use the skip message.
5. `EventApprovalResolved` gets an `answers` field, so the transcript and reload show the answers.
6. `ToolDetail` (`internal/exponential/claudestream.go`): the detail of an `AskUserQuestion` call is its first question's text. Without this, the partial-input scan would show an option's `description` while the input streams in.

### 3. Server (`internal/server/chat.go`)

1. `POST /api/local/chats/{id}/approvals/{approval}` accepts `{decision: "answer", answers: {...}}`.

### 4. Frontend (`web/src`)

1. `api/chat.ts`: add `"answer"` to the decision types, an `answers` field on events, and an `answerQuestion(chatId, approvalId, answers)` call.
2. `chat-thread-utils.ts`: store `answers` on the step approval. Add a parser for the AskUserQuestion input. Add a function that builds the answer string for a question from the picked labels and the free text. Add `TOOL_VERBS` for `AskUserQuestion`: "Asked" / "Ask", with a question icon.
3. New `QuestionCard.tsx`. `ChatThread` renders it instead of `ApprovalCard` when the step's tool is `AskUserQuestion`. Layout: one block per question (header chip, question text, option buttons with the description below the label, then an "Other…" button that reveals a text field). For a single-select question, "Other…" is one more choice: picking it clears the option, and picking an option hides the field. For a multi-select question, "Other…" toggles like an option. The footer has "Skip" and "Send", with "Send" as the primary button on the right, as on the approval card.
4. An answered question step becomes a turn segment of its own (`kind: "question"`), in its place in the turn. It shows each question in muted text with the answer under it, behind a left rule. While it is the turn's last segment, the turn shows "Thinking". An open or skipped question stays in its tool group: the row reads "Ask <question>" with "Skipped", or "Waiting for an answer".
5. `ToolGroup` says "Waiting for your answer" when the pending card is a question.

### 5. System prompt (`internal/chat/args.go`)

1. Both Claude access lines add: "To ask the user to choose between options, use AskUserQuestion."

### 6. Tests (written first)

- Go: profile tools and allow list; `claudePermissionResponse` for answer and skip; `Decide` validation; the runner records answers on the resolved event; the server accepts the answer body.
- TS: input parsing, answer string building, reducer stores answers, `QuestionCard` (Send disabled until answered, multi-select, free text, Skip), and `ChatThread` renders a question card for AskUserQuestion.

## Decisions

- **Reuse the approval pipeline, not a new one.** The call already arrives as `can_use_tool`, and cards, the waiting state, Stop and expiry all work for it today. A question is an approval card with a different body and different answers.
- **No "Allow for this chat".** A grant would let Claude Code run the tool without asking us, and there would be no answer.
- **Skip is a denial.** The agent gets a clear "skipped" message and can go on or ask in prose.
- **The user cannot send a normal message instead.** A running turn already blocks the composer (`ErrBusy`). The free-text field, Skip and Stop cover that need.
- **Answer string format follows Claude Code's own UI**: labels joined with ", ". Free text goes last.
- **An option's description is hidden when it repeats the label.** Agents often send `{label: "Cat", description: "Cat"}`.
- **Enter in the free-text field sends**, once every question has an answer.

## Edge Cases

- MEDIUM: A question without options. Show only the free-text field.
- MEDIUM: Two questions with the same text. Answers are keyed by question text in Claude Code, so the second overwrites the first. We accept this: it is Claude Code's format.
- LOW: The server restarts while a card is open. The existing expiry path marks it expired.
- LOW: An older claude without `AskUserQuestion`. claude drops unknown names from `--tools`.

## Resolved Questions (2026-10-14)

1. Codex: Claude-only now. Follow-up: xpo-eafa9b.
2. The free-text field shows only after the user clicks "Other…".
