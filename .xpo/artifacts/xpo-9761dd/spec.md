# Agent chat: question card takes focus when it appears

## What

The active pending card takes keyboard focus, whether it is an approval card or a question card. A question card's keys (1–9, ←/→) work as soon as it appears, without a click.

## Why

xpo-aaf8b1 gave approval cards focus-on-arrival and ruled that "question cards never become active". Its spec gives no reason for that rule. The question card already has a full keyboard model (xpo-46afc0), but the user has to click into the card before any key works.

## How

The question card joins the same "active card" model as approval cards, so there is one rule for which card holds focus.

1. In `chat-thread-utils.ts`, rename `activeApprovalId` to `activeCardId`. It counts every pending card, question cards included: the focused card, else the oldest.
2. Move `typing(el)` from `ApprovalCard.tsx` to `chat-thread-utils.ts` and export it, so both cards share it.
3. `QuestionCard` takes `active` and `onFocusChange`, as `ApprovalCard` does. When `active` becomes true, it focuses `rootRef` with `preventScroll`, unless `typing(document.activeElement)`.
4. `QuestionCard` reports focus in and out (`onFocus`, and `onBlur` when focus leaves the card), so a focused question card stays the active one.
5. In `ChatThread.tsx`, pass `active` and `onFocusChange` to `QuestionCard`.
6. The question card shows the focus border while focus is in it (`focus-within:border-[var(--color-border-focus)]`), as the approval card does.
7. `questionKeyAction` takes whether the key landed on the card itself. Enter there returns `{ kind: "primary" }`, and the card runs `advance()`: Next on a question, "Submit answers" on the review step or a single question. A focused button or field keeps its own Enter. Enter does nothing while the primary button is disabled.

## Acceptance Criteria

- [ ] A question card that appears with no other pending card takes focus. Pressing 1–9 picks an option with no click first.
- [ ] The question card does not take focus from a text field with text in it.
- [ ] With several pending cards, the focused card stays active, else the oldest takes focus. Question cards count like approval cards.
- [ ] After the question card is answered and it was the last card, the composer takes focus (existing `cardsDone` path).
- [ ] The question card shows the focus border while focus is in it.
- [ ] Focusing does not scroll the thread.
- [ ] With focus on the card itself, Enter presses the primary button: Next on a question, "Submit answers" on the review step or a single question. It does nothing while that button is disabled.
- [ ] With focus on a button or the free-text field, Enter keeps its own behavior.
- [ ] `activeCardId` tests cover question cards; the "skips question cards" test is replaced.
- [ ] `make test` passes.

## Decisions

- **One active-card rule for both kinds.** Simpler than a separate focus path for questions, and it keeps two cards from fighting over focus.
- **Focus the card, not the first option or the text field.** The card's keys then work. A question without options still needs a Tab or click into its field; autofocusing that field could swallow keys meant for elsewhere.
- **Focus border on the question card.** Without it, the user cannot see that the card holds focus.
- **Enter presses the primary button (from review).** It matches the approval card, where Enter on the card presses "Allow once". Found in tophat: the review step had no key to submit.

## Edge Cases

- LOW: An approval card and a question card are pending at once. The oldest takes focus; the user can Tab or click to the other.
- LOW: The user is typing in the composer when the question arrives. The card does not take focus.
