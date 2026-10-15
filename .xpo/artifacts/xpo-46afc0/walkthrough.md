# Walkthrough: Question card shows one question at a time

## What changed

The AskUserQuestion card from xpo-447687 stacked every question in one column. It is now a small wizard. With 2 or more questions, the card shows one question at a time behind tabs, then a review step, and "Submit answers" sends from the review step. With one question, the card shows that question and "Submit answers", with no tabs and no review step.

The design went through seven review rounds with the user, two of them after user testing. The spec's Why and Decisions sections record each round. The short version:

- "Skip" next to "Next" read as "skip this question", but it declined all questions. It is now "Chat about this", on the far left, and the agent is told to wait for the user.
- People reached for "Chat about this" to go back. So Previous is gone, the tabs are the way back, and the right side of the footer only moves forward.
- Clickable review rows looked like options, and people went round in circles. The review rows are now plain results with a ✓ or a ?, and only a hover "Change" button goes back.

## How the pieces fit

### Steps are indexes

`current` indexes the steps: `0 … n-1` are the questions, and `n` is the review step. `reviewStep(questions)` returns `n` with 2 or more questions, and null with one. All navigation rules are pure functions in `chat-thread-utils.ts`:

- `questionAdvance` decides what the primary button and Enter do. Before the last step, they go to the next step. On the last step (review, or the only question), they send when every question has an answer. Otherwise they go to the first open question. Next never sends.
- `autoAdvanceTo` gives the step after a single-select pick. A single question returns null, so nothing sends on its own.
- `questionKeyAction` maps ←/→ to steps and 1–9 to options. It returns null on the review step for 1–9.

The card (`QuestionCard.tsx`) keeps only state and wiring. `go(i)` is the one way to change step. It clears the auto-advance timer, sets the direction for the slide, and moves focus to the card root, so the keys keep working.

### Why pure helpers

The web tests have no DOM environment. `renderMarkup` renders static HTML, so tests cannot click. The navigation rules are therefore pure functions with direct tests. The card's parts (`Choice`, `QuestionTabs`, `QuestionReview`, `QuestionFooter`) are exported, so tests can render each state by its props.

### The option row

`Choice` is a bordered `div` around the `role="radio"` (or `checkbox`) button, not a button. An `<input>` cannot sit inside a `<button>`, and the "Other…" field must sit inside its row. The field is the `children` of the "Other…" `Choice`, indented to the label (`pl-8.5`: the row padding, the mark and the gap).

### Visual layers

The card uses three existing surface tokens: recessed bands (`surface-ai-1`) for the tabs and the footer, the card (`surface-ai-2`) for the body, and raised rows (`surface-ai-3`) for options. A selected option uses one new token, `--color-surface-ai-selected` (accent at 14%), so the tint can be changed in one place. The body and the footer both use 12 px of horizontal padding, so the footer buttons, the option rows and the review rows share one edge.

### "Chat about this"

This is the card's deny. `internal/chat/approval.go` now sends: "The user wants to talk about these questions in the chat instead of answering them. End your turn and wait for their message." Without "end your turn", the agent can guess and go on, and the user cannot type while the turn runs. `approvalOutcomeLabel` shows "Chat instead" for a declined question. Older transcripts that say "Skipped" now also read "Chat instead".

### Motion

Three animations form one moment: the pick lands (the dot pops in and the tint fades in), the tab's ✓ pops in, and the next step slides in from its side. The pop fills the 250 ms auto-advance pause, so the pause reads as a confirmation.

- The keyframes are in `index.css`. The classes are `@utility` blocks, because Tailwind's `motion-safe:` variant only applies to utilities. With reduced motion, the classes do nothing. The color fades stay, because they are not motion.
- The animations play on mount. The question panel is keyed by `current`, so each step mounts fresh. `stepInClass(null)` returns nothing, so the card does not animate when it opens.
- `stepInClass` lives in `chat-thread-utils.ts`, because `react-refresh/only-export-components` rejects non-component exports from the card file.
- The user's system has reduced motion on. So the gate went in last, after the user approved the motion.

## Things to know

- Revisiting an answered question pops its mark again, because the step remounts. This is accepted: it shows the answer as the step arrives.
- The keys work only while focus is in the card. The handler stops the event, so the board's own ←/→ listeners do not also fire.
- "Change" is hidden until hover, but it also shows on keyboard focus (`focus-visible`).
- Codex question cards are still to come (xpo-eafa9b).

## Acceptance criteria

### Steps and tabs

- [x] One step per question, then a review step; one step at a time; the first question opens. Evidence: "shows a tab per question, a last Review tab, and only the first question"; `reviewStep` tests.
- [x] Tab label is the header, or Q1, Q2, …. Evidence: `questionTabLabel` test.
- [x] The Review tab sits at the far right. Evidence: "pushes the Review tab to the far right".
- [x] ✓ only on answered question tabs; none on Review. Evidence: "shows a check only on answered tabs, and no other mark".
- [x] The current tab has primary text, with no other mark. Evidence: the same test; the user's review in the board.
- [x] A click on a tab makes that step current. Evidence: `go` wiring; the user's review in the board.
- [x] One question: no tabs, no review step. Evidence: "offers Chat about this and Submit answers … and no review step".

### Review step

- [x] No heading; full question over the answer. Evidence: "lists each full question over its answer, or Not answered, and no heading".
- [x] The question is secondary and normal weight; the answer is primary after an accent ✓. Evidence: "marks an answer with the answered check …"; the user's review in the board.
- [x] "Not answered" is muted italic after a muted ?, aligned with the answers. Evidence: the same test (`lucide-circle-question-mark`).
- [x] Rows have no background, no border, no option mark, and are not clickable. Evidence: "makes only Change clickable …"; no radio or checkbox roles.
- [x] Rows sit inside the body's padding. Evidence: "keeps the rows' padding inside the body …".
- [x] Hover shows a background and "Change"; "Change" goes to the question and shows on focus. Evidence: "makes only Change clickable, one per question, named for its question"; the user's review in the board.

### Footer

- [x] "Chat about this" at the far left in the secondary style; no Previous. Evidence: "puts Chat about this first …"; "shows Chat about this as a secondary button".
- [x] The primary button reads Next, or "Submit answers" on review and with one question. Evidence: card and footer tests.
- [x] The footer has 12 px padding like the body. Evidence: "pads like the body …".
- [x] Next goes to the next step, to review from the last question, is never gated, and never sends. Evidence: `questionAdvance` tests.
- [x] "Submit answers" is enabled only with every answer, with the same payload. Evidence: footer test; `questionAnswers` tests.
- [x] The error sits between the two buttons. Evidence: footer order test.
- [x] "Chat about this" sends the new deny message. Evidence: `TestClaudePermissionResponse_ChatAboutQuestion`, `TestRunner_QuestionChatAboutThis`.
- [x] A declined question reads "Chat instead". Evidence: `approvalOutcomeLabel` test.
- [x] Everything is disabled while sending. Evidence: the disabled tests for tabs, "Change" and footer.

### Auto-advance and keys

- [x] A single-select pick moves on after 250 ms, to review from the last question, also on a changed answer. Evidence: `autoAdvanceTo` tests; the user's review in the board.
- [x] With one question, nothing moves or sends on its own. Evidence: `autoAdvanceTo(one, 0)` is null; `questionAdvance` one-question test.
- [x] "Other…" and multi-select do not auto-advance. Evidence: `autoAdvanceTo` multi-select test; the "Other…" handler clears the timer.
- [x] Enter acts as the primary button, and jumps to the first open question on the last step. Evidence: `questionAdvance` tests.
- [x] ←/→ move between steps; 1–9 pick options, and do nothing on review. Evidence: "maps arrow and number keys to a step or an option".

### Question layout

- [x] With tabs, no header chip. Evidence: "leaves the header to the tab …".
- [x] With one question, the chip sits at the right end of the question line. Evidence: "shows the question with its header at the end of the line …".

### Visual hierarchy

- [x] Recessed tab and footer bands with dividers and 12 px vertical padding. Evidence: `py-3` tests for tabs and footer; the user's review in the board.
- [x] Raised option rows, centered contents, accent border and tint on pick. Evidence: "centers the mark, the label and the hint"; the user's review in the board.
- [x] Number hints 1–9, none on "Other…". Evidence: "shows each option's key as a number hint, and none on Other".
- [x] The "Other…" field sits inside its row; alone for a question without options. Evidence: "holds its children inside the row, after the button"; "shows the field at once for a question without options".

### Motion

- [x] The step slides in from its direction; not on open. Evidence: `stepInClass` test; "does not slide the first step in when the card opens"; the user's review in the board.
- [x] The pick lands and the tab ✓ pops. Evidence: "pops the mark in when the option is checked"; tab ✓ test.
- [x] 150 ms, slight overshoot, no height animation. Evidence: `index.css`; the user's review in the board.
- [x] Every animation is gated for reduced motion. Evidence: the tests expect `motion-safe:` classes. The built CSS wraps them in `@media (prefers-reduced-motion:no-preference)`.

### Build

- [x] `make test` passes: lint, 1030 frontend tests and the Go tests. `make frontend` builds.
