# Walkthrough: Question card takes focus when it appears

## What changed

Approval cards took keyboard focus on arrival (xpo-aaf8b1), but question cards did not. xpo-aaf8b1's rule "question cards never become active" kept them out, and its spec gave no reason. The question card already had keys (1–9, ←/→ from xpo-46afc0), but none worked until the user clicked into the card.

Now both kinds of card share one "active card" model, and the question card also answers Enter.

## How it fits together

- **`activeCardId`** (was `activeApprovalId`) in `chat-thread-utils.ts` picks the card that holds focus from all pending cards: the focused one, else the oldest. It no longer filters out questions.
- **`typing(el)`** moved from `ApprovalCard.tsx` to `chat-thread-utils.ts`, so both cards share the "never steal focus from a field with text" check.
- **`QuestionCard`** takes `active` and `onFocusChange`, the same props as `ApprovalCard`. When `active` turns true, a `useEffect` focuses `rootRef` with `preventScroll`, unless `typing(document.activeElement)`. `onFocus`/`onBlur` (blur only when focus leaves the card) report focus, so a focused question card stays active. The root gets `focus-within:border-[var(--color-border-focus)]`, the approval card's focus border.
- **`ChatThread`** builds one props object per pending card and passes it to either card kind. The turn's state is renamed `focusedCard`.
- **Enter:** `questionKeyAction` gets an `onCard` flag (default `false`). With focus on the card root, Enter returns `{ kind: "primary" }`, and the card runs `advance()`: Next on a question, "Submit answers" on the review step or a single question. `advance()` already does nothing when there are no answers, so Enter does nothing while Submit is disabled. A focused button or the free-text field keeps its own Enter.

## Decisions

- **One active-card rule for both kinds.** A separate focus path for questions could fight the approval card over focus.
- **Focus the card, not a field.** The card's keys then work. A text-only question still needs a Tab or click into its field. The user agreed to this in the spec review.
- **Enter presses the primary button everywhere** (added during tophat). The user found that the review step had no key to submit. Pressing the primary button on every step matches the approval card, where Enter on the card presses "Allow once". The user chose this over a "Submit only" Enter.

## Acceptance Criteria

- [x] A question card that appears with no other pending card takes focus. Pressing 1–9 picks an option with no click first. Evidence: tophat on the Vite dev server; the user confirmed "works".
- [x] The question card does not take focus from a text field with text in it. Evidence: the same `typing()` guard as the approval card; `QuestionCard.tsx` effect.
- [x] With several pending cards, the focused card stays active, else the oldest takes focus. Question cards count like approval cards. Evidence: `activeCardId` tests, "counts question cards like approval cards".
- [x] After the question card is answered and it was the last card, the composer takes focus. Evidence: the existing `cardsDone` path counts all pending cards; it was unchanged.
- [x] The question card shows the focus border while focus is in it. Evidence: QuestionCard test, "shows the approval card's focus border while focus is in it".
- [x] Focusing does not scroll the thread. Evidence: `focus({ preventScroll: true })`.
- [x] With focus on the card itself, Enter presses the primary button. It does nothing while that button is disabled. Evidence: `questionKeyAction` test, "presses the primary button on Enter…"; second tophat round confirmed by the user.
- [x] With focus on a button or the free-text field, Enter keeps its own behavior. Evidence: `onCard` is false off the root, and the same test covers it; the handler still returns early for inputs.
- [x] `activeCardId` tests cover question cards; the "skips question cards" test is replaced.
- [x] `make test` passes.

## Notes

- `typing()` has no unit test: Vitest runs without a DOM here, and the function checks `HTMLInputElement`/`HTMLTextAreaElement`.
