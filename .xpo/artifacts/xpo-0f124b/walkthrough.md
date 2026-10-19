# Walkthrough: approval card says "Allow", and sending does not shift the layout

## What changed

Two polish items in `web/src/components/Agent/ApprovalCard.tsx`.

**Label.** The primary button reads "Allow" instead of "Allow once". This reverses the "Allow once, not Allow" decision in xpo-aaf8b1. The user found the long label crowded the collapsed row, and the chevron menu ("Allow for this chat") already sets the two grants apart. The outcome label after a decision, "Allowed once" (`approvalOutcomeLabel` in `chat-thread-utils.ts`), stays: the user likes it there.

**Stable layout while sending.** Before, sending a decision put a `<Spinner />` in front of the label. The button got wider, the row shifted, and in a collapsed card the buttons could wrap for a moment. Now both buttons render through a new `ButtonLabel({ text, sending })`:

- The wrapper is `relative inline-grid place-items-center`.
- The label is always rendered. While sending it gets `invisible`, so it still takes its space and the button keeps its size.
- While sending, the spinner sits centred over the label in an `absolute inset-0` overlay.

Allow shows the spinner for both `allow` and `allow_session`, as before. Only the label's visibility changes, so the buttons' padding, gap and disabled styles stay the same.

## Why `invisible` and not a fixed width

`invisible` (`visibility: hidden`) keeps the label's exact box, whatever the text and font. A fixed or minimum width would have to be tuned per label and would break when a label or the font changes.

## Tests

`ButtonLabel` is exported so the static-markup tests can render both states. The card itself only renders its idle state in static markup.

## Acceptance criteria

- [x] The primary button reads "Allow". Evidence: `ApprovalCard.test.tsx` "offers Deny and a split Allow button…" matches `>Allow<` and asserts that no "Allow once" remains. The user tophatted it on the demo board.
- [x] Sending a decision (Enter, Esc, or a click) does not change the size of any button or the card's layout. Evidence: the "lays the spinner over the hidden label while sending" test checks that the label stays rendered with `invisible` and the spinner sits in an `absolute inset-0` overlay. The user tophatted it on the demo board.
- [x] The spinner still shows on the button for the decision being sent. Evidence: the same test finds `animate-spin` in the overlay. Deny gets `sending === "deny"`, and Allow gets `allow` or `allow_session`.

`make test` passes, lint included.
