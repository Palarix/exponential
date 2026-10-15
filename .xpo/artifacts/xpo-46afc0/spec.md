# Spec: Question card shows one question at a time

## What

When AskUserQuestion sends more than one question, the question card shows one question at a time, then a review step. A tab strip at the top lists the questions and, at its far right, a "Review" tab, and marks the questions with an answer. The footer has "Chat about this" on the far left and one primary button on the right: "Next" on a question, "Submit answers" on the review step. With one question, the card has no tabs and no review step: its primary button is "Submit answers", and its header chip sits at the right end of the question line. "Chat about this" declines the questions and tells the agent to wait for the user's message. The card has a clear visual hierarchy: a recessed tab band, a body with raised option rows, and a recessed footer band. Short animations connect a pick to the next step: the pick lands, the tab's ✓ pops in, and the next step slides in from its direction.

## Why

The card from xpo-447687 stacks all questions (up to 4, each with options and descriptions) in one column. With several questions, the card gets tall, pushes the thread up, and is hard to scan. A wizard keeps the card short and its height steady, and the tabs still show the whole set at a glance.

In review, the user pressed "Skip" next to "Next" to skip one question, and the card declined all questions. "Chat about this" is the label other agents use for this action, and it says what happens: the questions close and the user talks to the agent instead.

Also in review, the user found the card too plain: the tabs, the choices and the buttons sat on one surface, with no separation. After the first restyle, the user found the tabs too loud, and asked for centered option rows and an "Other…" field inside its row.

In user testing, people moved the mouse to "Chat about this" to go back, and found Previous only after that. So "Chat about this" moves to the far left, Previous goes, and the tabs are the way back. A review step at the end lets the user check all answers before they send them.

In the next round of user testing, people did not notice that they were on the review step. The review rows looked like option rows, so people clicked them, went back to a question, and came round to review again, in circles. So the review rows are now plain text, and only a "Change" button on hover goes back.

The user then found the review step's hierarchy odd: a heading, small muted questions, and larger answers competed. A two-column list of tab labels and answers did not work either. The review step now shows the full question and answer, in the card's design language, but marked as read-only: an answer has the ✓ of an answered tab, an open question has a question mark, and neither has an option mark or border. The Review tab moves to the far right, so it reads as a separate step.

Last, the user asked for animation. Without it, a step change is a hard cut, and the 250 ms auto-advance pause reads as lag.

## Acceptance Criteria

### Steps and tabs

- [x] With 2 or more questions, the card has one step per question and a last review step. It shows one step at a time. The first question is current when the card opens.
- [x] A tab strip above the step shows one tab per question and a last tab, "Review". A question tab's label is the question's `header`, or "Q1", "Q2", … when the header is empty.
- [x] The Review tab sits at the far right of the tab strip, apart from the question tabs.
- [x] A question tab shows a ✓ only when its question has an answer. The Review tab has no mark.
- [x] The current tab has primary (white) text. The other tabs have muted text. There is no underline and no other current-tab mark.
- [x] A click on a tab makes that step current.
- [x] With one question, the card has no tab strip and no review step.

### Review step

- [x] The review step has no heading. It shows one row per question: the full question text, with the answer under it.
- [x] The question text is normal weight, in secondary text color, at the same size as the answer.
- [x] The answer is in primary text color, after an accent ✓, the same check as an answered tab.
- [x] A question without an answer shows "Not answered" in muted italic text, after a muted question-mark icon in the slot of the ✓. It lines up with the answer text of the other rows.
- [x] A review row has no background, no border, no radio or checkbox mark, and is not clickable.
- [x] A review row keeps its inner padding inside the body's 12 px. Its edges line up with the option rows and the footer buttons, and its text lines up with the text in the footer buttons.
- [x] On hover, a review row shows a background and a "Change" button at its right end. "Change" makes that question current.
- [x] "Change" also shows when it has keyboard focus, so Tab reaches it.

### Footer

- [x] "Chat about this" is at the far left of the footer, on every step. There is no Previous button.
- [x] "Chat about this" uses the secondary button style: a border and primary text, as "Allow for this chat" on the approval card.
- [x] The primary button is at the right end. It reads "Next" on a question step, and "Submit answers" on the review step. With one question, it reads "Submit answers".
- [x] The footer has the same horizontal padding as the body (12 px), so its buttons line up with the edges of the option rows.
- [x] Next moves to the next step. On the last question, Next moves to the review step. Next is enabled even when the current question has no answer. Next never sends.
- [x] "Submit answers" is enabled only when every question has an answer. It sends the same payload as before.
- [x] An error shows between "Chat about this" and the primary button.
- [x] "Chat about this" denies the call. The agent gets the message: "The user wants to talk about these questions in the chat instead of answering them. End your turn and wait for their message."
- [x] A question declined with "Chat about this" reads "Chat instead" in the transcript, not "Skipped".
- [x] While the card sends, the tabs, the "Change" buttons, both footer buttons and the options are disabled.

### Auto-advance and keys

- [x] On a single-select question, a click on an option moves to the next step after 250 ms. From the last question, the next step is the review step. This also applies when the user changes an earlier answer.
- [x] With one question, a pick does not move anywhere, and nothing sends on its own.
- [x] A click on "Other…", or on a multi-select option, does not move to the next step.
- [x] Enter in the free-text field acts as the primary button. On the review step, or with one question, Enter with open questions moves to the first open question.
- [x] With focus in the card but not in the free-text field, ← and → move to the previous and next step. → on the last question moves to the review step.
- [x] With focus in the card but not in the free-text field, the keys 1–9 pick the option with that number on the current question. They do nothing on the review step.

### Question layout

- [x] With tabs, the card shows no header chip on the question line.
- [x] With one question, the header chip sits at the right end of the question line. The question text aligns left with the options.

### Visual hierarchy

- [x] The tab strip sits in a recessed band (`surface-ai-1`) with a divider below it.
- [x] The footer sits in a recessed band (`surface-ai-1`) with a divider above it.
- [x] The tab band and the footer band have 12 px of vertical padding. In the tab band, the padding is on the tabs, so the whole band height is clickable.
- [x] The body stays on the card surface (`surface-ai-2`). The question is a semibold line with space below it.
- [x] Each option is a raised row (`surface-ai-3`) with a border.
- [x] In an option row, the radio or checkbox mark, the label with its description, and the number hint are vertically centered on each other.
- [x] A selected option has an accent border and an accent tint. The tint comes from one token, `--color-surface-ai-selected`.
- [x] Each of the first nine options shows its number at the right end of its row, as a muted key hint. "Other…" has no number.
- [x] When "Other…" is open, its free-text field sits inside the "Other…" row, below the label and aligned with it. The row's selected border and tint include the field.
- [x] A question without options shows the free-text field on its own, with no "Other…" row.

### Motion

- [x] Step slide: when the current step changes, the new step fades in and slides 8 px. It comes from the right when the step index goes up, and from the left when it goes down. This applies to Next, auto-advance, tabs, "Change" and the arrow keys.
- [x] The first step does not animate when the card opens.
- [x] Pick lands: when an option becomes checked, its radio dot or checkbox tick pops in (scale from 60%), and the row's border and tint fade in.
- [x] Tab ✓: when a question gets an answer, its tab's ✓ pops in the same way.
- [x] The step slide and the pops last `--duration-normal` (150 ms). The pop uses a slight overshoot.
- [x] The card does not animate its height.
- [x] Every animation is gated with `motion-safe:`, so a user with reduced motion gets no animation. The color fades stay: they are not motion.

- [x] `make test` passes.

## Flow

### 1. Helpers (`web/src/components/Agent/chat-thread-utils.ts`)

1. Add `questionAnswered(question, pick): boolean`. It returns true when the pick gives a non-empty answer string.
2. Export the answer-string function as `questionAnswerText(question, pick)`. `questionAnswers` and `questionAnswered` use it.
3. Add `questionTabLabel(question, index): string`. It returns the header, or `Q<index+1>` when the header is empty.
4. Add `reviewStep(questions): number | null`. It returns `questions.length` when there are 2 or more questions, and null otherwise.
5. Add `questionAdvance(questions, picks, current)`. Before the last step, it returns `go` to the next step. On the last step (the review step, or the only question), it returns `send` when every question has an answer, and otherwise `go` to the first open question.
6. Add `autoAdvanceTo(questions, current)`. After a single-select pick, it returns the next step, which is the review step after the last question. It returns null for a multi-select question and for a card without a review step.
7. Add `questionKeyAction(key, questions, current)`. It maps ←/→ to `go` within the steps, and 1–9 to `pick` on a question step. It returns null for other keys.
8. Change `approvalOutcomeLabel` to return "Chat instead" for a denied question card.
9. Add the type `StepDirection` (`"forward" | "back" | null`) and `stepInClass(direction)`. It returns the gated step-in class for the direction, or an empty string for null.

### 2. Tokens and keyframes (`web/src/index.css`)

1. Add `--color-surface-ai-selected: color-mix(in srgb, var(--color-accent-primary) 14%, transparent)` next to the other AI surface tokens.
2. In the ANIMATIONS section, add the keyframes `step-in-forward` (from `translateX(8px)`, transparent) and `step-in-back` (from `translateX(-8px)`, transparent).
3. Add the keyframes `pop-in` (from `scale(0.6)`, transparent).
4. Add `animate-step-in-forward`, `animate-step-in-back` (`--duration-normal`, `ease-out`) and `animate-pop-in` (`--duration-normal`, with an overshoot curve) as `@utility` blocks. Plain classes do not take Tailwind's `motion-safe:` variant.

### 3. Card (`web/src/components/Agent/QuestionCard.tsx`)

1. Add `current` state, starting at 0. It indexes the steps: the questions, then the review step.
2. Add the exported `QuestionTabs` component. Render it when there is a review step, in the recessed band. Use `role="tablist"`, with one `role="tab"` button per question, a last "Review" tab with `ml-auto`, and `aria-selected` on the current one. Give each tab `py-3`. Show a ✓ only on answered question tabs, with `motion-safe:animate-pop-in`. Mark the current tab with primary text only.
3. On a question step, render only `questions[current]` in the body. On the review step, render the exported `QuestionReview` component, with `aria-label="Review your answers"` on its panel. Both use `role="tabpanel"` when there are tabs.
4. `QuestionReview`: one plain row per question. A row shows the question text (`text-sm`, secondary, normal weight), then the answer from `questionAnswerText` (`text-sm`, primary) after an accent `Check`, or "Not answered" (muted, italic) after a muted `CircleQuestionMark` in the same slot. Each row is a `group` with `px-2.5` and a hover background, inside the body's padding (no negative margin). Its "Change" button is hidden until the row is hovered or the button has focus. "Change" calls `onSelect(index)`, and its `aria-label` names the question.
5. On the question line, put the question text first and the header chip after it, pushed to the right end. Render the chip only when there are no tabs.
6. Export `Choice`. Make it a bordered container `div` that holds the `role="radio"`/`role="checkbox"` button and optional `children` below it. Center the mark, the label block and the hint in the button with `items-center`. Give the checked dot or tick `motion-safe:animate-pop-in`, and the container a `--duration-normal` color transition.
7. Give the first nine options a `kbd` number hint at the right end.
8. Pass the free-text field as the children of the "Other…" `Choice`, indented to the label. For a question without options, render the field alone.
9. Change the exported `QuestionFooter` component, in the recessed band with `px-3 py-3`: "Chat about this" at the far left in the secondary style, then the error, then the primary button at the right end. Remove Previous.
10. Clip the card root (`overflow-hidden`), so the bands follow its rounded corners and the sliding step does not show outside the card.
11. Add `direction` state (`StepDirection`, starting at null). `go(i)` sets it from `i` and `current`, and does nothing when `i` is the current step. Give the step panels the class from `stepInClass(direction)`. The question panel is keyed by `current`, so each step mounts fresh and its slide plays.
12. Make the primary button and Enter in the free-text field call `advance()`, which runs `questionAdvance`.
13. After a single-select pick, start a 250 ms timer that moves to `autoAdvanceTo`. Clear the timer on any other navigation, on "Other…", on send, on unmount, and on a new pick.
14. Add an `onKeyDown` handler on the card root that runs `questionKeyAction`. Ignore the keys in the free-text field, with a modifier key, and while the card sends. Stop a handled key from reaching the board's document listeners.
15. Make the card root focusable (`tabIndex={-1}`). After a navigation by tab, "Change", Next, a key or auto-advance, move focus to the card root.
16. Prefix every animation class with `motion-safe:`. This step came last, after the user approved the motion.

### 4. Deny message (`internal/chat/approval.go`)

1. In `claudePermissionResponse`, change the deny message for `AskUserQuestion` to the "Chat about this" message.

### 5. Tests (written first)

The web tests have no DOM environment: components render to static markup. So the navigation logic lives in pure helpers, and the tests cover it there. Colors, spacing and motion are checked by eye in the board. The tests check that the gated animation classes are in place.

- `internal/chat/approval_test.go`: a denied question card returns the new message.
- `chat-thread-utils.test.ts`: `questionAnswered`, `questionAnswerText`, `questionTabLabel`, `reviewStep`, `questionAdvance` (Next never sends, the last question goes to review, review sends, jump to the first open question, one question), `autoAdvanceTo` (to review from the last question, null for multi-select and one question), `questionKeyAction` (review bounds, no picks on review), and "Chat instead" from `approvalOutcomeLabel`.
- `QuestionCard.test.tsx`:
  - One question: the question text comes before the header chip. No tabs, no Previous, "Chat about this" before "Submit answers", and "Submit answers" disabled until answered.
  - Several questions: a tab per question and a last "Review" tab, only the first question, no header chip, "Chat about this" before Next, no Previous, and a focusable root. The first step has no step-in class.
  - Number hints: 1 and 2 on the options, none on "Other…".
  - `Choice`: the children render inside the row container, after the button. A checked mark has `motion-safe:animate-pop-in`.
  - `QuestionTabs`: a ✓ only on answered question tabs, with `motion-safe:animate-pop-in`, no mark on the others or on Review, `ml-auto` on Review only, `title` for long headers, `py-3` on the tabs, and disabled while sending.
  - `stepInClass`: the gated forward and back classes, and nothing for null.
  - `QuestionReview`: no heading. Each row shows the full question, then the answer, or "Not answered" for an open question. A ✓ before an answer, and a question mark before "Not answered". No radio or checkbox roles. The only buttons are one "Change" per question, labelled with the question, and disabled while sending. No negative margin, and `px-2.5` on the rows.
  - `QuestionFooter`: the order "Chat about this", error, primary. No Previous. `px-3 py-3` padding. "Chat about this" has a border (secondary style). "Submit answers" disabled until answered. Both buttons disabled while sending.

## Decisions

- **Tabs are the way back; no Previous.** The user chose this after user testing. People reached for "Chat about this" to go back. The right side of the footer now only moves forward, and the tabs move anywhere.
- **"Chat about this" at the far left.** The user chose this. It is the exit, away from the forward button.
- **"Chat about this" is a secondary button.** The user chose this. It is a real choice, not a quiet escape, so it gets a border. It reuses the approval card's secondary style.
- **A review step, as the last tab.** The user chose this. The user checks all answers in one place before sending. As a tab, it is reachable at any time.
- **The Review tab sits at the far right.** The user chose this. Apart from the question tabs, it reads as the destination, not as one more question.
- **Review rows are plain text, with "Change" on hover.** The user chose this after user testing. Raised, clickable rows looked like options, and people went round in circles. Plain rows read as a summary, and "Change" is a deliberate way back.
- **Review rows: full question over a checked answer, with no heading.** The user chose this, after a two-column version (tab label and answer) did not work. The heading repeated the Review tab and the "Submit answers" button. The question takes a quieter style than on its own step (normal weight, secondary), and the answer takes the primary color, so the answer is what the eye reads. The ✓ is the card's mark for "answered" (as on the tabs), not an option mark. So the row reads as a result to check, not as a choice to make.
- **A question mark for an open question.** The user chose this. Every review row then has a status mark in the same slot: ✓ for answered, ? for open. The question mark is muted, so the answered rows stay the strongest.
- **Review rows keep their padding inside the body.** The user chose this. A negative margin put the row text in line with the heading, but it broke the line with the option rows and the footer buttons.
- **"Change" also shows on keyboard focus.** A hover-only button cannot be found with the keyboard.
- **No review step for one question.** The user chose this. A review of one answer the user just picked adds a click and no value.
- **"Submit answers", not "Send".** The user chose this. It names what the button sends.
- **Keep "Next", not "Skip question".** The user chose this. Every question needs an answer before Submit, so a skip button would promise a skip that Submit then blocks. Review shows "Not answered" for a question the user moved past.
- **Next is not gated on an answer, and Next never sends.** The user can look ahead and come back. Only "Submit answers" sends.
- **Auto-advance from the last question goes to review.** Review sends nothing, so the move is safe. With one question, nothing auto-advances, and nothing sends on its own.
- **Auto-advance goes to the next step, not the next unanswered one.** That order is predictable when the user goes back to change an answer.
- **Auto-advance also fires on a changed answer.** The user chose this. The forward flow is the same every time.
- **Keyboard shortcuts are part of this issue.** The user chose this. Keys work only while focus is in the card, so they do not fight the composer and the rest of the board.
- **The header chip moves into the tab.** With tabs, the chip repeats the tab label, so the card hides it. The user confirmed this.
- **With one question, the header chip is metadata at the right end of the question line.** The user chose this. The question text then aligns left with the options and their descriptions.
- **"Chat about this", not "Skip" or "Skip all".** The user chose this. "Skip" next to "Next" reads as "skip this question". "Chat about this" is the convention in other agents and names the outcome.
- **The deny message tells the agent to end its turn.** Without that, the agent can guess an answer and go on, and the user cannot type while the turn runs. The message is a Go change, outside "presentation only". The user approved it, because the new label is wrong without it.
- **Three surface levels, from the existing AI tokens.** Recessed bands (`ai-1`) frame the card, the body is the card (`ai-2`), and options are raised (`ai-3`). No new surface colors.
- **The footer padding matches the body.** The user chose this. Both use 12 px horizontally, so the footer buttons line up with the option rows.
- **12 px vertical padding on both bands.** The user chose this, to try. The tab band puts it on the tabs, not on the band, so a click anywhere in the band's height hits a tab.
- **Quiet tabs: text color marks the current tab, a ✓ marks an answered one.** The user chose this. The first restyle had an accent underline and ○/● marks, and they were loud and repeated what the text color already said.
- **The option row is a container around its button.** An `<input>` cannot sit inside a `<button>`. A container lets the "Other…" field share the row's border and tint.
- **Accent tint for the selected option.** The user chose this, to try first. The tint lives in one token, so it can be calmed down in one place.
- **Number hints on the options.** The user chose this. They make the 1–9 keys discoverable.
- **Three animations that form one moment: pick lands, ✓ pops, step slides.** The user chose these from five options. The pick's pop fills the 250 ms auto-advance pause, so the pause reads as a confirmation, not lag. The slide direction gives the steps a left-to-right order. The review row stagger and a card entrance were left out.
- **CSS keyframes, no animation library.** The app already has a small set of keyframes and duration tokens. Mount-time CSS animations need no state beyond the step direction.
- **The new animations are `@utility` blocks, not plain classes.** Only utilities take Tailwind's `motion-safe:` variant. The older `animate-fade-in` and friends stay plain classes; this issue does not change them.
- **`stepInClass` lives in `chat-thread-utils.ts`, not in the card file.** The `react-refresh/only-export-components` lint rule allows only components as exports of a component file.
- **No height animation.** The card can change height between steps. Animating height is where step transitions get janky.
- **The reduced-motion gate came last.** The user's system has reduced motion on, so gated animations would not play for their review. The `motion-safe:` gate went in after the user approved the motion, before the commit.
- **No change to pick state or the payload.** `picks` already holds every question, so switching steps keeps all answers.

## Edge Cases

- MEDIUM: Long headers. Claude Code caps a header at 12 characters, but another agent might not. Truncate a tab label with an ellipsis and show the full header in `title`.
- MEDIUM: The user presses "Submit answers" while it is disabled. Nothing happens. Enter on the review step moves to the first open question instead.
- MEDIUM: Touch screens have no hover. A tap on a review row gives it hover in most mobile browsers, which shows "Change". The tabs also stay as a way back.
- MEDIUM: Revisiting a question remounts its step, so its checked mark pops again. This is acceptable: it shows the existing answer as the step arrives.
- LOW: An older transcript has a question declined with "Skip". It now reads "Chat instead". Both mean the user declined the questions, so the change is acceptable.
- LOW: A question that wraps to several lines. The chip stays at the top right, next to the first line.
- LOW: A long question or answer wraps to several lines in a review row. "Change" stays centered on the row.
- LOW: A long option description wraps to several lines. The mark and the hint stay centered on the whole label block.
- LOW: More than 9 options. The keys 1–9 reach only the first nine, and only those show a number. Mouse and Tab still reach the others.
- LOW: The user clicks a tab during the 250 ms pause. The click clears the timer, and the tab wins.
- LOW: The user clicks the current tab. The step does not change, and nothing animates.
- LOW: An unparseable input gives zero questions. No body, no tabs, and "Submit answers" disabled.

## Assumptions

- The 250 ms delay is long enough to see the pick land and short enough not to feel slow.
- A 14% accent tint is visible on `surface-ai-3` without being loud. The user approved it in the board.
- 150 ms and 8 px are enough to read as motion without slowing the flow. The user approved it in the board.
- The answered transcript segment (each question with its answer) stays as it is. Only the open card and the declined label change.

## Resolved Questions (2026-10-15)

1. Auto-advance on a changed answer: yes.
2. Keyboard shortcuts (←/→, 1–9): in this issue.
3. Header chip: hidden with tabs; at the right end of the question line with one question.
4. "Skip" becomes "Chat about this", with a deny message that asks the agent to wait, and "Chat instead" in the transcript.
5. Visual hierarchy: recessed bands, raised options, number hints, and an accent tint on the selected option.
6. Tabs: no underline, no ○/●; current tab by text color, ✓ only on answered tabs. Option rows: centered. "Other…" field inside its row.
7. Footer: "Chat about this" far left, no Previous, "Next" kept. Review step as the last tab, with "Submit answers". No review step for one question.
8. Footer padding: 12 px, as the body. Review rows: plain, with a hover background and a "Change" button.
9. Bands: 12 px vertical padding. "Chat about this": secondary style.
10. Review rows: padding kept inside the body, no negative margin.
11. Review tab at the far right. No heading on the review step.
12. Review rows: the two-column version is out. Full question (secondary, normal weight) over the answer (primary, after an accent ✓).
13. Open questions on the review step: a muted question-mark icon in the ✓ slot.
14. Motion: step slide by direction, pick lands, tab ✓ pops.
15. The reduced-motion gate: added after the user's approval, through `@utility` blocks and `motion-safe:`.
